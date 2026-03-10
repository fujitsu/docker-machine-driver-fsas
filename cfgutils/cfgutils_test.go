package cfgutils

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIsInit_Fail(t *testing.T) {
	manager := &StandardCfgManager{}
	observed := manager.IsInit()
	assert.Equal(t, false, observed)
}

func TestIsInit_Success(t *testing.T) {
	manager := NewStandardCfgManager("[]", "")
	observed := manager.IsInit()
	assert.Equal(t, true, observed)
}

func TestPrepareMetadata(t *testing.T) {

	testCases := []struct {
		instanceId string
		hostname   string
		expected   string
	}{
		{instanceId: "12345678-1234-1234-1234-123456789012", hostname: "host1",
			expected: `dsmode: local
instance-id: 12345678-1234-1234-1234-123456789012
hostname: host1`,
		},
		{instanceId: "12345678-1234-1234-1234-123456789012", hostname: "",
			expected: `dsmode: local
instance-id: 12345678-1234-1234-1234-123456789012
hostname: `,
		},
		{instanceId: "", hostname: "host1", expected: "dsmode: local\ninstance-id: \nhostname: host1"},
		{instanceId: "", hostname: "", expected: "dsmode: local\ninstance-id: \nhostname: "},
	}

	for _, tc := range testCases {
		t.Run(tc.hostname, func(t *testing.T) {
			manager := NewStandardCfgManager("[]", "")
			observed := manager.PrepareMetadata(tc.instanceId, tc.hostname)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func Test_prepareRke2ConfigProviderId(t *testing.T) {
	testCases := []struct {
		machineUUID string
		expected    string
	}{
		{machineUUID: "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
			expected: `kubelet-arg+: "provider-id=fsas-cdi://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"`},
		{machineUUID: "",
			expected: `kubelet-arg+: "provider-id=fsas-cdi://"`},
	}

	manager := NewStandardCfgManager("[]", "")

	for _, tc := range testCases {
		t.Run(tc.machineUUID, func(t *testing.T) {
			observed := manager.prepareRke2ConfigProviderId(tc.machineUUID)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func Test_prepareRke2ConfigNodeLabelsForGpu(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{name: "no GPU resources",
			expected: ""},
	}

	manager := NewStandardCfgManager("[]", "")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observed := manager.prepareRke2ConfigNodeLabelsForGpu()
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func Test_prepareRke2ConfigNodeLabels_Dynamic(t *testing.T) {
	devicesSpecJson := `[
		{
			"res_type": "gpu",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{"column": "model", "operator": "eq", "value": "a100"}
				]
			},
			"min_resource_count": 1
		},
		{
			"res_type": "gpu",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{"column": "model", "operator": "eq", "value": "L40S"}
				]
			},
			"min_resource_count": 2,
			"max_resource_count": 3
		}
	]`

	manager := NewStandardCfgManager(devicesSpecJson, "")

	labelStr := manager.prepareRke2ConfigNodeLabelsForGpu()
	expected := `kubelet-arg+: "node-labels=cohdi.io/nvidia-l40s-size-min=2,cohdi.io/nvidia-l40s-size-max=3"`
	assert.Equal(t, expected, labelStr)
}

func TestPrepareRke2ConfigScript(t *testing.T) {
	configName := "100-kubelet-provider-id"
	testCases := []struct {
		machineUUID string
		expected    string
	}{
		{machineUUID: "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas-cdi://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"`),
		},

		{machineUUID: "1234",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas-cdi://1234"`),
		},

		{machineUUID: "",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas-cdi://"`),
		},
	}

	manager := NewStandardCfgManager("[]", "")

	for _, tc := range testCases {
		t.Run(tc.machineUUID, func(t *testing.T) {
			observed := manager.PrepareRke2ConfigScript(configName, tc.machineUUID)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func TestPrepareRke2ConfigScript_WithGPUResources(t *testing.T) {
	devicesSpecJson := `[
		{
			"res_type": "gpu",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{"column": "model", "operator": "eq", "value": "L40S"}
				]
			},
			"min_resource_count": 1,
			"max_resource_count": 2
		}
	]`

	manager := NewStandardCfgManager(devicesSpecJson, "")

	configName := "100-gpu-labels"
	script := manager.PrepareRke2ConfigScript(configName, "my-machine-uuid")

	expected := fmt.Sprintf(rke2ConfigScriptContent, configName,
		`kubelet-arg+: "provider-id=fsas-cdi://my-machine-uuid"
kubelet-arg+: "node-labels=cohdi.io/nvidia-l40s-size-min=1,cohdi.io/nvidia-l40s-size-max=2"`)
	assert.Equal(t, expected, script)
}

func Test_prepareRke2ConfigNodeLabels_FromExactJSON(t *testing.T) {
	devicesSpecJson := `testJson`

	var resources []models.Resource
	if err := json.Unmarshal([]byte(devicesSpecJson), &resources); err != nil {
		t.Logf("Failed to unmarshal JSON: %v", err)
	}

	manager := NewStandardCfgManager(devicesSpecJson, "")

	labels := manager.prepareRke2ConfigNodeLabelsForGpu()
	t.Logf("Generated GPU label: %s", labels)
}

func TestExtendUserdata(t *testing.T) {
	testCases := []struct {
		name              string
		readFileContent   []byte
		input             []CloudConfigItem
		expectedStr       string
		nrExpectedItemsWF int
		nrExpectedItemsRC int
		expectedError     error
	}{
		{name: "case 1: add 1 item to section 'runcmd'",
			readFileContent:   []byte(userdataSampleContentBothSections),
			input:             input1ItemRunCmdCast,
			expectedStr:       expectedStr2Cmd1Write,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 2: add 1 item to section 'runcmd' and 1 item to 'write_files'",
			readFileContent:   []byte(userdataSampleContentBothSections),
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr2Cmd2Write,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 2,
			expectedError:     nil,
		},

		{name: "case 3: add 2 items to section 'runcmd' and 2 items to 'write_files'",
			readFileContent:   []byte(userdataSampleContentBothSections),
			input:             input2ItemsRunCmdCast2ItemsWriteFiles,
			expectedStr:       expectedStr3Cmd3Write,
			nrExpectedItemsRC: 3,
			nrExpectedItemsWF: 3,
			expectedError:     nil,
		},

		{name: "case 4: no section 'runcmd' available section 'write_files' 1 item cmd, 1 item write",
			readFileContent:   []byte(userdataSampleContentCmdNoWriteYes),
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr1Cmd2Write,
			nrExpectedItemsRC: 1,
			nrExpectedItemsWF: 2,
			expectedError:     nil,
		},

		{name: "case 5: no section 'write_files' available section 'runcmd' 1 item cmd, 1 item write",
			readFileContent:   []byte(userdataSampleContentCmdYesWriteNo),
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr2Cmd1WriteBis,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 6: no section 'write_files' neither 'runcmd' 1 item cmd, 1 item write",
			readFileContent:   []byte(userdataSampleContentNoSections),
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr1Cmd1Write,
			nrExpectedItemsRC: 1,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 7: input as empty list",
			readFileContent:   []byte(userdataSampleContentBothSections),
			input:             []CloudConfigItem{},
			expectedStr:       userdataSampleContentBothSections,
			nrExpectedItemsWF: 1,
			nrExpectedItemsRC: 1,
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		var expected, observed map[string][]any
		t.Run(tc.name, func(t *testing.T) {

			tempFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := tempFile.Close()
				require.NoError(t, err, "Failed to close temp file")
				err = os.Remove(tempFile.Name())
				require.NoError(t, err, "Failed to delete temp file")
			}()

			if _, err := tempFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sc := NewStandardCfgManager("[]", tempFile.Name())
			err = sc.extendUserdata(tc.input)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError,
					fmt.Sprintf("expected: %v, but got: %v", tc.expectedError, err))
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail.
				   Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				fileContent, err := os.ReadFile(tempFile.Name())
				require.NoError(t, err, "Failed to read from temp file")
				if err := yaml.Unmarshal(fileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, tc.nrExpectedItemsRC, len(observed["runcmd"]))
				assert.Equal(t, tc.nrExpectedItemsWF, len(observed["write_files"]))
			}

		})
	}

}

func TestExtendUserdataRunCmd(t *testing.T) {
	testCases := []struct {
		name            string
		readFileContent []byte
		input           []string
		expectedStr     string
		nrExpectedItems int
		expectedError   error
	}{
		{name: "case 1: add one item to section 'runcmd'",
			readFileContent: []byte(userdataSampleContent),
			input:           inputOneItemRunCmd,
			expectedStr:     expectedStr2Cmd,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 2: add two items to section 'runcmd'",
			readFileContent: []byte(userdataSampleContent),
			input:           inputTwoItemsRunCmd,
			expectedStr:     expectedStr3Cmd,
			nrExpectedItems: 3,
			expectedError:   nil,
		},

		{name: "case 3: section 'runcmd' does not exist",
			readFileContent: []byte(userdataSampleContentNoSections),
			input:           inputOneItemRunCmd,
			expectedStr:     expectedStr1Cmd,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 4: no content in userdata file",
			readFileContent: []byte{},
			input:           inputOneItemRunCmd,
			expectedStr:     expectedStr1Cmd,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 5: input as empty list",
			readFileContent: []byte(userdataSampleContent),
			input:           nil,
			expectedStr:     userdataSampleContent,
			nrExpectedItems: 1,
			expectedError:   nil,
		},
	}

	for _, tc := range testCases {
		var expected, observed map[string][]any
		t.Run(tc.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := tempFile.Close()
				require.NoError(t, err, "Failed to close temp file")
				err = os.Remove(tempFile.Name())
				require.NoError(t, err, "Failed to delete temp file")
			}()

			if _, err := tempFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sc := NewStandardCfgManager("[]", tempFile.Name())

			err = sc.ExtendUserdataRunCmd(tc.input)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError,
					fmt.Sprintf("expected: %v, but got: %v", tc.expectedError, err))
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail.
				   Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				fileContent, err := os.ReadFile(tempFile.Name())
				require.NoError(t, err, "Failed to read from temp file")
				if err := yaml.Unmarshal(fileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, tc.nrExpectedItems, len(observed["runcmd"]))
			}

		})
	}
}

func TestExtendUserdataRunCmd_YamlUnmarshalingError(t *testing.T) {

	testCases := []struct {
		name             string
		readFileContent  []byte
		expectedErrorStr []string
	}{
		{name: "case 1: invalid yaml file - random ascii chars",
			readFileContent: []byte(userdataSampleInvalidYamlContentRandomAscii),
			expectedErrorStr: []string{
				"yaml: unmarshal errors",
				"line 1: cannot unmarshal !!str",
			},
		},
		{name: "case 2: invalid yaml file - runcmd is not list but integer",
			readFileContent: []byte(userdataSampleInvalidYamlContentRunCmdIsInteger),
			expectedErrorStr: []string{
				"module runcmd exists but is not a list",
			},
		},
		{name: "case 3: invalid yaml file - runcmd is not list but string",
			readFileContent: []byte(userdataSampleInvalidYamlContentRunCmdIsString),
			expectedErrorStr: []string{
				"module runcmd exists but is not a list",
			},
		},
		{name: "case 4: invalid yaml file - runcmd is not list but bool",
			readFileContent: []byte(userdataSampleInvalidYamlContentRunCmdIsBool),
			expectedErrorStr: []string{
				"module runcmd exists but is not a list",
			},
		},
		{name: "case 5: invalid yaml file - runcmd is not list but map",
			readFileContent: []byte(userdataSampleInvalidYamlContentRunCmdIsMap),
			expectedErrorStr: []string{
				"module runcmd exists but is not a list",
			},
		},
		{name: "case 6: invalid yaml file - runcmd is not list but nil",
			readFileContent: []byte(userdataSampleInvalidYamlContentRunCmdIsNil),
			expectedErrorStr: []string{
				"module runcmd exists but is not a list",
			},
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := tempFile.Close()
				require.NoError(t, err, "Failed to close temp file")
				err = os.Remove(tempFile.Name())
				require.NoError(t, err, "Failed to delete temp file")
			}()

			if _, err := tempFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sc := NewStandardCfgManager("[]", tempFile.Name())
			err = sc.extendUserdata(input1ItemRunCmdCast1ItemWriteFiles)

			if err == nil {
				t.Fatal("expected error but got nil")
			} else {
				for _, errMsg := range tc.expectedErrorStr {
					assert.Contains(t, err.Error(), errMsg)
				}
			}

		})
	}
}

func TestExtendUserdataWriteFiles(t *testing.T) {

	inputOneItemWriteFilesExe := []CloudConfigItem{
		NewCloudConfigItemWriteFiles("/tmp/run.sh", "#!/bin/bash",
			SetCustomPermissions(os.FileMode(0744)))}

	inputOneItemWriteFilesSetPermissions := []CloudConfigItem{
		NewCloudConfigItemWriteFiles("/tmp/cdi.cert", "###begin cert",
			SetCustomPermissions(os.FileMode(0400)))}

	testCases := []struct {
		name            string
		readFileContent []byte
		input           []CloudConfigItem
		expectedStr     string
		nrExpectedItems int
		expectedError   error
	}{

		{name: "case 1: add one item to section 'write_files'",
			readFileContent: []byte(userdataSampleContentWriteFiles),
			input:           inputOneItemWriteFiles,
			expectedStr:     expectedStr2Write,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 2: add two items to section 'write_files'",
			readFileContent: []byte(userdataSampleContentWriteFiles),
			input:           inputTwoItemsWriteFiles,
			expectedStr:     expectedStr3Write,
			nrExpectedItems: 3,
			expectedError:   nil,
		},

		{name: "case 3: section 'write_files' does not exist",
			readFileContent: []byte(userdataSampleContentNoSections),
			input:           inputOneItemWriteFiles,
			expectedStr:     expectedStr1Write,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 4: input as empty list",
			readFileContent: []byte(userdataSampleContentWriteFiles),
			input:           []CloudConfigItem{},
			expectedStr:     userdataSampleContentWriteFiles,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 5: add one item to section 'write_files' with executable attribute ",
			readFileContent: []byte(userdataSampleContentWriteFiles),
			input:           inputOneItemWriteFilesExe,
			expectedStr:     expectedStr2WriteExe,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 6: add one item to section 'write_files' with custom permissions ",
			readFileContent: []byte(userdataSampleContentWriteFiles),
			input:           inputOneItemWriteFilesSetPermissions,
			expectedStr:     expectedStr2WriteSetPermissions,
			nrExpectedItems: 2,
			expectedError:   nil,
		},
	}

	for _, tc := range testCases {
		var expected, observed map[string][]any
		t.Run(tc.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := tempFile.Close()
				require.NoError(t, err, "Failed to close temp file")
				err = os.Remove(tempFile.Name())
				require.NoError(t, err, "Failed to delete temp file")
			}()

			if _, err := tempFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sc := NewStandardCfgManager("[]", tempFile.Name())

			err = sc.ExtendUserdataWriteFiles(tc.input)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError,
					fmt.Sprintf("expected: %v, but got: %v", tc.expectedError, err))
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail.
				   Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				fileContent, err := os.ReadFile(tempFile.Name())
				require.NoError(t, err, "Failed to read from temp file")
				if err := yaml.Unmarshal(fileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, tc.nrExpectedItems, len(observed["write_files"]))
			}

		})
	}

}

func Test_userdataFile_not_exists(t *testing.T) {

	testCases := []struct {
		name           string
		testedFunction func() error
	}{

		{name: "case 1: method 'extendUserdata'",
			testedFunction: func() error {
				sc := NewStandardCfgManager("[]", "some-non-existing-file")
				err := sc.extendUserdata([]CloudConfigItem{})
				return err
			},
		},
		{name: "case 2: method 'ExtendUserdataWriteFiles'",
			testedFunction: func() error {
				sc := NewStandardCfgManager("[]", "some-non-existing-file")
				err := sc.ExtendUserdataWriteFiles([]CloudConfigItem{})
				return err
			},
		},
		{name: "case 3: method 'ExtendUserdataRunCmd'",
			testedFunction: func() error {
				sc := NewStandardCfgManager("[]", "some-non-existing-file")
				err := sc.ExtendUserdataRunCmd([]string{})
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.testedFunction()
			if err == nil {
				t.Fatal("expected error bot got nil")
			} else {
				assert.ErrorIs(t, err, fs.ErrNotExist,
					fmt.Sprintf("expected: %v, but got: %v", fs.ErrNotExist, err))
			}
		})
	}

}

func TestExtendUserdataSshAuthKeys(t *testing.T) {
	testCases := []struct {
		name            string
		readFileContent []byte
		input           []string
		expectedStr     string
		nrExpectedItems int
		expectedError   error
	}{
		{name: "case 1: add one item to section 'ssh_authorized_keys'",
			readFileContent: []byte(userdataSampleContentSsh),
			input:           inputOneItemSshAuthKeys,
			expectedStr:     expectedStr2Ssh,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 2: add two items to section 'ssh_authorized_keys'",
			readFileContent: []byte(userdataSampleContentSsh),
			input:           inputTwoItemsSshAuthKeys,
			expectedStr:     expectedStr3Ssh,
			nrExpectedItems: 3,
			expectedError:   nil,
		},

		{name: "case 3: section 'ssh_authorized_keys' does not exist",
			readFileContent: []byte(userdataSampleContentNoSections),
			input:           inputOneItemSshAuthKeys,
			expectedStr:     expectedStr1Ssh,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 4: no content in userdata file",
			readFileContent: []byte{},
			input:           inputOneItemSshAuthKeys,
			expectedStr:     expectedStr1Ssh,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 5: input as empty list",
			readFileContent: []byte(userdataSampleContentSsh),
			input:           nil,
			expectedStr:     userdataSampleContentSsh,
			nrExpectedItems: 1,
			expectedError:   nil,
		},
	}

	for _, tc := range testCases {
		var expected, observed map[string][]any
		t.Run(tc.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := tempFile.Close()
				require.NoError(t, err, "Failed to close temp file")
			}()

			if _, err := tempFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sc := NewStandardCfgManager("[]", tempFile.Name())

			err = sc.ExtendUserdataSshAuthKeys(tc.input)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError,
					fmt.Sprintf("expected: %v, but got: %v", tc.expectedError, err))
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail.
				   Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				fileContent, err := os.ReadFile(tempFile.Name())
				require.NoError(t, err, "Failed to read from temp file")
				if err := yaml.Unmarshal(fileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, tc.nrExpectedItems, len(observed["ssh_authorized_keys"]))
			}

		})
	}
}

func TestImplantSSHKey(t *testing.T) {
	testCases := []struct {
		name              string
		readFileContent   []byte
		nrItemsRunCmd     int
		nrItemsWriteFiles int
		expectedError     error
	}{
		{name: "case 1: empty cloud-init file",
			readFileContent:   []byte(userdataSampleContentNoSections),
			nrItemsRunCmd:     0,
			nrItemsWriteFiles: 0,
			expectedError:     nil,
		},

		{name: "case 2: cloud-init file with single section 'runcmd'",
			readFileContent:   []byte(userdataSampleContent),
			nrItemsRunCmd:     1,
			nrItemsWriteFiles: 0,
			expectedError:     nil,
		},

		{name: "case 3: cloud-init file with single section 'write_files'",
			readFileContent:   []byte(userdataSampleContentWriteFiles),
			nrItemsRunCmd:     0,
			nrItemsWriteFiles: 1,
			expectedError:     nil,
		},

		{name: "case 4: cloud-init file with both sections 'runcmd' and 'write_files'",
			readFileContent:   []byte(userdataSampleContentBothSections),
			nrItemsRunCmd:     1,
			nrItemsWriteFiles: 1,
			expectedError:     nil,
		},
	}

	sshUser := "rancher"

	for _, tc := range testCases {
		var observed map[string][]any
		t.Run(tc.name, func(t *testing.T) {
			userdataFile, err := os.CreateTemp(t.TempDir(), "userdata.yaml")
			require.NoError(t, err, "Failed to create temp file")
			defer func() {
				err := userdataFile.Close()
				require.NoError(t, err, "Failed to close temp file")
				err = os.Remove(userdataFile.Name())
				require.NoError(t, err, "Failed to delete temp file")
			}()

			if _, err := userdataFile.WriteString(string(tc.readFileContent)); err != nil {
				require.NoError(t, err, "Failed to write to temp file")
			}

			sshPrivKeyPath := filepath.Join(t.TempDir(), "id_rsa")
			defer func() {
				err = os.Remove(sshPrivKeyPath)
				require.NoError(t, err, "Failed to delete temp file with ssh private key")
				err = os.Remove(fmt.Sprintf("%s.pub", sshPrivKeyPath))
				require.NoError(t, err, "Failed to delete temp file with ssh public key")
			}()

			sc := NewStandardCfgManager("[]", userdataFile.Name())
			err = sc.ImplantSSHKey(sshPrivKeyPath, sshUser)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError,
					fmt.Sprintf("expected: %v, but got: %v", tc.expectedError, err))
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail.
				   Thus compare YAML semantically and not textually.
				*/
				fileContent, err := os.ReadFile(userdataFile.Name())
				require.NoError(t, err, "Failed to read from temp file")
				if err := yaml.Unmarshal(fileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				username := observed["users"][0].(map[string]any)["name"].(string)
				assert.Equal(t, sshUser, username)

				assert.NotNil(t, observed["users"])
				assert.Equal(t, len(observed["users"]), 1, "Expected exactly one user in the cloud-init config")
				userMap := observed["users"][0].(map[string]any)

				assert.NotNil(t, userMap["ssh_authorized_keys"])
				sshKeys := userMap["ssh_authorized_keys"].([]any)
				assert.Equal(t, len(sshKeys), 1)

				if _, ok := observed["runcmd"]; ok {
					assert.Equal(t, len(observed["runcmd"]), tc.nrItemsRunCmd, "Number of items differ for 'runcmd'")
				}

				if _, ok := observed["write_files"]; ok {
					assert.Equal(t, len(observed["write_files"]), tc.nrItemsWriteFiles, "Number of items differ for 'write_files'")
				}
			}

		})
	}
}
