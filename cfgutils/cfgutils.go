package cfgutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"gopkg.in/yaml.v3"
)

var (
	isInit      = false
	osWriteFile = os.WriteFile
	osReadFile  = os.ReadFile
	osStat      = os.Stat
)

// CfgManager interface defines the methods for interacting with the Configuration Manager.
type CfgManager interface {
	IsInit() bool
	PrepareMetadata(instanceId, hostname string) string
	PrepareRke2ConfigScript(configName, machineUUID string) string
	ExtendUserdataRunCmd(commands []string) error
	ExtendUserdataWriteFiles(fileObjects []CloudConfigItem) error
}

// StandardCfgManager struct holds configuration for Configuration Manager interaction.
type StandardCfgManager struct {
	resources    []models.Resource
	userDataFile string
}

var _ CfgManager = (*StandardCfgManager)(nil)

// NewStandardCfgManager Returns new instance of Standard Configuration Manager
func NewStandardCfgManager(devicesSpecJson, userDataFile string) *StandardCfgManager {
	var resources []models.Resource
	if err := json.Unmarshal([]byte(devicesSpecJson), &resources); err != nil {
		slog.Warn("Failed to parse DevicesSpecJson, proceeding with empty resources: ", "err", err)
		resources = []models.Resource{}
	}

	isInit = true
	return &StandardCfgManager{resources: resources, userDataFile: userDataFile}
}

// IsInit Returns true if constructor succeed else false
func (sc *StandardCfgManager) IsInit() bool {
	return isInit
}

const metadataContent = `dsmode: local
instance-id: %s
hostname: %s`

// PrepareMetadata Returns multiline string with metadata containing instanceId and hostname
func (sc *StandardCfgManager) PrepareMetadata(instanceId, hostname string) string {
	content := fmt.Sprintf(metadataContent, instanceId, hostname)
	return content
}

// prepareRke2ConfigScript Prepares script for RKE2
func (sc *StandardCfgManager) PrepareRke2ConfigScript(configName, machineUUID string) string {
	slog.Debug(fmt.Sprintf("Prepare RKE2 Config Script: %s", configName))
	providerIdEntry := sc.prepareRke2ConfigProviderId(machineUUID)
	nodeLabelEntry := sc.prepareRke2ConfigNodeLabelsForGpu()

	var configContent string
	if nodeLabelEntry != "" {
		configContent = fmt.Sprintf("%s\n%s", providerIdEntry, nodeLabelEntry)
	} else {
		configContent = providerIdEntry
	}
	return fmt.Sprintf(rke2ConfigScriptContent, configName, configContent)
}

/*
WARNING: const below (#!/bin/sh ...) must be aligned to left because otherwise it does not work.
*/
const rke2ConfigScriptContent = `
#!/bin/sh
for d in k3s rke2; do
mkdir -p /etc/rancher/${d}/config.yaml.d
cat << EOF > /etc/rancher/${d}/config.yaml.d/%s.yaml
%s
EOF
done
`

// prepareRke2ConfigProviderId Returns string with provider ID containing machine UUID
func (sc *StandardCfgManager) prepareRke2ConfigProviderId(MachineUUID string) string {
	slog.Debug("Prepare RKE2 Config Provider ID")
	providerIdEntry := fmt.Sprintf(`kubelet-arg+: "provider-id=fsas://%s"`, MachineUUID)
	slog.Debug(providerIdEntry)
	return providerIdEntry
}

// prepareRke2ConfigNodeLabelsForGpu returns a string with node labels
func (sc *StandardCfgManager) prepareRke2ConfigNodeLabelsForGpu() string {
	slog.Debug("Prepare RKE2 Config Node Labels")

	// GPU map (short names to full names)
	allowedGPUs := map[string]string{
		"nvidia-a100-40g": "nvidia-a100-40g",
		"nvidia-a100-80g": "nvidia-a100-80g",
		"nvidia-h100":     "nvidia-h100",
		"a100-40g":        "nvidia-a100-40g",
		"a100-80g":        "nvidia-a100-80g",
		"h100":            "nvidia-h100",
	}

	labels := []string{}

	for _, res := range sc.resources {
		if res.ResourceType != "gpu" || res.ResourceSpec == nil {
			continue
		}

		model := ""
		for _, cond := range res.ResourceSpec.Condition {
			if cond.Column == "model" && cond.Operator == "eq" {
				model = cond.Value
				break
			}
		}

		fullModel, ok := allowedGPUs[model]
		if !ok {
			slog.Warn("Skipping labels because GPU model not allowed: ", "value", model)
			continue
		}

		if res.MinResourceCount > res.MaxResourceCount {
			slog.Warn("Invalid GPU config: MinResourceCount > MaxResourceCount ", "model", fullModel, "min", res.MinResourceCount, "max", res.MaxResourceCount)
			continue
		}

		if res.MinResourceCount > 0 {
			labels = append(labels, fmt.Sprintf("cohdi.io/%s-size-min=%d", fullModel, res.MinResourceCount))
		} else {
			slog.Warn("MinResourceCount missing for GPU: ", "model", fullModel)
		}

		if res.MaxResourceCount > 0 {
			labels = append(labels, fmt.Sprintf("cohdi.io/%s-size-max=%d", fullModel, res.MaxResourceCount))
		} else {
			slog.Warn("MaxResourceCount missing for GPU: ", "model", fullModel)
		}
	}

	if len(labels) == 0 {
		slog.Debug("No GPU labels generated because of empty GPU resources")
		return ""
	}

	return fmt.Sprintf(`kubelet-arg+: "node-labels=%s"`, strings.Join(labels, ","))
}

func (sc *StandardCfgManager) ExtendUserdataRunCmd(commands []string) error {
	cloudConfigItems := []CloudConfigItem{NewCloudConfigItemRunCmd(commands)}
	return sc.extendUserdata(cloudConfigItems)
}

func (sc *StandardCfgManager) ExtendUserdataWriteFiles(fileObjects []CloudConfigItem) error {
	return sc.extendUserdata(fileObjects)
}

// extendUserdata Extends cloud config userdata file
func (sc *StandardCfgManager) extendUserdata(cci []CloudConfigItem) error {
	if sc.userDataFile == "" {
		return nil
	}

	if _, err := osStat(sc.userDataFile); os.IsNotExist(err) {
		slog.Error("User data file does not exist:", "path", sc.userDataFile, "err", err)
		return err
	}

	userdata, err := osReadFile(sc.userDataFile)
	if err != nil {
		return err
	}

	cloudConfig := make(map[string]interface{})
	if err := yaml.Unmarshal(userdata, &cloudConfig); err != nil {
		return err
	}

	for _, ccItem := range cci {
		moduleName := ccItem.getModuleName()

		newContent, err := ccItem.getNewCloudConfigContent()
		if err != nil {
			return fmt.Errorf("error while appending userdata file; module= %s: %w", moduleName, err)
		}

		existing, ok := cloudConfig[moduleName]
		if !ok {
			cloudConfig[moduleName] = newContent
			continue
		}

		slice, ok := existing.([]interface{})
		if !ok {
			return fmt.Errorf("module %s exists but is not a list", moduleName)
		}

		cloudConfig[moduleName] = append(slice, newContent...)
	}

	yamlBytes, err := yaml.Marshal(cloudConfig)
	if err != nil {
		return err
	}

	trimmed := bytes.TrimSpace(yamlBytes)

	if !bytes.HasPrefix(trimmed, []byte("#cloud-config")) {
		trimmed = append([]byte("#cloud-config\n"), trimmed...)
	}

	return osWriteFile(sc.userDataFile, trimmed, writeFilePermissions)
}
