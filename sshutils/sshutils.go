package sshutils

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"

	"github.com/pkg/sftp"
	"github.com/rancher/machine/libmachine/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const (
	port                                    = 22
	cmdRebootCloudInit                      = "sudo cloud-init clean --logs --reboot"
	remoteScriptDir                         = "/tmp/fsas-nodedriver"
	SSH_CONNECT_ATTEMPT_DELAY time.Duration = 5 * time.Second
)

var (
	ErrNoneOfConstructorArgsCanBeEmpty = errors.New("none of the arguments can be empty; neither 'hostName', 'userName', 'sshPassword', 'hostPublicKey'")
	isInit                             = false
	publicKeyIsValid                   = false
)

// SshManager interface defines the methods for interacting with the SSH Manager.
type SshManager interface {
	IsInit() bool
	SendStopCommand() error
	ExchangeKeys(sshKeyPath string) error
	ExecuteScript(scriptPath, scriptContent string, postRemove bool, runWithSudo bool) error
	WriteFileOnRemoteMachine(path, fileContent string, fileMode os.FileMode) error
	DisablePasswordSSHLogin() error
	RebootCloudInit() error
}

// StandardSshManager struct holds configuration for SSH Manager interaction.
type StandardSshManager struct {
	HostName      string
	UserName      string
	SshPassword   string
	SshKeyPath    string
	Client        ssh.Client
	HostPublicKey gossh.PublicKey
}

var _ SshManager = (*StandardSshManager)(nil)

// NewStandardSshManager Returns new instance of Standard SSH Manager and error
func NewStandardSshManager(hostName, userName, sshPassword, hostPublicKey string) (*StandardSshManager, error) {
	slog.Debug("Standard SSH Manager constructor: ", "host", hostName, "user", userName)

	if hostName == "" || userName == "" || sshPassword == "" || hostPublicKey == "" {

		slog.Error(ErrNoneOfConstructorArgsCanBeEmpty.Error())
		return nil, ErrNoneOfConstructorArgsCanBeEmpty
	}

	publicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(hostPublicKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse host public key: %w", err)
	}

	isInit = true
	return &StandardSshManager{
		HostName:      hostName,
		UserName:      userName,
		SshPassword:   sshPassword,
		Client:        &ssh.NativeClient{},
		HostPublicKey: publicKey,
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

// SendStopCommand Sends stop (shutdown) command to remote.
/*
WARNING: user that sends stop command must be configured as sudo without password,
	     otherwise it will not work
*/
func (sc *StandardSshManager) SendStopCommand() error {
	_, err := sc.runCommand("sudo shutdown -h now")
	return err
}

func (sc *StandardSshManager) getSshClientConfig() *gossh.ClientConfig {
	config := &gossh.ClientConfig{
		User: sc.UserName,
		Auth: []gossh.AuthMethod{
			gossh.Password(sc.SshPassword),
		},
		HostKeyCallback:   gossh.FixedHostKey(sc.HostPublicKey),
		HostKeyAlgorithms: []string{sc.HostPublicKey.Type()},
	}
	return config
}

// hostPublicKeyIsValid checks if the remote host's public key complies with the given one passed as param.
func (sc *StandardSshManager) hostPublicKeyIsValid() error {

	if publicKeyIsValid {
		return nil
	}

	config := sc.getSshClientConfig()
	address := fmt.Sprintf("%s:%d", sc.HostName, port)
	maxAttempts := 20
	currentAttempt := 1
	for {
		slog.Debug("Attempt to dial: ", "currentAttempt", currentAttempt)
		client, err := gossh.Dial("tcp", address, config)

		if err != nil {
			slog.Warn("Failed to dial SSH server: ", "err", err)

			if currentAttempt > maxAttempts {
				return fmt.Errorf("failed to dial SSH server: %w", err)
			}

			// sleep added to handle immediate connection refused response
			slog.Info("Waiting for next attempt to dial: ", "sleepTime", SSH_CONNECT_ATTEMPT_DELAY)
			time.Sleep(SSH_CONNECT_ATTEMPT_DELAY)

			currentAttempt++
		} else {
			defer client.Close()
			slog.Info("Host public key verification succeeded")
			publicKeyIsValid = true
			return nil
		}
	}
}

// initNativeClient Initialize Native client object
func (sc *StandardSshManager) initNativeClient() error {
	if _, ok := sc.Client.(*ssh.NativeClient); ok {

		var auth *ssh.Auth

		if sc.SshKeyPath == "" {
			auth = &ssh.Auth{
				Passwords: []string{sc.SshPassword},
			}
		} else {
			auth = &ssh.Auth{
				Passwords: []string{sc.SshPassword},
				Keys:      []string{sc.SshKeyPath},
			}
		}

		nativeClient, err := ssh.NewNativeClient(sc.UserName, sc.HostName, port, auth)
		if err != nil {
			slog.Error("Error creating SSH client:", "err", err)
			return err
		}
		slog.Info("SSH native client successfully initialized: ", "host", sc.HostName, "user", sc.UserName)
		sc.Client = nativeClient
	}
	return nil
}

// runCommand Runs command on remote host and returns command's result and error
func (sc *StandardSshManager) runCommand(command string) (string, error) {

	if err := sc.hostPublicKeyIsValid(); err != nil {
		return "", err
	}

	if err := sc.initNativeClient(); err != nil {
		return "", err
	}
	slog.Debug("Running command via SSH: ", "command", command, "host", sc.HostName, "user", sc.UserName)
	output, err := sc.Client.Output(command)

	isReboot := strings.Contains(command, "reboot") || strings.Contains(command, "shutdown")

	if err != nil {
		var exitMissingErr *gossh.ExitMissingError
		if errors.As(err, &exitMissingErr) && isReboot {
			slog.Debug("Running command via SSH interrupted by restart: ", "command", command)
			return output, nil // Treat as success for reboot scenarios
		}

		slog.Error("Error running command:", "command", command, "output", output, "err", err)
		return "", err
	}
	slog.Debug("Running command via SSH succeed")
	return output, nil
}

func (sc *StandardSshManager) ExchangeKeys(sshKeyPath string) error {

	if err := sc.createSSHKey(sshKeyPath); err != nil {
		slog.Error("Could not generate SSH keys because of an error: ", "err", err)
		return err
	}

	if err := sc.transferSSHKeyToMachine(sshKeyPath); err != nil {
		slog.Error("Could not transfer SSH keys because of an error: ", "err", err)
		return err
	}

	slog.Info("SSH key pair exchanged successfully")
	return nil
}

// createSSHKey is responsible for generating new SSH key pair
func (sc *StandardSshManager) createSSHKey(sshKeyPath string) error {

	if err := ssh.GenerateSSHKey(sshKeyPath); err != nil {
		slog.Error("SSH key could not be generated because of an error: ", "err", err)
		return err
	}
	slog.Info("SSH key pair generated successfully: ", "path", sshKeyPath)

	return nil
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
			slog.Error("Failed to execute command: ", "cmd", cmd, "err", err)
			return fmt.Errorf("error running '%s': %w", cmd, err)
		}
	}
	slog.Info("Password authentication disabled successfully.")

	return nil
}

// RebootCloudInit() error
func (sc *StandardSshManager) RebootCloudInit() error {
	_, err := sc.runCommand(cmdRebootCloudInit)
	if err != nil {
		slog.Error("Error executing cloud-init reboot: ", "err", err)
		return err
	}
	slog.Info("Cloud-init reboot executed successfully")
	return nil
}

// transferSSHKeyToMachine is responsible for transferring existing SSH key to newly created machine
func (sc *StandardSshManager) transferSSHKeyToMachine(sshKeyPath string) error {

	pubSshKeyPath := sshKeyPath + ".pub"
	slog.Debug("Opening file: ", "file", pubSshKeyPath)

	buffer, err := os.ReadFile(pubSshKeyPath)
	if err != nil {
		slog.Error("Error opening file: ", "sshKeyPath", sshKeyPath)
		return err
	}

	command := fmt.Sprintf(`echo "%s" >> $HOME/.ssh/authorized_keys`, string(buffer))
	_, err = sc.runCommand(command)
	if err != nil {
		slog.Error("Error writing public key: ", "err", err)
		return err
	}

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
		slog.Error("Failed to generate secure random number: ", "err", err)
		return "", err
	}

	path := fmt.Sprintf("%s/%s-via-ssh-%03d.sh", remoteScriptDir, sc.UserName, randInt)
	slog.Debug("Generated random script path: ", "path", path)
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
		slog.Warn("Failed to ensure directory permissions for custom path: ", "dir", dir, "err", err)
	}

	if err := sc.WriteFileOnRemoteMachine(path, scriptContent, 0744); err != nil {
		return err
	}

	defer func() {
		if postRemove {
			if err := sc.removeRemoteFile(path, runWithSudo); err != nil {
				slog.Error("Failed to remove remote script file: ", "err", err)
			}
		}

		if err := sc.removeRemoteDir(remoteScriptDir, runWithSudo); err != nil {
			slog.Error("Failed to remove remote directory: ", "err", err)
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
			slog.Error("Failed to dial SSH host: ", "host", sc.HostName, "err", err)
			return err
		}

		sftpClient, err := sftp.NewClient(conn)
		if err != nil {
			slog.Error("Failed to create SFTP client: ", "host", sc.HostName, "err", err)
			return err
		}
		defer sftpClient.Close()

		dstFile, err := sftpClient.Create(path)
		if err != nil {
			slog.Error("Failed to create remote file: ", "host", sc.HostName, "path", path, "err", err)
			return err
		}
		defer dstFile.Close()

		err = dstFile.Chmod(fileMode)
		if err != nil {
			slog.Error("Failed to change permissions: ", "permissions", fileMode.String(), "err", err)
			return err
		}
		slog.Info("Successfully added permissions to file: ", "file", dstFile.Name(), "permissions", fileMode.String())

		srcBuffer := bytes.NewBuffer([]byte(fileContent))
		_, err = io.Copy(dstFile, srcBuffer)
		if err != nil {
			slog.Error("Failed to copy data: ", "host", sc.HostName, "err", err)
			return err
		}
		slog.Info("File content successfully written to remote destination: ", "dstFile", dstFile.Name())
	}

	return nil
}

// createRemoteDir creates a remote directory with permissions set to 700
func (sc *StandardSshManager) createRemoteDir(path string) error {
	command := fmt.Sprintf("mkdir -p %s && chmod 700 %s", path, path)
	_, err := sc.runCommand(command)
	if err != nil {
		slog.Error("Failed to create secure remote directory: ", "path", path, "err", err)
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
