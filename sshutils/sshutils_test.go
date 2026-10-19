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

	"github.com/fujitsu/docker-machine-driver-fsas/models"
	mock "github.com/fujitsu/docker-machine-driver-fsas/sshutils/mock"
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

// parsedHostPublicKey returns the pre-parsed gossh.PublicKey for HOST_PUBLIC_KEY.
func parsedHostPublicKey(t *testing.T) gossh.PublicKey {
	t.Helper()
	key, err := ParseSSHPublicKey(HOST_PUBLIC_KEY)
	require.NoError(t, err)
	return key
}

// MockSSHClient implements the ssh.Client interface
type MockSSHClient struct {
	ExecutedCommands []string
	OutputFunc       func(command string) (string, error)
}

type sshDialFunc func(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error)

func (dial sshDialFunc) Dial(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
	return dial(network, addr, config)
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
	IsPublicKeyValid = true

	exitCode := m.Run() // run tests

	// tear-down code here
	IsPublicKeyValid = false

	os.Exit(exitCode)
}

func TestParseSSHPublicKeySuccess(t *testing.T) {
	key, err := ParseSSHPublicKey(HOST_PUBLIC_KEY)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, "ecdsa-sha2-nistp256", key.Type())
}

func TestParseSSHPublicKeyEmptyString(t *testing.T) {
	key, err := ParseSSHPublicKey("")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.ErrorContains(t, err, "SSH public key string is empty")
}

func TestParseSSHPublicKeyInvalidFormat(t *testing.T) {
	key, err := ParseSSHPublicKey("not-a-valid-ssh-key format")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.ErrorContains(t, err, "Failed to parse host public key")
}

func TestParseSSHPublicKeySingleField(t *testing.T) {
	key, err := ParseSSHPublicKey("ecdsa-sha2-nistp256")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.ErrorContains(t, err, "must have at least key-type and base64 data")
}

func TestParseSSHPublicKeyTypeMismatch(t *testing.T) {
	// The base64 data encodes a valid ecdsa-sha2-nistp256 key, but the prefix says "test".
	// ParseAuthorizedKey would silently accept this; our validation must reject it.
	key, err := ParseSSHPublicKey("test AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHMl5neMPoM8XtWmsrSI1TxGe+iWUwfeA62gW0y7a5SM+vJ7WafERtBZJlWBgfOv+zVEPdUjkO0fnWoVbELLjSI=")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.ErrorContains(t, err, "ssh: authorized keys key type mismatch")
}

func TestParseSSHPublicKeyMissingTypePrefix(t *testing.T) {
	// Valid base64 key data but without the key-type prefix (e.g. user pasted only the base64 part).
	key, err := ParseSSHPublicKey("AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8=")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.ErrorContains(t, err, "must have at least key-type and base64 data")
}

func TestParseSSHPublicKeyDoubleQuoted(t *testing.T) {
	// Key enclosed in double quotes, as might happen when copy-pasting from a JSON or UI field.
	quotedKey := `"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="`
	key, err := ParseSSHPublicKey(quotedKey)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestParseSSHPublicKeySingleQuoted(t *testing.T) {
	// Key enclosed in single quotes.
	quotedKey := `'ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8='`
	key, err := ParseSSHPublicKey(quotedKey)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestParseSSHPublicKeyBase64PartDoubleQuoted(t *testing.T) {
	// Only the base64 key value is enclosed in double quotes, type prefix is correct.
	quotedKey := `ecdsa-sha2-nistp256 "AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="`
	key, err := ParseSSHPublicKey(quotedKey)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestNewStandardSshManagerNilHostPublicKey(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, nil)
	assert.Nil(t, manager)
	assert.ErrorIs(t, err, ErrNoneOfConstructorArgsCanBeEmpty)
}

func Test_runCommand_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{} // This is a trick to avoid initializing SSH key auth method during the unit test

	command := "echo Hello"
	output, err := manager.runCommand(command)

	assert.NoError(t, err)
	assert.Equal(t, command, output)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, command, mockClient.ExecutedCommands[0])
}

func Test_runCommand_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{} // This is a trick to avoid initializing SSH key auth method during the unit test

	output, err := manager.runCommand("custom command")

	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Equal(t, output, "")
}

func Test_runCommand_Success_Missing_Exit(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return command, &gossh.ExitMissingError{}
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{} // This is a trick to avoid initializing SSH key auth method during the unit test

	command := "cloud-init --reboot"
	output, err := manager.runCommand(command)

	assert.NoError(t, err)
	assert.Equal(t, command, output)
}

func TestWriteFileOnRemoteMachine_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	manager.Client = &MockSSHClient{}
	manager.SshKeyPaths = []string{}

	err = manager.WriteFileOnRemoteMachine("", "Lorem impsum", 0744)
	assert.NoError(t, err)
}

func Test_executeRemoteFile_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.executeRemoteFile("custom command", true)
	assert.NoError(t, err)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "sudo custom command")
}

func Test_executeRemoteFile_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.executeRemoteFile("custom command", false)
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "custom command")
}

func Test_removeRemoteFile_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.removeRemoteFile("file", true)
	assert.NoError(t, err)
	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, mockClient.ExecutedCommands[0], "sudo rm file")
}

func Test_removeRemoteFile_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

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
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			return "", nil
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

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
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.createRemoteDir("/tmp/mytestdir")
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[0], "mkdir -p /tmp/mytestdir")
}

func Test_getRandomScriptPath(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)
	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	path, err := manager.getRandomScriptPath()
	assert.NoError(t, err)

	assert.Contains(t, path, "/tmp/fsas-nodedriver/user1-via-ssh-")
	assert.True(t, strings.HasSuffix(path, ".sh"))
}

func Test_removeRemoteDir(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.removeRemoteDir("/tmp/mytestdir", false)
	assert.NoError(t, err)

	assert.Contains(t, mockClient.ExecutedCommands[0], "rmdir /tmp/mytestdir")
}

func Test_RebootCloudInit_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.RebootCloudInit()
	assert.NoError(t, err)

	require.Len(t, mockClient.ExecutedCommands, 1)
	assert.Equal(t, cmdRebootCloudInit, mockClient.ExecutedCommands[0])
}

func Test_RebootCloudInit_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.RebootCloudInit()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
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

	keyPaths := []string{"/path/to/invalid/key1", "/path/to/invalid/key2"}
	mockParser.On("Parse", keyPaths[0]).Return(nil)
	mockParser.On("Parse", keyPaths[1]).Return(nil)

	manager := &StandardSshManager{
		UserName:      "testuser",
		SshKeyPaths:   keyPaths,
		HostPublicKey: sshPub,
		keyParser:     mockParser,
	}

	config := manager.getSshClientConfig()
	assert.NotNil(t, config)
	assert.Len(t, config.Auth, 0)
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
	keyPaths := []string{"/path/to/invalid/key1"}

	// Configure the mock to return a valid signer when Parse is called.
	// This simulates finding and successfully parsing a private key.
	mockParser.On("Parse", keyPaths[0]).Return(signer)

	manager := &StandardSshManager{
		UserName:      "testuser",
		SshKeyPaths:   keyPaths,
		HostPublicKey: sshPub,
		keyParser:     mockParser, // Inject the mock
	}

	config := manager.getSshClientConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "testuser", config.User)
	// We expect two auth methods: password and public key.
	assert.Len(t, config.Auth, 1, "There should be public key authentication")
	assert.Equal(t, []string{sshPub.Type()}, config.HostKeyAlgorithms)

	// Verify that the mock was called as expected.
	mockParser.AssertExpectations(t)
}

func Test_DeregisterOS_Success(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	assert.NoError(t, err)
	emptyProducts := []models.SuseProduct{}
	jsonOutput, _ := json.Marshal(emptyProducts)
	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			if cmd == cmdGetStatusOS {
				return string(jsonOutput), nil
			}
			return "", fmt.Errorf("unexpected command")
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.DeregisterOS()
	assert.NoError(t, err)

	assert.Equal(t, []string{cmdGetStatusOS}, mockClient.ExecutedCommands)
}

func Test_DeregisterOS_Fail(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	assert.NoError(t, err)
	mockClient := &MockSSHClient{
		OutputFunc: func(command string) (string, error) {
			return "", MOCK_ERROR_FOR_OUTPUT_METHOD
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.DeregisterOS()
	assert.ErrorIs(t, err, MOCK_ERROR_FOR_OUTPUT_METHOD)
	assert.Equal(t, []string{cmdGetStatusOS}, mockClient.ExecutedCommands)
}

func Test_DeregisterOS_Skip_When_NoRegisteredProducts(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	assert.NoError(t, err)
	products := []models.SuseProduct{
		{Identifier: "SLES", Version: "15.6", Arch: "x86_64", Status: "Not Registered"},
	}
	jsonOutput, _ := json.Marshal(products)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			if cmd == cmdGetStatusOS {
				return string(jsonOutput), nil
			}
			return "", fmt.Errorf("unexpected command")
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.DeregisterOS()
	assert.NoError(t, err)

	assert.Equal(t, []string{cmdGetStatusOS}, mockClient.ExecutedCommands)
}

func Test_DeregisterOS_Run_WhenRegistered(t *testing.T) {
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	assert.NoError(t, err)
	products := []models.SuseProduct{
		{Identifier: "SLES", Version: "15.6", Arch: "x86_64", Status: "Registered"},
	}
	jsonOutput, _ := json.Marshal(products)

	mockClient := &MockSSHClient{
		OutputFunc: func(cmd string) (string, error) {
			if cmd == cmdGetStatusOS {
				return string(jsonOutput), nil
			}
			if cmd == cmdDeregisterOS {
				return "", nil
			}
			return "", fmt.Errorf("unexpected command")
		},
	}
	manager.Client = mockClient
	manager.SshKeyPaths = []string{}

	err = manager.DeregisterOS()
	assert.NoError(t, err)

	assert.Equal(t, []string{cmdGetStatusOS, cmdDeregisterOS}, mockClient.ExecutedCommands)
}

func TestHostPublicKeyIsValid_AlreadyValid(t *testing.T) {
	parsedKey := parsedHostPublicKey(t)
	manager, err := NewStandardSshManager("host1", "user1", []string{"mock/path1", "mock/path2"}, parsedKey)
	require.NoError(t, err)

	oldIsPublicKeyValid := IsPublicKeyValid
	defer func() {
		IsPublicKeyValid = oldIsPublicKeyValid
	}()

	IsPublicKeyValid = false
	manager.sshDial = sshDialFunc(func(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
		assert.Equal(t, "tcp", network)
		assert.Equal(t, "host1:22", addr)
		assert.Equal(t, "user1", config.User)
		require.NoError(t, config.HostKeyCallback("host1", nil, parsedKey))
		return nil, fmt.Errorf("handshake terminated after host key verification")
	})

	err = manager.HostPublicKeyIsValid(1)
	assert.NoError(t, err)
	assert.True(t, IsPublicKeyValid)
}

func TestHostPublicKeyIsValid_OtherDialError_ReturnsWrappedError(t *testing.T) {
	IsPublicKeyValid = false
	oldIsPublicKeyValid := IsPublicKeyValid
	defer func() {
		IsPublicKeyValid = oldIsPublicKeyValid
	}()

	manager, err := NewStandardSshManager("nonexistent-host.invalid", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	manager.sshDial = sshDialFunc(func(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
		assert.Equal(t, "tcp", network)
		assert.Equal(t, "nonexistent-host.invalid:22", addr)
		assert.Equal(t, "user1", config.User)
		return nil, fmt.Errorf("lookup nonexistent-host.invalid: no such host")
	})

	err = manager.HostPublicKeyIsValid(1)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to verify SSH host key")
	assert.ErrorContains(t, err, "lookup nonexistent-host.invalid: no such host")
}

func TestHostPublicKeyIsValid_ZeroMaxAttempts_NormalizedToOne(t *testing.T) {
	IsPublicKeyValid = false
	oldIsPublicKeyValid := IsPublicKeyValid
	defer func() {
		IsPublicKeyValid = oldIsPublicKeyValid
	}()

	manager, err := NewStandardSshManager("nonexistent-host.invalid", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	dialCalls := 0
	manager.sshDial = sshDialFunc(func(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
		dialCalls++
		assert.Equal(t, "tcp", network)
		assert.Equal(t, "nonexistent-host.invalid:22", addr)
		assert.Equal(t, "user1", config.User)
		return nil, fmt.Errorf("lookup nonexistent-host.invalid: no such host")
	})

	err = manager.HostPublicKeyIsValid(0)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to verify SSH host key")
	assert.Equal(t, 1, dialCalls)
}

func TestHostPublicKeyIsValid_ConnectionRefused_ReturnsError(t *testing.T) {
	IsPublicKeyValid = false
	oldIsPublicKeyValid := IsPublicKeyValid
	defer func() {
		IsPublicKeyValid = oldIsPublicKeyValid
	}()

	manager, err := NewStandardSshManager("127.0.0.1", "user1", []string{"mock/path1", "mock/path2"}, parsedHostPublicKey(t))
	require.NoError(t, err)

	manager.sshDial = sshDialFunc(func(network, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
		assert.Equal(t, "tcp", network)
		assert.Equal(t, "127.0.0.1:22", addr)
		assert.Equal(t, "user1", config.User)
		return nil, fmt.Errorf("dial tcp 127.0.0.1:22: connect: connection refused")
	})

	err = manager.HostPublicKeyIsValid(1)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to verify SSH host key")
	assert.ErrorContains(t, err, "connection refused")
}
