package cfgutils

import (
	"fmt"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
)

var (
	isInit = false
)

// CfgManager interface defines the methods for interacting with the Configuration Manager.
type CfgManager interface {
	IsInit() bool
	PrepareMetadata(instanceId, hostname string) string
	PrepareRke2ConfigScript(configName, machineUUID string) string
	PrepareRootPartitionResizeScript() string
}

// StandardCfgManager struct holds configuration for Configuration Manager interaction.
type StandardCfgManager struct{}

var _ CfgManager = (*StandardCfgManager)(nil)

// NewStandardCfgManager Returns new instance of Standard Configuration Manager
func NewStandardCfgManager() *StandardCfgManager {
	isInit = true
	return &StandardCfgManager{}
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
	configContent := sc.prepareRke2ConfigProviderId(machineUUID)
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

/*
WARNING: const below (#!/bin/sh ...) must be aligned to left because otherwise it does not work.
*/
const rootPartitionResizeScriptContent = `
#!/bin/sh
set -e
set -o pipefail
if [ "$(systemd-detect-virt)" != "none" ]; then
    exit 0 # Exit successfully as no action is needed
fi
ROOT_PARTITION=$(/usr/bin/findmnt / -o source -n)
DEVICES=$(/usr/bin/lsblk -no NAME,TYPE | /usr/bin/awk '$2=="disk"{print "/dev/"$1}')
DEVICE=""
PARTITION_NUMBER=""

for CURRENT_DEVICE in $DEVICES; do
if [[ "$ROOT_PARTITION" == $CURRENT_DEVICE* ]]; then
DEVICE=$CURRENT_DEVICE
PARTITION_NUMBER=${ROOT_PARTITION#$DEVICE}
PARTITION_NUMBER=${PARTITION_NUMBER#p}
break
fi
done

if [[ -z "$DEVICE" ]]; then
/usr/bin/echo "Error: No matching device found for root partition." >&2
exit 1
elif [[ -z "$PARTITION_NUMBER" ]]; then
/usr/bin/echo "Error: Partition number not found." >&2
exit 1
fi

/usr/bin/echo -e "Fix\n$PARTITION_NUMBER\n100%\nYes\n" | /usr/sbin/parted ---pretend-input-tty $DEVICE resizepart $PARTITION_NUMBER 100%
/usr/sbin/resize2fs $ROOT_PARTITION
`

// PrepareRootPartitionResizeScript prepares script content string to resize root partition from 10GB to max
func (sc *StandardCfgManager) PrepareRootPartitionResizeScript() string {
	slog.Debug("Prepare root partition resize script")
	return rootPartitionResizeScriptContent
}
