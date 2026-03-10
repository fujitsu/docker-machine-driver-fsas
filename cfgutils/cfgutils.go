package cfgutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"gopkg.in/yaml.v3"
)

var (
	isInit = false
)

// CfgManager interface defines the methods for interacting with the Configuration Manager.
type CfgManager interface {
	IsInit() bool
	PrepareMetadata(instanceId, hostname string) string
	ExtendUserdataRunCmd(commands []string) error
	ExtendUserdataWriteFiles(fileObjects []CloudConfigItem) error
	ImplantRKE2Config(configName, machineUUID string) error
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
		slog.Warn("Failed to parse DevicesSpecJson, proceeding with empty resources:", "err", err)
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

// prepareRke2ConfigProviderId Returns string with provider ID containing machine UUID
func (sc *StandardCfgManager) prepareRke2ConfigProviderId(MachineUUID string) string {
	slog.Debug("Prepare RKE2 Config Provider ID")
	providerIdEntry := fmt.Sprintf(`kubelet-arg+: "provider-id=fsas-cdi://%s"`, MachineUUID)
	slog.Debug(providerIdEntry)
	return providerIdEntry
}

// prepareRke2ConfigNodeLabelsForGpu returns a string with node labels
func (sc *StandardCfgManager) prepareRke2ConfigNodeLabelsForGpu() string {
	slog.Debug("Prepare RKE2 Config Node Labels")

	// GPU map (short names to full names)
	allowedGPUs := map[string]string{
		"Gaudi3":  "intel-gaudi3",
		"H200NVL": "nvidia-h200nvl",
		"L40S":    "nvidia-l40s",
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

	userdata, err := os.ReadFile(sc.userDataFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Error("User data file does not exist:", "path", sc.userDataFile, "err", err)
		} else {
			slog.Error("User data cannot be read:", "path", sc.userDataFile, "err", err)
		}
		return err
	}

	if len(cci) == 0 {
		slog.Warn("No items were passed for extending user data")
		return nil
	}

	cloudConfig := make(map[string]any)
	if err := yaml.Unmarshal(userdata, &cloudConfig); err != nil {
		slog.Error("Failed to parse user data as YAML:", "path", sc.userDataFile, "err", err)
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

		slice, ok := existing.([]any)
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

	if err := os.WriteFile(sc.userDataFile, trimmed, os.FileMode(0644)); err != nil {
		slog.Error("Failed to write userdata file:", "path", sc.userDataFile, "err", err)
		return err
	}
	return nil
}

// ImplantRKE2Config extends userdata cloud-config file and prepare files that configure rke2.
func (sc *StandardCfgManager) ImplantRKE2Config(configName, machineUUID string) error {
	rke2ConfigFileContent := sc.getRke2ConfigFileContent(machineUUID)
	rke2ConfigScriptWriteFilesItems := []CloudConfigItem{
		NewCloudConfigItemWriteFiles(fmt.Sprintf("/etc/rancher/k3s/config.yaml.d/%s", configName), rke2ConfigFileContent),
		NewCloudConfigItemWriteFiles(fmt.Sprintf("/etc/rancher/rke2/config.yaml.d/%s", configName), rke2ConfigFileContent),
	}

	if err := sc.ExtendUserdataWriteFiles(rke2ConfigScriptWriteFilesItems); err != nil {
		return err
	}

	return nil
}

// getRke2ConfigFileContent prepares content of file with rke2 configuration that will be added to cloud config userdata file
func (sc *StandardCfgManager) getRke2ConfigFileContent(machineUUID string) string {
	providerIdEntry := sc.prepareRke2ConfigProviderId(machineUUID)
	nodeLabelEntry := sc.prepareRke2ConfigNodeLabelsForGpu()

	var configContent string
	if nodeLabelEntry != "" {
		configContent = fmt.Sprintf("%s\n%s", providerIdEntry, nodeLabelEntry)
	} else {
		configContent = providerIdEntry
	}
	return configContent
}
