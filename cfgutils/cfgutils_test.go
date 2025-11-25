package cfgutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/stretchr/testify/assert"
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
			expected: `kubelet-arg+: "provider-id=fsas://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"`},
		{machineUUID: "",
			expected: `kubelet-arg+: "provider-id=fsas://"`},
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
					{"column": "model", "operator": "eq", "value": "a100-40g"}
				]
			},
			"min_resource_count": 1
		},
		{
			"res_type": "gpu",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{"column": "model", "operator": "eq", "value": "nvidia-h100"}
				]
			},
			"min_resource_count": 2,
			"max_resource_count": 3
		}
	]`

	manager := NewStandardCfgManager(devicesSpecJson, "")

	labelStr := manager.prepareRke2ConfigNodeLabelsForGpu()

	expected := `kubelet-arg+: "node-labels=cohdi.io/nvidia-h100-size-min=2,cohdi.io/nvidia-h100-size-max=3"`

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
				`kubelet-arg+: "provider-id=fsas://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"`),
		},
		{machineUUID: "1234",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas://1234"`),
		},
		{machineUUID: "",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas://"`),
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
					{"column": "model", "operator": "eq", "value": "a100-40g"}
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
		`kubelet-arg+: "provider-id=fsas://my-machine-uuid"
kubelet-arg+: "node-labels=cohdi.io/nvidia-a100-40g-size-min=1,cohdi.io/nvidia-a100-40g-size-max=2"`)

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

var (
	osStatErrorMessage = ""
	osStatMock         = func(name string) (os.FileInfo, error) {
		if osStatErrorMessage != "" {
			if osStatErrorMessage == "no such file" {
				return nil, &os.PathError{
					Op:   "stat",
					Path: name,
					Err:  fs.ErrNotExist,
				}
			} else {
				return nil, errors.New(osStatErrorMessage)
			}
		} else {
			return nil, nil
		}
	}
	mockOsReadFileContent  = []byte{}
	osReadFileMock         = func(name string) ([]byte, error) { return mockOsReadFileContent, nil }
	mockOsWriteFileContent = []byte{}
	osWriteFileMock        = func(name string, data []byte, perm os.FileMode) error {
		mockOsWriteFileContent = data
		return nil
	}
)

func resetOsMocks(userdataContent string) {
	osStatErrorMessage = ""
	osStatMock = func(name string) (os.FileInfo, error) {
		if osStatErrorMessage != "" {
			if osStatErrorMessage == "no such file" {
				return nil, &os.PathError{
					Op:   "stat",
					Path: name,
					Err:  fs.ErrNotExist,
				}
			} else {
				return nil, errors.New(osStatErrorMessage)
			}
		} else {
			return nil, nil
		}
	}
	mockOsReadFileContent = []byte{}
	osReadFileMock = func(name string) ([]byte, error) { return mockOsReadFileContent, nil }
	mockOsWriteFileContent = []byte{}
	osWriteFileMock = func(name string, data []byte, perm os.FileMode) error {
		mockOsWriteFileContent = data
		return nil
	}

	osStat = osStatMock
	mockOsReadFileContent = []byte(userdataContent)
	osReadFile = osReadFileMock
	osWriteFile = osWriteFileMock

}

func TestExtendUserdataRunCmd(t *testing.T) {
	sc := NewStandardCfgManager("", "/tmp/userdata.yaml")

	testCases := []struct {
		action          func()
		name            string
		input           []string
		expectedStr     string
		nrExpectedItems int
		expectedError   error
	}{
		{name: "case 1: input as empty list",
			action:          func() { resetOsMocks(userdataSampleContent) },
			input:           []string{},
			expectedStr:     userdataSampleContent,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 2: add one item to section 'runcmd'",
			action:          func() { resetOsMocks(userdataSampleContent) },
			input:           inputOneItemRunCmd,
			expectedStr:     expectedStr2Cmd,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 3: add two items to section 'runcmd'",
			action:          func() { resetOsMocks(userdataSampleContent) },
			input:           inputTwoItemsRunCmd,
			expectedStr:     expectedStr3Cmd,
			nrExpectedItems: 3,
			expectedError:   nil,
		},

		{name: "case 4: section 'runcmd' does not exist",
			action: func() {
				resetOsMocks(userdataSampleContentNoSections)
			},
			input:           inputOneItemRunCmd,
			expectedStr:     expectedStr1Cmd,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 5: no usedata file",
			action: func() {
				resetOsMocks(userdataSampleContent)
				osStatErrorMessage = "no such file"
			},
			input:           nil,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   fs.ErrNotExist,
		},

		{name: "case 6: error while reading from usedata file",
			action: func() {
				resetOsMocks(userdataSampleContent)
				osReadFileMock = func(name string) ([]byte, error) { return []byte{}, expectedErrorReadingFromFile }
				osReadFile = osReadFileMock
			},
			input:           nil,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   expectedErrorReadingFromFile,
		},

		{name: "case 7: error while writing to usedata file",
			action: func() {
				resetOsMocks(userdataSampleContent)
				osWriteFileMock = func(name string, data []byte, perm os.FileMode) error {
					mockOsWriteFileContent = nil
					return expectedErrorWritingToFile
				}
				osWriteFile = osWriteFileMock
			},
			input:           inputOneItemRunCmd,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   expectedErrorWritingToFile,
		},
	}

	var expected, observed map[string][]any

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.action != nil {
				tc.action()
			}
			err := sc.ExtendUserdataRunCmd(tc.input)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected: %v, but got: %v", tc.expectedError, err)
				}
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail. Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				if err := yaml.Unmarshal(mockOsWriteFileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, len(observed["runcmd"]), tc.nrExpectedItems)
			}
		})
	}

}

func TestExtendUserdataRunCmd_YamlUnmarshallinError(t *testing.T) {
	sc := NewStandardCfgManager("", "/tmp/userdata.yaml")

	resetOsMocks(userdataSampleContent)
	mockOsReadFileContent = []byte(userdataSampleInvalidYamlContent)
	err := sc.ExtendUserdataRunCmd(inputOneItemRunCmd)
	if err == nil {
		t.Fatal("expected error, but got nil")
	}

	expectedErrMsg := []string{
		"yaml: unmarshal errors",
		"line 1: cannot unmarshal !!str",
	}

	for _, errMsg := range expectedErrMsg {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func TestExtendUserdataWriteFiles(t *testing.T) {
	sc := NewStandardCfgManager("", "/tmp/userdata.yaml")

	testCases := []struct {
		action          func()
		name            string
		input           []CloudConfigItem
		expectedStr     string
		nrExpectedItems int
		expectedError   error
	}{
		{name: "case 1: input as empty list",
			action:          func() { resetOsMocks(userdataSampleContentWriteFiles) },
			input:           []CloudConfigItem{},
			expectedStr:     userdataSampleContentWriteFiles,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 2: add one item to section 'write_files'",
			action:          func() { resetOsMocks(userdataSampleContentWriteFiles) },
			input:           inputOneItemWriteFiles,
			expectedStr:     expectedStr2Write,
			nrExpectedItems: 2,
			expectedError:   nil,
		},

		{name: "case 3: add two items to section 'write_files'",
			action:          func() { resetOsMocks(userdataSampleContentWriteFiles) },
			input:           inputTwoItemsWriteFiles,
			expectedStr:     expectedStr3Write,
			nrExpectedItems: 3,
			expectedError:   nil,
		},

		{name: "case 4: section 'write_files' does not exist",
			action:          func() { resetOsMocks(userdataSampleContentNoSections) },
			input:           inputOneItemWriteFiles,
			expectedStr:     expectedStr1Write,
			nrExpectedItems: 1,
			expectedError:   nil,
		},

		{name: "case 5: no usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentWriteFiles)
				osStatErrorMessage = "no such file"
			},
			input:           nil,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   fs.ErrNotExist,
		},

		{name: "case 6: error while reading from usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentWriteFiles)
				osReadFileMock = func(name string) ([]byte, error) { return []byte{}, expectedErrorReadingFromFile }
				osReadFile = osReadFileMock
			},
			input:           nil,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   expectedErrorReadingFromFile,
		},

		{name: "case 7: error while writing to usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentWriteFiles)
				osWriteFileMock = func(name string, data []byte, perm os.FileMode) error {
					mockOsWriteFileContent = nil
					return expectedErrorWritingToFile
				}
				osWriteFile = osWriteFileMock
			},
			input:           inputOneItemWriteFiles,
			expectedStr:     "",
			nrExpectedItems: 0,
			expectedError:   expectedErrorWritingToFile,
		},
	}

	var expected, observed map[string][]any

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.action != nil {
				tc.action()
			}
			err := sc.ExtendUserdataWriteFiles(tc.input)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected: %v, but got: %v", tc.expectedError, err)
				}
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail. Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				if err := yaml.Unmarshal(mockOsWriteFileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, len(observed["write_files"]), tc.nrExpectedItems)
			}
		})
	}

}

func TestExtendUserdata(t *testing.T) {
	sc := NewStandardCfgManager("", "/tmp/userdata.yaml")

	testCases := []struct {
		action            func()
		name              string
		input             []CloudConfigItem
		expectedStr       string
		nrExpectedItemsWF int
		nrExpectedItemsRC int
		expectedError     error
	}{
		{name: "case 1: input as empty list",
			action:            func() { resetOsMocks(userdataSampleContentBothSections) },
			input:             []CloudConfigItem{},
			expectedStr:       userdataSampleContentBothSections,
			nrExpectedItemsWF: 1,
			nrExpectedItemsRC: 1,
			expectedError:     nil,
		},

		{name: "case 2: add 1 item to section 'runcmd'",
			action:            func() { resetOsMocks(userdataSampleContentBothSections) },
			input:             input1ItemRunCmdCast,
			expectedStr:       expectedStr2Cmd1Write,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 3: add 1 item to section 'runcmd' and 1 item to 'write_files'",
			action:            func() { resetOsMocks(userdataSampleContentBothSections) },
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr2Cmd2Write,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 2,
			expectedError:     nil,
		},

		{name: "case 4: add 2 items to section 'runcmd' and 2 items to 'write_files'",
			action:            func() { resetOsMocks(userdataSampleContentBothSections) },
			input:             input2ItemsRunCmdCast2ItemsWriteFiles,
			expectedStr:       expectedStr3Cmd3Write,
			nrExpectedItemsRC: 3,
			nrExpectedItemsWF: 3,
			expectedError:     nil,
		},

		{name: "case 5: no section 'runcmd' available section 'write_files' 1 item cmd, 1 item write",
			action:            func() { resetOsMocks(userdataSampleContentCmdNoWriteYes) },
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr1Cmd2Write,
			nrExpectedItemsRC: 1,
			nrExpectedItemsWF: 2,
			expectedError:     nil,
		},

		{name: "case 6: no section 'write_files' available section 'runcmd' 1 item cmd, 1 item write",
			action:            func() { resetOsMocks(userdataSampleContentCmdYesWriteNo) },
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr2Cmd1WriteBis,
			nrExpectedItemsRC: 2,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 7: no section 'write_files' neither 'runcmd' 1 item cmd, 1 item write",
			action:            func() { resetOsMocks(userdataSampleContentNoSections) },
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       expectedStr1Cmd1Write,
			nrExpectedItemsRC: 1,
			nrExpectedItemsWF: 1,
			expectedError:     nil,
		},

		{name: "case 8: no usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentNoSections)
				osStatErrorMessage = "no such file"
			},
			input:             nil,
			expectedStr:       "",
			nrExpectedItemsRC: 0,
			nrExpectedItemsWF: 0,
			expectedError:     fs.ErrNotExist,
		},

		{name: "case 9: error while reading from usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentNoSections)
				osReadFileMock = func(name string) ([]byte, error) { return []byte{}, expectedErrorReadingFromFile }
				osReadFile = osReadFileMock
			},
			input:             nil,
			expectedStr:       "",
			nrExpectedItemsRC: 0,
			nrExpectedItemsWF: 0,
			expectedError:     expectedErrorReadingFromFile,
		},

		{name: "case 10: error while writing to usedata file",
			action: func() {
				resetOsMocks(userdataSampleContentNoSections)
				osWriteFileMock = func(name string, data []byte, perm os.FileMode) error {
					mockOsWriteFileContent = nil
					return expectedErrorWritingToFile
				}
				osWriteFile = osWriteFileMock
			},
			input:             input1ItemRunCmdCast1ItemWriteFiles,
			expectedStr:       "",
			nrExpectedItemsRC: 0,
			nrExpectedItemsWF: 0,
			expectedError:     expectedErrorWritingToFile,
		},
	}

	var expected, observed map[string][]any

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.action != nil {
				tc.action()
			}
			err := sc.extendUserdata(tc.input)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected: %v, but got: %v", tc.expectedError, err)
				}
			} else {

				/* convert to YAML objects;
				   Since YAML maps do not preserve ordering, comparing YAML as raw text will always fail. Thus compare YAML semantically and not textually.
				*/
				if err := yaml.Unmarshal([]byte(tc.expectedStr), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				if err := yaml.Unmarshal(mockOsWriteFileContent, &observed); err != nil {
					t.Fatalf("failed to unmarshal observed: %v", err)
				}

				assert.Equal(t, expected, observed)
				assert.Equal(t, len(observed["runcmd"]), tc.nrExpectedItemsRC)
				assert.Equal(t, len(observed["write_files"]), tc.nrExpectedItemsWF)
			}
		})
	}

}
