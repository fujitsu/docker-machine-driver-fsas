package cfgutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"gopkg.in/yaml.v3"
)

const (
	LanportTypeUndetermined int = 0
	LanportTypeOnboard      int = 1
	LanportTypeComposable   int = 2
)

var (
	isInit = false
)

// CfgManager interface defines the methods for interacting with the Configuration Manager.
type CfgManager interface {
	IsInit() bool
	PrepareMetadata(instanceId, hostname string) string
	PrepareNetworkConfig(lanports []models.Lanport, subnets map[string]string) (string, error)
	ExtendUserdataRunCmd(commands []string) error
	ExtendUserdataWriteFiles(fileObjects []CloudConfigItem) error
	ImplantSSHKey(sshKeyPath, sshUser string) error
	ImplantRKE2Config(configName, machineUUID string) error
	InjectOSRegistration(regcode, email string) error
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
		slog.Warn("Failed to parse DevicesSpecJson, proceeding with empty resources", "err", err)
		resources = []models.Resource{}
	}

	isInit = true
	return &StandardCfgManager{resources: resources, userDataFile: userDataFile}
}

// IsInit Returns true if constructor succeed else false
func (sc *StandardCfgManager) IsInit() bool {
	return isInit
}

func (sc *StandardCfgManager) PrepareNetworkConfig(lanports []models.Lanport, subnets map[string]string) (string, error) {
	if len(lanports) == 0 {
		slog.Error("No lanports provided for network configuration")
		return "", fmt.Errorf("Bonding requested but no lanports available")
	}

	ethernets := make(map[string]models.Ethernet)
	bonds := make(map[string]models.Bond)
	bondInterfaces := []string{}

	for idx, lanport := range lanports {
		ifaceName := ""
		if lanport.SubnetUUID == subnets["provisioning"] {
			ifaceName = fmt.Sprintf("prov%d", idx)
		} else if lanport.SubnetUUID == subnets["baremetal"] {
			ifaceName = fmt.Sprintf("bare%d", idx)
		} else {
			ifaceName = fmt.Sprintf("custom%d", idx)
		}
		ethernets[ifaceName] = models.Ethernet{
			Match: models.Match{MACAddress: lanport.MACAddress},
			DHCP4: true,
		}
		// Only onboard interfaces with lanport indices 1 and 2 are to be bonded together
		if (lanport.LanportIdx == 1 || lanport.LanportIdx == 2) && lanport.LanportType == LanportTypeOnboard {
			bondInterfaces = append(bondInterfaces, ifaceName)
		}
	}
	// Create bond interface
	if len(bondInterfaces) > 1 {
		bonds["bond0"] = models.Bond{
			Interfaces: bondInterfaces,
			DHCP4:      true,
			Parameters: models.BondParameters{
				Mode:              models.BondModeActiveBackup,
				FailoverMacPolicy: models.FailoverMacPolicyActive,
			},
		}
		slog.Debug("Created bond interface")
	}

	networkConfig := models.NetworkConfig{
		Network: models.NetworkSpec{
			SchemaVersion: models.NetworkConfigVersion2,
			Renderer:      models.RendererNetworkManager,
			Ethernets:     ethernets,
			Bonds:         bonds,
		},
	}

	rawYaml, err := yaml.Marshal(networkConfig)
	if err != nil {
		slog.Error("Failed to marshal network config to YAML", "err", err)
		return "", err
	}

	return string(rawYaml), nil
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
			slog.Warn("Skipping labels because GPU model not allowed", "value", model)
			continue
		}

		if res.MinResourceCount > res.MaxResourceCount {
			slog.Warn("Invalid GPU config: MinResourceCount > MaxResourceCount", "model", fullModel, "min", res.MinResourceCount, "max", res.MaxResourceCount)
			continue
		}

		if res.MinResourceCount > 0 {
			labels = append(labels, fmt.Sprintf("cohdi.io/%s-size-min=%d", fullModel, res.MinResourceCount))
		} else {
			slog.Warn("MinResourceCount missing for GPU", "model", fullModel)
		}

		if res.MaxResourceCount > 0 {
			labels = append(labels, fmt.Sprintf("cohdi.io/%s-size-max=%d", fullModel, res.MaxResourceCount))
		} else {
			slog.Warn("MaxResourceCount missing for GPU", "model", fullModel)
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

func (sc *StandardCfgManager) ExtendUserdataSshAuthKeys(commands []string) error {
	cloudConfigItems := []CloudConfigItem{NewCloudConfigItemSshAuthKeys(commands)}
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
			slog.Error("User data file does not exist", "path", sc.userDataFile, "err", err)
		} else {
			slog.Error("User data cannot be read", "path", sc.userDataFile, "err", err)
		}
		return err
	}

	if len(cci) == 0 {
		slog.Warn("No items were passed for extending user data")
		return nil
	}

	cloudConfig := make(map[string]any)
	if err := yaml.Unmarshal(userdata, &cloudConfig); err != nil {
		slog.Error("Failed to parse user data as YAML", "path", sc.userDataFile, "err", err)
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
		slog.Error("Failed to write userdata file", "path", sc.userDataFile, "err", err)
		return err
	}
	return nil
}

func (sc *StandardCfgManager) ImplantSSHKey(sshKeyPath, sshUser string) error {

	sshPubKeyContent, err := getFileContent(fmt.Sprintf("%s.pub", sshKeyPath))
	if err != nil {
		return err
	}
	sshPubKeyContentTrimmed := strings.TrimSpace(string(sshPubKeyContent))

	slog.Debug("ssh public key", "keyName", fmt.Sprintf("%s.pub", filepath.Base(sshKeyPath)), "keyValue", sshPubKeyContentTrimmed)

	items := []CloudConfigItem{
		NewCloudConfigItemUsers(strings.TrimSpace(sshUser), []string{sshPubKeyContentTrimmed}),
	}
	if err := sc.extendUserdata(items); err != nil {

		return err
	}

	return nil
}

/*
	InjectOSRegistration registers SUSE products by extending cloud config with SUSEConnect registration commands.

It extends the section "runcmd" and "write_files" with commands for registration and commands for attaching products.
Simplified example:

runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh

write_files:
  - path: /tmp/register-suse-modules.sh
    permissions: "0755"
    content: |
    #!/bin/bash
    SUSEConnect -r 111-222-333 -e john@doe.com
    SUSEConnect -p sles/15.7/x86_64
    SUSEConnect -p sle-module-basesystem/15.7/x86_64
    SUSEConnect -p sle-module-public-cloud/15.7/x86_64
*/
func (sc *StandardCfgManager) InjectOSRegistration(regCode, email string) error {

	if regCode == "" {
		slog.Info("OS registration skipped: no registration code provided")
		return nil
	}

	scriptPath := "/tmp/register-suse-modules.sh"

	if err := sc.ExtendUserdataRunCmd([]string{
		fmt.Sprintf("sh %s", scriptPath),
		fmt.Sprintf("rm %s", scriptPath), // remove script because it contains sensitive data (registration code)
	}); err != nil {
		slog.Error("Failed to extend user data with OS registration commands (runcmd)", "err", err)
		return err
	}

	writeFilesRegisteringScriptContent, err := getWriteFilesContentForSuseRegistration(regCode, email)
	if err != nil {
		return err
	}

	if err := sc.ExtendUserdataWriteFiles([]CloudConfigItem{
		NewCloudConfigItemWriteFiles(scriptPath, writeFilesRegisteringScriptContent)}); err != nil {
		slog.Error("Failed to extend user data with SUSE registration script (write_files)", "err", err)
		return err
	}

	return nil
}

/*
getWriteFilesContentForSuseRegistration returns content for cloud config write_files section
for SUSE registration script based on provided registration code, email and list of SUSE products to register.
*/
func getWriteFilesContentForSuseRegistration(regCode, email string) (string, error) {
	templateForScript := `#!/bin/bash
set -e
timestamp=$(date +%Y-%m-%d__%H_%M_%S)
exec > "/tmp/register-suse-modules-${timestamp}.log" 2>&1

if ! command -v jq 2>&1 >/dev/null; then
  echo "System OS registration cannot proceed. 'jq' command-line JSON processor is required but not found on your system. 'jq' must be installed as part of the OS image preparation steps. Please refer to the user manual for detailed instructions on preparing your OS image before proceeding with registration. Aborting registration."
  exit 1
fi

echo "Starting SUSE Registration..."

cmd="SUSEConnect -r {{.RegCode}} -e {{.Email}}"
echo "$> SUSEConnect -r ***** -e {{.Email}}"
$cmd

sudo SUSEConnect --status | jq -r '.[] | "\(.identifier)\t\(.status)\t\(.arch)\t\(.version)"' | while IFS=$'\t' read -r id status arch ver; do
  echo "id=$id/$ver/$arch, status=$status"
  if [ "$status" == "Not Registered" ]; then
    echo "ACTION: Registering $id"
	for i in {1..4}; do
        echo "Attempt $i for registering $id"

        # Try to register the module
		cmd="SUSEConnect -p ${id}/${ver}/${arch}"
		echo "$> $cmd"
        if $cmd; then
            echo "Successfully activated $id"
            break
        else
            echo "Got Error for $id. Wait and retry"
            cmd="sleep 30"
			echo "$> $cmd"
			$cmd
        fi
    done

  else
    echo "Already registered: $id"
  fi
done`

	var data = struct {
		RegCode string
		Email   string
	}{
		RegCode: regCode,
		Email:   email,
	}

	t, err := template.New("script").Parse(templateForScript)
	if err != nil {
		return "", err
	}

	var buffer strings.Builder
	if err := t.Execute(&buffer, data); err != nil {
		slog.Error("Failed to execute template for SUSE registration script", "err", err)
		return "", err
	}

	return buffer.String(), nil
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

func getFileContent(pathToFile string) (fileContent []byte, err error) {

	fileContent, err = os.ReadFile(pathToFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Error("File does not exist", "path", pathToFile, "err", err)
		} else {
			slog.Error("File cannot be read", "path", pathToFile, "err", err)
		}
		return nil, err
	}

	return fileContent, err
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
