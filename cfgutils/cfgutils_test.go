package cfgutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInit_Fail(t *testing.T) {
	manager := &StandardCfgManager{}
	observed := manager.IsInit()
	assert.Equal(t, false, observed)
}

func TestIsInit_Success(t *testing.T) {
	manager := NewStandardCfgManager()
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
		{instanceId: "", hostname: "host1", expected: "dsmode: local\ninstance-id: \nhostname: host1",},
		{instanceId: "", hostname: "", expected: "dsmode: local\ninstance-id: \nhostname: ",},
	}

	for _, tc := range testCases {
		t.Run(tc.hostname, func(t *testing.T) {
			manager := NewStandardCfgManager()
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

	manager := NewStandardCfgManager()

	for _, tc := range testCases {
		t.Run(tc.machineUUID, func(t *testing.T) {
			observed := manager.prepareRke2ConfigProviderId(tc.machineUUID)
			assert.Equal(t, tc.expected, observed)
		})
	}
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

	manager := NewStandardCfgManager()

	for _, tc := range testCases {
		t.Run(tc.machineUUID, func(t *testing.T) {
			observed := manager.PrepareRke2ConfigScript(configName, tc.machineUUID)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func TestPrepareRootPartitionResizeScript(t *testing.T) {
	manager := NewStandardCfgManager()
	scriptContent := manager.PrepareRootPartitionResizeScript()
	assert.Equal(t, rootPartitionResizeScriptContent, scriptContent)
}
