package sshutils

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	gossh "golang.org/x/crypto/ssh"
)

const (
	HOST_PUBLIC_KEY = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="
)

var (
	MOCK_ERROR_FOR_OUTPUT_METHOD = fmt.Errorf("mock error for Output method")
	RETURN_ERROR_EXIT_MISSING = false
	RETURN_ERROR = false

	// global variable to store the last executed SSH command
	LastExecutedSSHCommand string
)

// MockSSHClient implements the ssh.Client interface
type MockSSHClient struct{
	ExecutedCommands []string
}

// Output runs a command on the remote host and returns its output
func (c *MockSSHClient) Output(command string) (string, error) {
	LastExecutedSSHCommand = command
	c.ExecutedCommands = append(c.ExecutedCommands, command)
	if strings.Contains(command, "return-error") || RETURN_ERROR {
		return "", MOCK_ERROR_FOR_OUTPUT_METHOD
	}
	if RETURN_ERROR_EXIT_MISSING {
		return command, &gossh.ExitMissingError{}
	}
	return command, nil
}

// Shell requests a shell from the remote host.
func (c *MockSSHClient) Shell(args ...string) error {
	return nil
}

// Start starts the specified command without waiting for it to finish.
func (c *MockSSHClient) Start(command string) (io.ReadCloser, io.ReadCloser, error) {
	return nil, nil, nil
}

// Wait waits for the command started by Start to exit.
func (c *MockSSHClient) Wait() error {
	return nil
}

func TestMain(m *testing.M) {

	// setup code here
	publicKeyIsValid = true
	LastExecutedSSHCommand = ""

	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	sshKeyDir, err := filepath.Abs(filepath.Join(homeDir, ".ssh"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = os.MkdirAll(sshKeyDir, 0o700)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	exitCode := m.Run() // run tests

	// tear-down code here
	publicKeyIsValid = false

	os.Exit(exitCode)
}

func Test_runCommand_Success(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	command := "echo Hello"
	output, err := manager.runCommand(command)
	assert.NoError(t, err)
	assert.Equal(t, command, output)
	assert.Equal(t, "echo Hello", LastExecutedSSHCommand)
	//t.Logf("captured command: %s", LastExecutedSSHCommand)
}

func Test_runCommand_Fail(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	command := "return-error"
	output, err := manager.runCommand(command)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Equal(t, output, "")
}

func Test_runCommand_Success_Missing_Exit(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	command := "cloud-init --reboot"
	RETURN_ERROR_EXIT_MISSING = true
	output, err := manager.runCommand(command)
	RETURN_ERROR_EXIT_MISSING = false
	assert.NoError(t, err)
	assert.Equal(t, command, output)
}

func Test_createSSHKey_error(t *testing.T) {
	sc := &StandardSshManager{}

	err := sc.createSSHKey("example/path/to/key")

	assert.EqualError(t, err, "Error writing keys to file(s): Unable to write file")
}

func Test_createSSHKey(t *testing.T) {
	sc := &StandardSshManager{}

	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	sshKeyName := "newSssKeys"
	sshKeyNamePub := "newSssKeys.pub"
	sshKeyPath, err := filepath.Abs(filepath.Join(homeDir, ".ssh", sshKeyName))
	assert.NoError(t, err)

	err = sc.createSSHKey(sshKeyPath)

	assert.NoError(t, err)

	homeAbsPath, err := filepath.Abs(homeDir)
	assert.NoError(t, err)
	dirs, err := os.ReadDir(homeAbsPath)
	assert.NoError(t, err)

	s := []string{}
	for _, entry := range dirs {
		s = append(s, entry.Name())
	}
	assert.Contains(t, s, ".ssh")

	sshDirPath, err := filepath.Abs(filepath.Join(homeDir, ".ssh"))
	assert.NoError(t, err)
	sshDirs, err := os.ReadDir(sshDirPath)
	assert.NoError(t, err)
	ss := []string{}
	for _, entrya := range sshDirs {
		ss = append(ss, entrya.Name())
	}
	assert.Contains(t, ss, sshKeyName)
	assert.Contains(t, ss, sshKeyNamePub)
}

func Test_transferSSHKeyToMachineOpenFail(t *testing.T) {
	sc := &StandardSshManager{}

	err := sc.transferSSHKeyToMachine("dummy_file")

	assert.EqualError(t, err, fmt.Sprintf("open %s.pub: no such file or directory", "dummy_file"))
}

func Test_transferSSHKeyToMachine(t *testing.T) {
	sc, _ := NewStandardSshManager("host", "user", "password", HOST_PUBLIC_KEY)
	sc.Client = &MockSSHClient{}

	sshKeyPath, _ := os.CreateTemp("", "*id_rsa.pub")
	defer os.Remove(sshKeyPath.Name())

	err := sc.transferSSHKeyToMachine(strings.Trim(sshKeyPath.Name(), ".pub"))
	assert.NoError(t, err)
}

func TestWriteFileOnRemoteMachine_Success(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	err = manager.WriteFileOnRemoteMachine("", "Lorem impsum", 0744)
	assert.NoError(t, err)
}

func Test_executeRemoteFile_Success(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	err = manager.executeRemoteFile("", false)
	assert.NoError(t, err)
}

func Test_executeRemoteFile_Fail(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	err = manager.executeRemoteFile("return-error", false)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
}

func Test_removeRemoteFile_Success(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	err = manager.removeRemoteFile("", false)
	assert.NoError(t, err)
}

func Test_removeRemoteFile_Fail(t *testing.T) {

	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	if err != nil {
		assert.NoError(t, err, "unexpected error")
	}
	manager.Client = &MockSSHClient{}

	err = manager.removeRemoteFile("return-error", false)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
}

func TestGenerateSecureRandomInt(t *testing.T) {
    for i := 0; i < 1000; i++ {
        result, err := generateSecureRandomInt(1, 999)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if result < 1 || result > 999 {
            t.Errorf("result out of bounds: %d", result)
        }
    }
}

func TestGenerateSecureRandomInt_InvalidRange(t *testing.T) {
    _, err := generateSecureRandomInt(10, 1)
    if err == nil {
        t.Errorf("generateSecureRandomInt() should return an error when min > max, but got nil")
    }
}

func Test_ExecuteScript_RemovesScriptAndDirectory(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	script := "echo Hello"
	err = manager.ExecuteScript("", script, true, false)
	assert.NoError(t, err)

	sliceLength := len(mockClient.ExecutedCommands)
	assert.Contains(t, mockClient.ExecutedCommands[0], "mkdir -p /tmp/fsas-nodedriver")
	assert.Contains(t, mockClient.ExecutedCommands[sliceLength-2], "rm ")
	assert.Contains(t, mockClient.ExecutedCommands[sliceLength-1], "rmdir ")
}

func Test_createRemoteDir(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	err = manager.createRemoteDir("/tmp/mytestdir")
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[len(mockClient.ExecutedCommands)-1], "mkdir -p /tmp/mytestdir")
}

func Test_getRandomScriptPath(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	path, err := manager.getRandomScriptPath()
	assert.NoError(t, err)

	assert.Contains(t, path, "/tmp/fsas-nodedriver/user1-via-ssh-")
	assert.True(t, strings.HasSuffix(path, ".sh"))
}

func Test_removeRemoteDir(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	err = manager.removeRemoteDir("/tmp/mytestdir", false)
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[len(mockClient.ExecutedCommands)-1], "rmdir /tmp/mytestdir")
}

func Test_DisablePasswordSSHLogin_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	err = manager.DisablePasswordSSHLogin()
	assert.NoError(t, err)

	expectedCommands := []string {
		"sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak",
		`echo "PasswordAuthentication no" | sudo tee /etc/ssh/sshd_config.d/99-disable-password.conf`,
		`echo "AuthenticationMethods publickey" | sudo tee /etc/ssh/sshd_config.d/99-auth-methods.conf`,
		"sudo systemctl reload sshd",
	}
	assert.ElementsMatch(t, mockClient.ExecutedCommands, expectedCommands)
}

func Test_DisablePasswordSSHLogin_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil
	RETURN_ERROR = true

	err = manager.DisablePasswordSSHLogin()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	RETURN_ERROR = false
}

func Test_RebootCloudInit_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil

	err = manager.RebootCloudInit()
	assert.NoError(t, err)

	expectedCommands := []string {
		"sudo cloud-init clean --logs --reboot",
	}
	assert.ElementsMatch(t, mockClient.ExecutedCommands, expectedCommands)
}

func Test_RebootCloudInit_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	mockClient.ExecutedCommands = nil
	RETURN_ERROR = true

	err = manager.RebootCloudInit()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	RETURN_ERROR = false
}
