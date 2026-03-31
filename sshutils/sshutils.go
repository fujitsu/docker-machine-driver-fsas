package sshutils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"github.com/fujitsu/docker-machine-driver-fsas/models"

	"github.com/pkg/sftp"
	"github.com/rancher/machine/libmachine/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const (
	port                                    = 22
	cmdRebootCloudInit                      = "sudo cloud-init clean --logs --reboot"
	cmdGetStatusOS                          = "sudo -E SUSEConnect -s"
	cmdDeregisterOS                         = "sudo -E SUSEConnect -d"
	remoteScriptDir                         = "/tmp/fsas-nodedriver"
	SSH_CONNECT_ATTEMPT_DELAY time.Duration = 5 * time.Second
)

var (
	ErrNoneOfConstructorArgsCanBeEmpty = errors.New("none of the arguments can be empty; neither 'hostName', 'userName', 'sshPassword', 'sshKeyPath', 'hostPublicKey'")
	isInit                             = false
	IsPublicKeyIsValid                 = false
)

// SSHKeyParser defines the interface for parsing an SSH key from a path.
type SSHKeyParser interface {
	Parse(keyPath string) gossh.Signer
}

// fileSSHKeyParser is the default implementation that reads from the filesystem.
type fileSSHKeyParser struct{}

// NewFileSSHKeyParser creates a new parser that reads from the filesystem.
func NewFileSSHKeyParser() SSHKeyParser {
	return &fileSSHKeyParser{}
}

// Parse implements the SSHKeyParser interface by calling the original global function.
func (p *fileSSHKeyParser) Parse(keyPath string) gossh.Signer {
	if keyPath == "" {
		return nil
	}
	if _, err := os.Stat(keyPath); err != nil {
		return nil
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		slog.Warn("Could not read SSH key path", "path", keyPath, "err", err)
		return nil
	}
	privateKey, err := gossh.ParsePrivateKey(keyBytes)
	if err != nil {
		slog.Warn("Could not parse SSH key", "path", keyPath, "err", err)
		return nil
	}
	return privateKey
}

var _ SSHKeyParser = (*fileSSHKeyParser)(nil)

// SshManager interface defines the methods for interacting with the SSH Manager.
type SshManager interface {
	IsInit() bool
	ExecuteScript(scriptPath, scriptContent string, postRemove bool, runWithSudo bool) error
	WriteFileOnRemoteMachine(path, fileContent string, fileMode os.FileMode) error
	DisablePasswordSSHLogin() error
	RebootCloudInit() error
	DeregisterOS() error
	HostPublicKeyIsValid() error
}

// StandardSshManager struct holds configuration for SSH Manager interaction.
type StandardSshManager struct {
	HostName      string
	UserName      string
	SshPassword   string
	SshKeyPath    string
	Client        ssh.Client
	HostPublicKey gossh.PublicKey
	keyParser     SSHKeyParser
}

var _ SshManager = (*StandardSshManager)(nil)

// ParseSSHPublicKey parses an SSH public key string in authorized_keys format
// and returns the parsed PublicKey. This can be used to validate the key early
// (e.g. during configuration) before passing it to NewStandardSshManager.
func ParseSSHPublicKey(key string) (gossh.PublicKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("SSH public key string is empty")
	}

	// Validate that the first field is the key type (e.g. "ecdsa-sha2-nistp256").
	// ParseAuthorizedKey is lenient and treats unrecognized prefixes as authorized_keys
	// options, which would silently accept malformed input like "test AAAA...".
	fields := strings.Fields(key)
	if len(fields) < 2 {
		return nil, fmt.Errorf("SSH public key must have at least key-type and base64 data, got %d field(s)", len(fields))
	}

	publicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse host public key: %w", err)
	}

	if fields[0] != publicKey.Type() {
		return nil, fmt.Errorf("SSH public key type mismatch: declared %q but parsed key is %q", fields[0], publicKey.Type())
	}

	slog.Debug("Successfully parsed SSH host public key", "type", publicKey.Type())
	return publicKey, nil
}

// NewStandardSshManager Returns new instance of Standard SSH Manager and error.
// The hostPublicKey parameter must be a pre-parsed gossh.PublicKey (use ParseSSHPublicKey to obtain one).
func NewStandardSshManager(hostName, userName, sshPassword, sshKeyPath string, hostPublicKey gossh.PublicKey) (*StandardSshManager, error) {
	slog.Debug("Standard SSH Manager constructor", "host", hostName, "user", userName)

	if hostName == "" || userName == "" || sshPassword == "" || sshKeyPath == "" || hostPublicKey == nil {

		slog.Error(ErrNoneOfConstructorArgsCanBeEmpty.Error())
		return nil, ErrNoneOfConstructorArgsCanBeEmpty
	}

	isInit = true
	return &StandardSshManager{
		HostName:      hostName,
		UserName:      userName,
		SshPassword:   sshPassword,
		Client:        &ssh.NativeClient{},
		HostPublicKey: hostPublicKey,
		SshKeyPath:    sshKeyPath,
		keyParser:     NewFileSSHKeyParser(),
	}, nil
}

func (sc *StandardSshManager) String() string {
	return "{" +
		fmt.Sprintf("HostName: %s, ", sc.HostName) +
		fmt.Sprintf("UserName: %s, ", sc.UserName) +
		fmt.Sprintf("SshKeyPath: %s", sc.SshKeyPath) +
		"}"
}

// IsInit Returns true if constructor succeed else false
func (sc *StandardSshManager) IsInit() bool {
	return isInit
}

func (sc *StandardSshManager) getSshClientConfig() *gossh.ClientConfig {
	authMethods := []gossh.AuthMethod{
		gossh.Password(sc.SshPassword),
	}
	if sshPrivateKey := sc.keyParser.Parse(sc.SshKeyPath); sshPrivateKey != nil {
		authMethods = append(authMethods, gossh.PublicKeys(sshPrivateKey))
	}

	config := &gossh.ClientConfig{
		User:              sc.UserName,
		Auth:              authMethods,
		HostKeyCallback:   gossh.FixedHostKey(sc.HostPublicKey),
		HostKeyAlgorithms: []string{sc.HostPublicKey.Type()},
	}
	return config
}

// HostPublicKeyIsValid checks if the remote host's public key complies with the given one passed as param.
func (sc *StandardSshManager) HostPublicKeyIsValid() error {

	if IsPublicKeyIsValid {
		return nil
	}

	config := sc.getSshClientConfig()
	address := fmt.Sprintf("%s:%d", sc.HostName, port)
	maxAttempts := 5
	currentAttempt := 1
	for {
		slog.Debug("Attempt to dial", "currentAttempt", currentAttempt)
		client, err := gossh.Dial("tcp", address, config)

		if err != nil {
			slog.Warn("Failed to dial SSH server", "err", err)

			if currentAttempt > maxAttempts {
				return fmt.Errorf("failed to dial SSH server: %w", err)
			}

			currentAttempt++
		} else {
			defer client.Close()
			slog.Info("Host public key verification succeeded")
			IsPublicKeyIsValid = true
			return nil
		}
	}
}

// initNativeClient Initialize Native client object
func (sc *StandardSshManager) initNativeClient() error {
	if _, ok := sc.Client.(*ssh.NativeClient); ok {

		slog.Debug("Authenticating using password and SSH key", "path", sc.SshKeyPath)
		auth := &ssh.Auth{
			Passwords: []string{sc.SshPassword},
			Keys:      []string{sc.SshKeyPath},
		}

		nativeClient, err := ssh.NewNativeClient(sc.UserName, sc.HostName, port, auth)
		if err != nil {
			slog.Error("Error creating SSH client", "err", err)
			return err
		}
		slog.Info("SSH native client successfully initialized", "host", sc.HostName, "user", sc.UserName)
		sc.Client = nativeClient
	}
	return nil
}

// runCommand Runs command on remote host and returns command's result and error
func (sc *StandardSshManager) runCommand(command string) (string, error) {

	if err := sc.initNativeClient(); err != nil {
		return "", err
	}
	slog.Debug("Running command via SSH", "command", command, "host", sc.HostName, "user", sc.UserName)
	output, err := sc.Client.Output(command)

	isReboot := strings.Contains(command, "reboot") || strings.Contains(command, "shutdown")

	if err != nil {
		var exitMissingErr *gossh.ExitMissingError
		if errors.As(err, &exitMissingErr) && isReboot {
			slog.Debug("Running command via SSH interrupted by restart", "command", command)
			return output, nil // Treat as success for reboot scenarios
		}

		slog.Error("Error running command", "command", command, "output", output, "err", err)
		return "", err
	}
	slog.Debug("Running command via SSH succeed")
	return output, nil
}

// DisablePasswordSSHLogin disables password authentication for SSH on newly created machine
func (sc *StandardSshManager) DisablePasswordSSHLogin() error {

	commands := []string{
		"sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak",
		`echo "PasswordAuthentication no" | sudo tee /etc/ssh/sshd_config.d/99-disable-password.conf`,
		`echo "AuthenticationMethods publickey" | sudo tee /etc/ssh/sshd_config.d/99-auth-methods.conf`,
		"sudo systemctl reload sshd",
	}

	for _, cmd := range commands {
		if _, err := sc.runCommand(cmd); err != nil {
			slog.Error("Failed to execute command", "cmd", cmd, "err", err)
			return fmt.Errorf("error running '%s': %w", cmd, err)
		}
	}
	slog.Info("Password authentication disabled successfully")

	return nil
}

// RebootCloudInit() error
func (sc *StandardSshManager) RebootCloudInit() error {
	_, err := sc.runCommand(cmdRebootCloudInit)
	if err != nil {
		slog.Error("Error executing cloud-init reboot", "err", err)
		return err
	}
	slog.Info("Cloud-init reboot executed successfully")
	return nil
}

// generateSecureRandomInt generates a secure random integer in the range [min, max] inclusive
func generateSecureRandomInt(min, max int64) (int64, error) {
	if min > max {
		return 0, errors.New("min must be less than or equal to max")
	}

	rangeSize := max - min + 1
	nBig, err := rand.Int(rand.Reader, big.NewInt(rangeSize))
	if err != nil {
		return 0, err
	}

	return min + nBig.Int64(), nil
}

// getRandomPath generates a random path for a script to be executed on the remote machine
func (sc *StandardSshManager) getRandomScriptPath() (string, error) {
	randInt, err := generateSecureRandomInt(1, 999)
	if err != nil {
		slog.Error("Failed to generate secure random number", "err", err)
		return "", err
	}

	path := fmt.Sprintf("%s/%s-via-ssh-%03d.sh", remoteScriptDir, sc.UserName, randInt)
	slog.Debug("Generated random script path", "path", path)
	return path, nil
}

// ExecuteScript Executes script defined by path and content. Executing procedure consists of few steps:
/*
- if script path is not set (empty), then create a secure temporary subdirectory (e.g. /tmp/fsas-nodedriver) with minimal permissions
  and prepare the script using a random filename inside that directory (e.g. /tmp/fsas-nodedriver/username-via-ssh-256.sh)
- generate file in above location with given content and grant executable privileges to file
- execute script
- if param 'postRemove' is true then:
	- remove script after execution else leave it
	- delete secure directory if it is empty.
- if script path is provided, the script is written and executed at that path without directory creation or deletion.
*/
func (sc *StandardSshManager) ExecuteScript(scriptPath, scriptContent string, postRemove, runWithSudo bool) error {
	var path string
	var err error
	if scriptPath == "" {
		path, err = sc.getRandomScriptPath()
		if err != nil {
			return err
		}
	} else {
		path = scriptPath
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	dir := filepath.Dir(absPath)
	if err := sc.createRemoteDir(dir); err != nil {
		slog.Warn("Failed to ensure directory permissions for custom path", "dir", dir, "err", err)
	}

	if err := sc.WriteFileOnRemoteMachine(path, scriptContent, 0744); err != nil {
		return err
	}

	defer func() {
		if postRemove {
			if err := sc.removeRemoteFile(path, runWithSudo); err != nil {
				slog.Error("Failed to remove remote script file", "err", err)
			}
		}

		if err := sc.removeRemoteDir(remoteScriptDir, runWithSudo); err != nil {
			slog.Error("Failed to remove remote directory", "err", err)
		}
	}()

	if err := sc.executeRemoteFile(path, runWithSudo); err != nil {
		return err
	}

	return nil
}

// WriteFileOnRemoteMachine Writes content to file defined in path
func (sc *StandardSshManager) WriteFileOnRemoteMachine(path, fileContent string, fileMode os.FileMode) error {

	if _, ok := sc.Client.(*ssh.NativeClient); ok {
		config := sc.getSshClientConfig()
		address := fmt.Sprintf("%s:%d", sc.HostName, port)
		conn, err := gossh.Dial("tcp", address, config)
		if err != nil {
			slog.Error("Failed to dial SSH host", "host", sc.HostName, "err", err)
			return err
		}

		sftpClient, err := sftp.NewClient(conn)
		if err != nil {
			slog.Error("Failed to create SFTP client", "host", sc.HostName, "err", err)
			return err
		}
		defer sftpClient.Close()

		dstFile, err := sftpClient.Create(path)
		if err != nil {
			slog.Error("Failed to create remote file", "host", sc.HostName, "path", path, "err", err)
			return err
		}
		defer dstFile.Close()

		err = dstFile.Chmod(fileMode)
		if err != nil {
			slog.Error("Failed to change permissions", "permissions", fileMode.String(), "err", err)
			return err
		}
		slog.Info("Successfully added permissions to file", "file", dstFile.Name(), "permissions", fileMode.String())

		srcBuffer := bytes.NewBuffer([]byte(fileContent))
		_, err = io.Copy(dstFile, srcBuffer)
		if err != nil {
			slog.Error("Failed to copy data", "host", sc.HostName, "err", err)
			return err
		}
		slog.Info("File content successfully written to remote destination", "dstFile", dstFile.Name())
	}

	return nil
}

// createRemoteDir creates a remote directory with permissions set to 700
func (sc *StandardSshManager) createRemoteDir(path string) error {
	command := fmt.Sprintf("mkdir -p %s && chmod 700 %s", path, path)
	_, err := sc.runCommand(command)
	if err != nil {
		slog.Error("Failed to create secure remote directory", "path", path, "err", err)
	}
	return err
}

// removeRemoteDir removes the remote directory from remote machine
func (sc *StandardSshManager) removeRemoteDir(path string, runWithSudo bool) error {
	var cmd string
	if runWithSudo {
		cmd = fmt.Sprintf(`if [ -d %[1]s ] && [ -z "$(ls -A %[1]s)" ]; then sudo rmdir %[1]s; fi`, path)
	} else {
		cmd = fmt.Sprintf(`if [ -d %[1]s ] && [ -z "$(ls -A %[1]s)" ]; then rmdir %[1]s; fi`, path)
	}
	if _, err := sc.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to remove empty remote directory %s: %w", path, err)
	}
	return nil
}

// executeRemoteFile Executes file on remote machine but without providing output, only result
func (sc *StandardSshManager) executeRemoteFile(path string, runWithSudo bool) error {
	var command string
	if runWithSudo {
		command = fmt.Sprintf("sudo %s", path)
	} else {
		command = fmt.Sprintf("%s", path)
	}

	if _, err := sc.runCommand(command); err != nil {
		return err
	}

	return nil
}

// removeRemoteFile Removes file from remote machine
func (sc *StandardSshManager) removeRemoteFile(path string, runWithSudo bool) error {
	var command string
	if runWithSudo {
		command = fmt.Sprintf("sudo rm %s", path)
	} else {
		command = fmt.Sprintf("rm %s", path)
	}

	if _, err := sc.runCommand(command); err != nil {
		return err
	}

	return nil
}

// DeregisterOS - De-registers SLES OS using SUSEConnect
func (sc *StandardSshManager) DeregisterOS() error {
	jsonOutput, err := sc.runCommand(cmdGetStatusOS)
	if err != nil {
		slog.Error("Could not get SUSE product status before deregistration", "err", err)
		return err
	}

	var products []models.SuseProduct
	if err := json.Unmarshal([]byte(jsonOutput), &products); err != nil {
		slog.Error("Failed to parse SUSE status JSON before deregistration", "err", err)
		return err
	}

	anyRegistered := false
	for _, p := range products {
		if p.Status == "Registered" {
			anyRegistered = true
			break
		}
	}

	// Only deregister if something was registered
	if !anyRegistered {
		slog.Info("Skipping OS deregistration: no products registered")
		return nil
	}

	_, err = sc.runCommand(cmdDeregisterOS)
	if err != nil {
		slog.Error("Error executing SLES OS deregistration", "err", err)
		return err
	}
	slog.Info("SLES OS deregistered successfully")
	return nil
}
