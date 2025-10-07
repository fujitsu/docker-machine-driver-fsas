package cfgutils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/stretchr/testify/assert"
)

func TestIsInit_Fail(t *testing.T) {
	manager := &StandardCfgManager{}
	observed := manager.IsInit()
	assert.Equal(t, false, observed)
}

func TestIsInit_Success(t *testing.T) {
	manager := NewStandardCfgManager("[]")
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
			manager := NewStandardCfgManager("[]")
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

	manager := NewStandardCfgManager("[]")

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
	manager := NewStandardCfgManager("[]")
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
	manager := NewStandardCfgManager(devicesSpecJson)
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
				`kubelet-arg+: "provider-id=fsas://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"`)},
		{machineUUID: "1234",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas://1234"`)},
		{machineUUID: "",
			expected: fmt.Sprintf(rke2ConfigScriptContent, configName,
				`kubelet-arg+: "provider-id=fsas://"`)},
	}

	manager := NewStandardCfgManager("[]")

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
	manager := NewStandardCfgManager(devicesSpecJson)
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
	manager := NewStandardCfgManager(devicesSpecJson)
	labels := manager.prepareRke2ConfigNodeLabelsForGpu()
	t.Logf("Generated GPU label: %s", labels)
}

func TestPrepareRootPartitionResizeScript(t *testing.T) {
	manager := NewStandardCfgManager("[]")
	scriptContent := manager.PrepareRootPartitionResizeScript()
	assert.Equal(t, rootPartitionResizeScriptContent, scriptContent)
}
