package sshutils

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mock "github.com/fujitsu/docker-machine-driver-fsas/sshutils/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

const (
	HOST_PUBLIC_KEY = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="
)

var (
	MOCK_ERROR_FOR_OUTPUT_METHOD = fmt.Errorf("mock error for Output method")
)

// MockSSHClient implements the ssh.Client interface
type MockSSHClient struct {
	ExecutedCommands []string
	OutputFunc func(command string) (string, error)
}

// Output runs a command on the remote host and returns its output
func (c *MockSSHClient) Output(command string) (string, error) {
	c.ExecutedCommands = append(c.ExecutedCommands, command)
	if c.OutputFunc != nil {
		return c.OutputFunc(command)
	}
	// Default behavior if OutputFunc is not set: return the command as output.
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

	exitCode := m.Run() // run tests

	// tear-down code here
	publicKeyIsValid = false

	os.Exit(exitCode)
}

func Test_runCommand_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = "" // This is a trick to avoid initializing SSH key auth method during the unit test

	command := "echo Hello"
	output, err := manager.runCommand(command)

	assert.NoError(t, err)
	assert.Equal(t, command, output)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, command, mockClient.ExecutedCommands[0])
}

func Test_runCommand_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""  // This is a trick to avoid initializing SSH key auth method during the unit test

	output, err := manager.runCommand("custom command")

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Equal(t, output, "")
}

func Test_runCommand_Success_Missing_Exit(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return command, &gossh.ExitMissingError{}
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""  // This is a trick to avoid initializing SSH key auth method during the unit test

	command := "cloud-init --reboot"
	output, err := manager.runCommand(command)

	assert.NoError(t, err)
	assert.Equal(t, command, output)
}

func Test_createSSHKey_error(t *testing.T) {
	sc := &StandardSshManager{SshKeyPath: "example/path/to/key"}

	err := sc.createSSHKey()

	assert.EqualError(t, err, "Error writing keys to file(s): Unable to write file")
}

func Test_createSSHKey(t *testing.T) {
	sc := &StandardSshManager{}

	tempDir := t.TempDir()
	sshKeyName := "newSshKey"
	sshKeyPath := filepath.Join(tempDir, sshKeyName)
	sc.SshKeyPath = sshKeyPath
	err := sc.createSSHKey()

	assert.NoError(t, err)
	assert.FileExists(t, sshKeyPath)
	assert.FileExists(t, sshKeyPath + ".pub")
}

func Test_transferSSHKeyToMachineOpenFail(t *testing.T) {
	sc, err := NewStandardSshManager("host", "user", "password", "mock/path/to/key", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	err = sc.transferSSHKeyToMachine()

	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func Test_transferSSHKeyToMachine(t *testing.T) {
	sc, err := NewStandardSshManager("host", "user", "password", "mock/path/to/key", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	sc.Client = mockClient

	// Create a temporary public key file
	tempDir := t.TempDir()
	sshKeyName := "id_rsa"
	pubKeyName := sshKeyName + ".pub"
	pubKeyPath := filepath.Join(tempDir, pubKeyName)
	keyContent := "ssh-rsa AAAA..."
	err = os.WriteFile(pubKeyPath, []byte(keyContent), 0644)
	require.NoError(t, err)

	sshKeyPath := filepath.Join(tempDir, sshKeyName)
	sc.SshKeyPath = sshKeyPath
	err = sc.transferSSHKeyToMachine()

	assert.NoError(t, err)
	require.Len(t, mockClient.ExecutedCommands, 1)
	expectedCommand := fmt.Sprintf(`echo "%s" >> $HOME/.ssh/authorized_keys`, keyContent)
	assert.Equal(t, expectedCommand, mockClient.ExecutedCommands[0])
}

func TestWriteFileOnRemoteMachine_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.Client = &MockSSHClient{}
	manager.SshKeyPath = ""

	err = manager.WriteFileOnRemoteMachine("", "Lorem impsum", 0744)
	assert.NoError(t, err)
}

func Test_executeRemoteFile_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.executeRemoteFile("custom command", true)
	assert.NoError(t, err)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "sudo custom command")
}

func Test_executeRemoteFile_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.executeRemoteFile("custom command", false)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "custom command")
}

func Test_removeRemoteFile_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.removeRemoteFile("file", true)
	assert.NoError(t, err)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "sudo rm file")
}

func Test_removeRemoteFile_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.removeRemoteFile("file", false)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "rm file")
}

func TestGenerateSecureRandomInt(t *testing.T) {
    for i := 0; i < 100; i++ {
        result, err := generateSecureRandomInt(1, 999)
        require.NoError(t, err, "unexpected error")
        assert.True(t, result >= 1 && result <= 999, "result out of bounds: %d", result)
    }
}

func TestGenerateSecureRandomInt_InvalidRange(t *testing.T) {
    _, err := generateSecureRandomInt(10, 1)
    assert.Error(t, err, "generateSecureRandomInt() should return an error when min > max, but got nil")
}

func Test_ExecuteScript_RemovesScriptAndDirectory(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			return "", nil
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	scriptName := "" // it means generating filename
	scriptContent := "echo Hello"
	postRemove := true
	runWithSudo := true
	err = manager.ExecuteScript(scriptName, scriptContent, postRemove, runWithSudo)
	assert.NoError(t, err)

	fmt.Println(mockClient.ExecutedCommands)
	require.Len(t, mockClient.ExecutedCommands, 4)

	assert.Contains(t, mockClient.ExecutedCommands[0], "mkdir -p /tmp/fsas-nodedriver && chmod 700 /tmp/fsas-nodedriver")
	assert.Contains(t, mockClient.ExecutedCommands[1], "sudo /tmp/fsas-nodedriver/user1-via-ssh")
	assert.Contains(t, mockClient.ExecutedCommands[2], "sudo rm /tmp/fsas-nodedriver/user1-via-ssh")
	assert.Contains(t, mockClient.ExecutedCommands[3], "sudo rmdir /tmp/fsas-nodedriver")
}

func Test_createRemoteDir(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.createRemoteDir("/tmp/mytestdir")
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[0], "mkdir -p /tmp/mytestdir")
}

func Test_getRandomScriptPath(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	path, err := manager.getRandomScriptPath()
	assert.NoError(t, err)

	assert.Contains(t, path, "/tmp/fsas-nodedriver/user1-via-ssh-")
	assert.True(t, strings.HasSuffix(path, ".sh"))
}

func Test_removeRemoteDir(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.removeRemoteDir("/tmp/mytestdir", false)
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[0], "rmdir /tmp/mytestdir")
}

func Test_DisablePasswordSSHLogin_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.DisablePasswordSSHLogin()
	assert.NoError(t, err)

	expectedCommands := []string{
		"sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak",
		`echo "PasswordAuthentication no" | sudo tee /etc/ssh/sshd_config.d/99-disable-password.conf`,
		`echo "AuthenticationMethods publickey" | sudo tee /etc/ssh/sshd_config.d/99-auth-methods.conf`,
		"sudo systemctl reload sshd",
	}
	assert.Equal(t, mockClient.ExecutedCommands, expectedCommands)
}

func Test_DisablePasswordSSHLogin_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			// Fail on the second command
			if strings.Contains(cmd, "tee") {
				return "", MOCK_ERROR_FOR_OUTPUT_METHOD
			}
			return "", nil
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.DisablePasswordSSHLogin()

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Len(t, mockClient.ExecutedCommands, 2)
}

func Test_RebootCloudInit_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.RebootCloudInit()
	assert.NoError(t, err)

	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, cmdRebootCloudInit, mockClient.ExecutedCommands[0])
}

func Test_RebootCloudInit_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.RebootCloudInit()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
}

func TestRegisterOS_SuccessWithUnregisteredModules(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockRegcode := "somecode0101001"
	mockEmail := "hoge@example.com"
	initialRegCmd := fmt.Sprintf(cmdRegisterOS, mockRegcode, mockEmail)
	getStatusCmd := cmdGetStatusOS
	moduleRegCmd := "sudo -E SUSEConnect -p sle-module-public-cloud/15.6/x86_64"

	products := []models.SuseProduct{
		{Identifier: "SLES", Version: "15.6", Arch: "x86_64", Status: "Registered"},
		{Identifier: "sle-module-public-cloud", Version: "15.6", Arch: "x86_64", Status: "Not Registered"},
	}
	jsonOutput, err := json.Marshal(products)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			switch command {
			case initialRegCmd, moduleRegCmd:
				return "", nil
			case getStatusCmd:
				return string(jsonOutput), nil
			default:
				return "", fmt.Errorf("unexpected command: %s", command)
			}
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS(mockRegcode, mockEmail)

	assert.NoError(t, err)
	expectedCommands := []string{initialRegCmd, getStatusCmd, moduleRegCmd}
	assert.Equal(t, expectedCommands, mockClient.ExecutedCommands)
}

func TestRegisterOS_SuccessAllModulesRegistered(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockRegcode := "somecode0101001"
	mockEmail := "hoge@example.com"
	products := []models.SuseProduct{{Identifier: "SLES", Version: "15.6", Arch: "x86_64", Status: "Registered"}}
	jsonOutput, err := json.Marshal(products)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			if command == cmdGetStatusOS {
				return string(jsonOutput), nil
			}
			return "", nil
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS(mockRegcode, mockEmail)

	assert.NoError(t, err)
	expectedCommands := []string{
		fmt.Sprintf(cmdRegisterOS, mockRegcode, mockEmail),
		cmdGetStatusOS,
	}
	assert.Equal(t, expectedCommands, mockClient.ExecutedCommands)
}

func TestRegisterOS_SkipWithNoRegcode(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockClient := &MockSSHClient{}
	manager.Client = mockClient

	err = manager.RegisterOS("", "hoge@example.com")

	assert.NoError(t, err)
	assert.Empty(t, mockClient.ExecutedCommands)
}

func TestRegisterOS_FailOnInitialRegistration(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS("somecode", "email")

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, fmt.Sprintf(cmdRegisterOS, "somecode", "email"), mockClient.ExecutedCommands[0])
}

func TestRegisterOS_FailOnGetStatus(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			if command == cmdGetStatusOS {
				return "", MOCK_ERROR_FOR_OUTPUT_METHOD
			}
			return "", nil
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS("somecode", "email")

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Len(t, mockClient.ExecutedCommands, 2)
}

func TestRegisterOS_FailOnInvalidJSON(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			if command == cmdGetStatusOS {
				return "this is not valid json", nil
			}
			return "", nil
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS("somecode", "email")

	require.Error(t, err)
	_, isJsonError := err.(*json.SyntaxError)
	assert.True(t, isJsonError, "error should be a JSON syntax error")
	assert.Len(t, mockClient.ExecutedCommands, 2)
}

func TestRegisterOS_FailOnModuleRegistration(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	require.NoError(t, err)
	manager.SshKeyPath = ""

	products := []models.SuseProduct{{Identifier: "sle-module", Status: "Not Registered"}}
	jsonOutput, err := json.Marshal(products)
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			if command == cmdGetStatusOS {
				return string(jsonOutput), nil
			}
			if strings.HasPrefix(command, "sudo -E SUSEConnect -p") {
				return "", MOCK_ERROR_FOR_OUTPUT_METHOD
			}
			return "", nil
		},
	}
	manager.Client = mockClient

	err = manager.RegisterOS("somecode", "email")

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Len(t, mockClient.ExecutedCommands, 3)
}

func Test_getSSHKeyAuthMethod_EmptyPath(t *testing.T) {
	parser := NewFileSSHKeyParser()
	signer := parser.Parse("")
	assert.Nil(t, signer)
}

func Test_getSSHKeyAuthMethod_FileNotExist(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "non-existent-key.pem")
	parser := NewFileSSHKeyParser()
	signer := parser.Parse(nonExistentPath)
	assert.Nil(t, signer)
}

func Test_getSSHKeyAuthMethod_NotAFile(t *testing.T) {
	parser := NewFileSSHKeyParser()
	dirPath := t.TempDir()
	signer := parser.Parse(dirPath)
	assert.Nil(t, signer)
}

func Test_getSSHKeyAuthMethod_InvalidKey(t *testing.T) {
	parser := NewFileSSHKeyParser()
	tempFile, err := os.CreateTemp(t.TempDir(), "test-file-*.tmp")
	require.NoError(t, err, "Failed to create temp file")

	_, err = tempFile.WriteString("this is not a valid ssh key")
	require.NoError(t, err, "Failed to write to temp file")

	err = tempFile.Close()
	require.NoError(t, err, "Failed to close temp file")

	signer := parser.Parse(tempFile.Name())
	assert.Nil(t, signer)
}

func Test_getSSHKeyAuthMethod_Success(t *testing.T) {
	parser := NewFileSSHKeyParser()
	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate rsa2048 key")

	// Marshal the private key into PKCS1 format
	pemBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	require.NoError(t, err, "Failed to marshal private key")

	// Create a temporary file to store the key
	tempFile, err := os.CreateTemp(t.TempDir(), "test-ssh-key-*.pem")
	require.NoError(t, err, "Failed to create temp file for key")

	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pemBytes,
	}

	// Write the PEM-encoded key to the file
	err = pem.Encode(tempFile, pemBlock)
	require.NoError(t, err, "Failed to write PEM block to file")

	err = tempFile.Close()
	require.NoError(t, err, "Failed to close temp key file")

	signer := parser.Parse(tempFile.Name())
	assert.NotNil(t, signer)
}

func Test_getSshClientConfig_PassOnly(t *testing.T) {
	mockParser := mock.NewMockSSHKeyParser(t)
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := gossh.NewPublicKey(pub)
	require.NoError(t, err)

	keyPath := "/path/to/invalid/key"
	mockParser.On("Parse", keyPath).Return(nil)

	manager := &StandardSshManager{
		UserName:      "testuser",
		SshPassword:   "password123",
		SshKeyPath:    keyPath,
		HostPublicKey: sshPub,
		keyParser:     mockParser,
	}

	config := manager.getSshClientConfig()
	assert.NotNil(t, config)
	assert.Len(t, config.Auth, 1)
	mockParser.AssertExpectations(t)
}

func Test_getSshClientConfig_PassAndKey(t *testing.T) {
	mockParser := mock.NewMockSSHKeyParser(t)
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := gossh.NewPublicKey(pub)
	require.NoError(t, err)
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)
	keyPath := "/path/to/valid/key"

	// Configure the mock to return a valid signer when Parse is called.
	// This simulates finding and successfully parsing a private key.
	mockParser.On("Parse", keyPath).Return(signer)

	manager := &StandardSshManager{
		UserName:      "testuser",
		SshPassword:   "password123",
		SshKeyPath:    keyPath,
		HostPublicKey: sshPub,
		keyParser:     mockParser, // Inject the mock
	}

	config := manager.getSshClientConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "testuser", config.User)
	// We expect two auth methods: password and public key.
	assert.Len(t, config.Auth, 2, "There should be both password and public key authentication")
	assert.Equal(t, []string{sshPub.Type()}, config.HostKeyAlgorithms)

	// Verify that the mock was called as expected.
	mockParser.AssertExpectations(t)
}

func Test_DeregisterOS_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.DeregisterOS()
	assert.NoError(t, err)

	expectedCommands := []string{cmdDeregisterOS}
	assert.Equal(t, mockClient.ExecutedCommands, expectedCommands)
}

func Test_DeregisterOS_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", "password1", "mock/path", HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPath = ""

	err = manager.DeregisterOS()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
}
