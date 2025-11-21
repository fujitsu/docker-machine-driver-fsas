package fsas

import (
	"encoding/json"
	"fmt"
	"net/mail"

	"os"
	"path/filepath"
	"time"

	"github.com/fujitsu/docker-machine-driver-fsas/cfgutils"
	"github.com/fujitsu/docker-machine-driver-fsas/fm"
	"github.com/fujitsu/docker-machine-driver-fsas/keycloak"
	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/fujitsu/docker-machine-driver-fsas/sshutils"
	"github.com/fujitsu/docker-machine-driver-fsas/timeutils"
	"github.com/rancher/machine/libmachine/drivers"

	rpcdriver "github.com/rancher/machine/libmachine/drivers/rpc"
	"github.com/rancher/machine/libmachine/mcnflag"
	"github.com/rancher/machine/libmachine/state"
)

var statusClock timeutils.Clock = timeutils.NewRealClock()

const (
	WAIT_FOR_STATUS_TIMEOUT               time.Duration = 30 * time.Minute
	WAIT_FOR_STATUS_INSTALL_TIMEOUT       time.Duration = 30 * time.Minute
	WAIT_FOR_STATUS_STEP                  time.Duration = 5 * time.Second
	WAIT_FOR_STATUS_STEP_FOR_INSTALLATION time.Duration = 5 * time.Second
	WAIT_FOR_STATUS_INSTALL_STEP          time.Duration = 5 * time.Second
	WAIT_FOR_STATUS_STOPPED_TIMEOUT       time.Duration = 15 * time.Second
	WAIT_FOR_STATUS_NOT_FOUND_TIMEOUT     time.Duration = 15 * time.Second
	WAIT_FOR_START_AFTER_REBOOT           time.Duration = 60 * time.Second
)

// Driver is the implementation of BaseDriver interface
type Driver struct {
	*drivers.BaseDriver
	SSHPassword               string
	TenantUuid                string
	Username                  string
	Password                  string
	ApiUrl                    string
	NtpUrl                    string
	DnsIp                     string
	ComputeConditionsJson     string
	DevicesSpecJson           string
	NetworkBaremetalPort      int
	NetworkBaremetalUUID      string
	NetworkBaremetalDefaultGW string
	NetworkProvisionPort      int
	NetworkProvisionUUID      string
	NetworkProvisionDefaultGW string
	PrivateIPAddress          string
	OsImageName               string
	OsImageSshHostPubKey      string
	MachineUUID               string
	UserDataFile              string
	SlesRegistrationCode      string
	SlesRegistrationEmail     string
	FabricManager             fm.FabricManager    `json:"-"`
	Keycloak                  keycloak.Keycloak   `json:"-"`
	SshManager                sshutils.SshManager `json:"-"`
	CfgManager                cfgutils.CfgManager `json:"-"`
}

// NewDriver creates and returns a new instance of the FSAS CDI driver
func NewDriver() *Driver {
	return &Driver{
		BaseDriver:                &drivers.BaseDriver{},
		SSHPassword:               "",
		TenantUuid:                "",
		Username:                  "",
		Password:                  "",
		ApiUrl:                    "",
		NtpUrl:                    "",
		DnsIp:                     "",
		ComputeConditionsJson:     "",
		DevicesSpecJson:           "",
		NetworkBaremetalPort:      -1,
		NetworkBaremetalUUID:      "",
		NetworkBaremetalDefaultGW: "",
		NetworkProvisionPort:      -1,
		NetworkProvisionUUID:      "",
		NetworkProvisionDefaultGW: "",
		PrivateIPAddress:          "",
		OsImageName:               "",
		MachineUUID:               "",
		UserDataFile:              "",
		SlesRegistrationCode:      "",
		SlesRegistrationEmail:     "",
		FabricManager:             &fm.FabricManagerClient{},
		Keycloak:                  &keycloak.KeycloakClient{},
		SshManager:                &sshutils.StandardSshManager{},
		CfgManager:                &cfgutils.StandardCfgManager{},
	}
}

const (
	defaultSSHUser               = "rancher"
	defaultSSHPassword           = "rancher"
	defaultMachineType           = "ALL"
	defaultFabricManagerEndpoint = "/fabric_manager/api/v1"
	defaultKeycloakEndpoint      = "/id_manager"
	errorMandatoryOption         = "%s must be specified using the CLI option %s"
	cloudInitDirPath             = "/etc/cdi/cloud-init-discovery/"
)

func (d *Driver) String() string {
	return "{" +
		fmt.Sprintf("Tenant: %s, ", d.TenantUuid) +
		fmt.Sprintf("Username: %s, ", d.Username) +
		fmt.Sprintf("ApiUrl: %s, ", d.ApiUrl) +
		fmt.Sprintf("NtpUrl: %s, ", d.NtpUrl) +
		fmt.Sprintf("DnsIp: %s, ", d.DnsIp) +
		fmt.Sprintf("ComputeConditionsJson: %s, ", d.ComputeConditionsJson) +
		fmt.Sprintf("DevicesSpecJson: %s, ", d.DevicesSpecJson) +
		fmt.Sprintf("NetworkBaremetalPort: %d, ", d.NetworkBaremetalPort) +
		fmt.Sprintf("NetworkBaremetalUUID: %s, ", d.NetworkBaremetalUUID) +
		fmt.Sprintf("NetworkBaremetalDefaultGW: %s, ", d.NetworkBaremetalDefaultGW) +
		fmt.Sprintf("NetworkProvisionPort: %d, ", d.NetworkProvisionPort) +
		fmt.Sprintf("NetworkProvisionUUID: %s, ", d.NetworkProvisionUUID) +
		fmt.Sprintf("NetworkProvisionDefaultGW: %s, ", d.NetworkProvisionDefaultGW) +
		fmt.Sprintf("PrivateIPAddress: %s, ", d.PrivateIPAddress) +
		fmt.Sprintf("OsImageName: %s, ", d.OsImageName) +
		fmt.Sprintf("OsImageSshHostPubKey: %s, ", d.OsImageSshHostPubKey) +
		fmt.Sprintf("MachineUUID: %s, ", d.MachineUUID) +
		fmt.Sprintf("UserDataFile: %s", d.UserDataFile) +
		fmt.Sprintf("SlesRegistrationEmail: %s", d.SlesRegistrationEmail) +
		"}"
}

// GetCreateFlags returns the mcnflag.Flag slice representing the flags
// that can be set, their descriptions and defaults.
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	slog.Debug("Try to get create flags")
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:   "fsas-ssh-user",
			Usage:  "OS image SSH username",
			Value:  defaultSSHUser,
			EnvVar: "FSAS_SSH_USER",
		},
		mcnflag.StringFlag{
			Name:   "fsas-ssh-password",
			Usage:  "OS image SSH password",
			Value:  defaultSSHPassword,
			EnvVar: "FSAS_SSH_PASSWORD",
		},
		mcnflag.StringFlag{
			Name:   "fsas-tenant-uuid",
			Usage:  "FSAS CDI Tenant UUID",
			EnvVar: "FSAS_TENANT_UUID",
		},
		mcnflag.StringFlag{
			Name:   "fsas-credentials-username",
			Usage:  "FSAS CDI Credentials Username (Keycloak user)",
			EnvVar: "FSAS_CREDENTIALS_USERNAME",
		},
		mcnflag.StringFlag{
			Name:   "fsas-credentials-password",
			Usage:  "FSAS CDI Credentials Password (Keycloak password)",
			EnvVar: "FSAS_CREDENTIALS_PASSWORD",
		},
		mcnflag.StringFlag{
			Name:   "fsas-api-url",
			Usage:  "FSAS CDI API URL (API redirector URL, e.g. 'http://192.168.122.1')",
			EnvVar: "FSAS_API_URL",
		},
		mcnflag.StringFlag{
			Name:   "fsas-ntp-url",
			Usage:  "FSAS CDI NTP Server URL (URL address of NTP server e.g. '192.168.122.1')",
			EnvVar: "FSAS_NTP_URL",
		},
		mcnflag.StringFlag{
			Name:   "fsas-dns-ip",
			Usage:  "FSAS CDI DNS Server IP (IP address of DNS server e.g. '192.168.122.1')",
			EnvVar: "FSAS_DNS_IP",
		},
		mcnflag.StringFlag{
			Name:   "fsas-compute-conditions-json",
			Usage:  `FSAS CDI compute conditions JSON (string with CPU spec, e.g. "[{"column":"model","operator":"eq","value":"PRIMERGYRX2540M6"}]")`,
			EnvVar: "FSAS_COMPUTE_CONDITIONS_JSON",
		},
		mcnflag.IntFlag{
			Name:   "fsas-network-baremetal-port",
			Usage:  "Node LAN port index for baremetal subnet communication, e.g. 1",
			EnvVar: "FSAS_NETWORK_BAREMETAL_PORT",
		},
		mcnflag.StringFlag{
			Name:   "fsas-network-baremetal-uuid",
			Usage:  `Node subnet UUID for baremetal-baremetal communication`,
			EnvVar: "FSAS_NETWORK_BAREMETAL_UUID",
		},
		mcnflag.StringFlag{
			Name:   "fsas-network-baremetal-default-gw",
			Usage:  `Node subnet default gateway for baremetal-baremetal communication`,
			EnvVar: "FSAS_NETWORK_BAREMETAL_DEFAULT_GW",
		},
		mcnflag.IntFlag{
			Name:   "fsas-network-provision-port",
			Usage:  "Node LAN port index for provisioning subnet communication, e.g. 1",
			EnvVar: "FSAS_NETWORK_PROVISION_PORT",
		},
		mcnflag.StringFlag{
			Name:   "fsas-network-provision-uuid",
			Usage:  `Node subnet UUID for Rancher-baremetal communication`,
			EnvVar: "FSAS_NETWORK_PROVISION_UUID",
		},
		mcnflag.StringFlag{
			Name:   "fsas-network-provision-default-gw",
			Usage:  `Node subnet default gateway for Rancher-baremetal communication`,
			EnvVar: "FSAS_NETWORK_PROVISION_DEFAULT_GW",
		},
		mcnflag.StringFlag{
			Name:   "fsas-devices-spec-json",
			Usage:  `FSAS CDI devices specifications JSON (string with devices spec, e.g. "[{"res_type":"storage","res_num":1,"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]},"tags":{"is_bootstorage":true}}]")`,
			EnvVar: "FSAS_DEVICES_SPEC_JSON",
		},
		mcnflag.StringFlag{
			Name:   "fsas-os-image-name",
			Usage:  `OS image name used for system installation, e.g. sles3.img`,
			EnvVar: "FSAS_OS_IMAGE_NAME",
		},
		mcnflag.StringFlag{
			Name:   "fsas-image-os-ssh-host-pub-key",
			Usage:  `OS SSH host public key e.g. ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbml...AeLqg=`,
			EnvVar: "FSAS_IMAGE_OS_SSH_HOST_PUB_KEY",
		},
		mcnflag.StringFlag{
			Name:   "fsas-sles-registration-code",
			Usage:  "SLES registration code",
			EnvVar: "FSAS_SLES_REGISTRATION_CODE",
		},
		mcnflag.StringFlag{
			Name:   "fsas-sles-registration-email",
			Usage:  "SLES registration email",
			EnvVar: "FSAS_SLES_REGISTRATION_EMAIL",
		},
		mcnflag.StringFlag{
			Name:   "fsas-userdata",
			Usage:  "Warning: this field should remain empty as custom userdata are not supported!",
			EnvVar: "FSAS_USERDATA",
		},
	}
}

// UnmarshalJSON loads driver config from JSON. This function is used by the RPCServerDriver that wraps
// all drivers as a means of populating an already-initialized driver with new configuration.
// See `RPCServerDriver.SetConfigRaw`.
func (d *Driver) UnmarshalJSON(data []byte) error {
	slog.Debug("unmarshalling raw data")
	// Unmarshal driver config into an aliased type to prevent infinite recursion on UnmarshalJSON.
	type targetDriver Driver

	// Copy data from `d` to `target` before unmarshalling. This will ensure that already-initialized values
	// from `d` that are left untouched during unmarshal (like functions) are preserved.
	target := targetDriver(*d)

	if err := json.Unmarshal(data, &target); err != nil {
		return fmt.Errorf("error unmarshalling driver config from JSON: %w", err)
	}

	// Copy unmarshalled data back to `d`.
	*d = Driver(target)

	// Make sure to reload values that are subject to change from envvars and os.Args.
	driverOpts := rpcdriver.GetDriverOpts(d.GetCreateFlags(), os.Args)

	if _, ok := driverOpts.Values["fsas-tenant-uuid"]; ok {
		d.TenantUuid = driverOpts.String("fsas-tenant-uuid")
	}

	if _, ok := driverOpts.Values["fsas-credentials-username"]; ok {
		d.Username = driverOpts.String("fsas-credentials-username")
	}

	if _, ok := driverOpts.Values["fsas-credentials-password"]; ok {
		d.Password = driverOpts.String("fsas-credentials-password")
	}

	if _, ok := driverOpts.Values["fsas-api-url"]; ok {
		d.ApiUrl = driverOpts.String("fsas-api-url")
	}

	if _, ok := driverOpts.Values["fsas-ntp-url"]; ok {
		d.NtpUrl = driverOpts.String("fsas-ntp-url")
	}

	if _, ok := driverOpts.Values["fsas-dns-ip"]; ok {
		d.DnsIp = driverOpts.String("fsas-dns-ip")
	}

	if _, ok := driverOpts.Values["fsas-sles-registration-code"]; ok {
		d.SlesRegistrationCode = driverOpts.String("fsas-sles-registration-code")
	}
	if _, ok := driverOpts.Values["fsas-sles-registration-email"]; ok {
		d.SlesRegistrationEmail = driverOpts.String("fsas-sles-registration-email")
	}

	if _, ok := driverOpts.Values["fsas-userdata"]; ok {
		userDataFile := driverOpts.String("fsas-userdata")
		slog.Info("Logging content of cloud config file during UnmarshallJSON")
		logContentOfCloudConfigFile(userDataFile)
	}

	return nil
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	driverName := "fsas"
	slog.Debug("Driver ", "name", driverName)
	return driverName

}

// SetConfigFromFlags configures the driver with the object that was returned
// by RegisterCreateFlags
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	slog.Debug("Try to set config from flags")

	d.SSHUser = flags.String("fsas-ssh-user")
	slog.Debug("Driver ", "ssh-user", d.SSHUser)

	d.SSHPassword = flags.String("fsas-ssh-password")
	slog.Debug("Driver ", "ssh-password", "<hidden-for-security-reasons>")

	d.TenantUuid = flags.String("fsas-tenant-uuid")
	slog.Debug("Driver ", "Tenant Uuid", d.TenantUuid)

	d.Username = flags.String("fsas-credentials-username")
	slog.Debug("Driver ", "Credentials Username", d.Username)

	d.Password = flags.String("fsas-credentials-password")
	slog.Debug("Driver ", "Credentials Password", "<hidden-for-security-reasons>")

	d.ApiUrl = flags.String("fsas-api-url")
	slog.Debug("Driver ", "FSAS API url", d.ApiUrl)

	d.NtpUrl = flags.String("fsas-ntp-url")
	slog.Debug("Driver ", "FSAS NTP Server url", d.NtpUrl)

	d.DnsIp = flags.String("fsas-dns-ip")
	slog.Debug("Driver ", "FSAS DNS Server IP", d.DnsIp)

	d.ComputeConditionsJson = flags.String("fsas-compute-conditions-json")
	slog.Debug("Driver ", "FSAS compute conditions JSON", d.ComputeConditionsJson)

	d.NetworkBaremetalPort = flags.Int("fsas-network-baremetal-port")
	slog.Debug("Driver ", "FSAS baremetal subnet LAN port index", d.NetworkBaremetalPort)

	d.NetworkBaremetalUUID = flags.String("fsas-network-baremetal-uuid")
	slog.Debug("Driver ", "FSAS baremetal subnet UUID", d.NetworkBaremetalUUID)

	d.NetworkBaremetalDefaultGW = flags.String("fsas-network-baremetal-default-gw")
	slog.Debug("Driver ", "FSAS baremetal subnet Default GW", d.NetworkBaremetalDefaultGW)

	d.NetworkProvisionPort = flags.Int("fsas-network-provision-port")
	slog.Debug("Driver ", "FSAS provosioning subnet LAN port index", d.NetworkProvisionPort)

	d.NetworkProvisionUUID = flags.String("fsas-network-provision-uuid")
	slog.Debug("Driver ", "FSAS provisioning subnet UUID", d.NetworkProvisionUUID)

	d.NetworkProvisionDefaultGW = flags.String("fsas-network-provision-default-gw")
	slog.Debug("Driver ", "FSAS provisioning subnet Default GW", d.NetworkBaremetalDefaultGW)

	d.DevicesSpecJson = flags.String("fsas-devices-spec-json")
	slog.Debug("Driver ", "FSAS devices specification JSON", d.DevicesSpecJson)

	d.OsImageName = flags.String("fsas-os-image-name")
	slog.Debug("Driver ", "FSAS OS image name", d.OsImageName)

	d.UserDataFile = flags.String("fsas-userdata")
	slog.Debug("Driver ", "FSAS user data file", d.UserDataFile)

	if err := d.initClients(); err != nil {
		slog.Error("Error while initializing Keycloak and Fabric Manager clients", "err", err)
		return err
	}

	d.OsImageSshHostPubKey = flags.String("fsas-image-os-ssh-host-pub-key")
	slog.Debug("Driver ", "FSAS OS image ssh host public key ", d.OsImageSshHostPubKey)

	d.SlesRegistrationCode = flags.String("fsas-sles-registration-code")
	slog.Debug("Driver ", "FSAS SLES registration code", "<hidden-for-security-reasons>")

	d.SlesRegistrationEmail = flags.String("fsas-sles-registration-email")
	slog.Debug("Driver ", "FSAS SLES registration email", d.SlesRegistrationEmail)

	return d.checkConfig()
}

// initClients Initialize clients: keycloak and Fabric Manager
func (d *Driver) initClients() error {
	slog.Debug("Init Keycloak and Fabric Manager clients")
	if err := d.initKeycloak(); err != nil {
		slog.Error("Error while initializing Keycloak client", "err", err)
		return err
	}
	if err := d.initFabricManager(); err != nil {
		slog.Error("Error while initializing Fabric Manager client", "err", err)
		return err
	}

	return nil
}

// initKeycloak Initialize keycloak client.
// Initialization procedure consists of two steps:
// 1) connect and authenticate to keycloak service and get tokens (access and refresh)
// 2) authorization - verify if logged user is allowed to create cluster, if not return error
func (d *Driver) initKeycloak() error {
	if !d.Keycloak.IsInit() {
		slog.Warn("keycloak is NOT initialized then start init procedure")
		keycloak, err := keycloak.NewKeycloak(d.TenantUuid, d.Username, d.Password, d.ApiUrl, defaultKeycloakEndpoint)
		if err != nil {
			return err
		}
		d.Keycloak = keycloak
		if err := d.Keycloak.InitConnection(); err != nil {
			return err
		}
		if err := d.Keycloak.UserIsAllowedToCreateCluster(); err != nil {
			return err
		}
	}

	return nil
}

// initFabricManager Initialize Fabric Manager client
func (d *Driver) initFabricManager() error {
	if !d.FabricManager.IsInit() {
		slog.Warn("Fabric Manager is NOT initialized then start init procedure")
		fmc, err := fm.NewFabricManagerClient(d.ApiUrl, defaultFabricManagerEndpoint, d.DevicesSpecJson)
		if err != nil {
			slog.Error("Could not create Fabric Manager client because of an error: ", "err", err)
			return err
		}
		d.FabricManager = fmc
	}

	return nil
}

// checkConfig Verify if mandatory flags are set
func (d *Driver) checkConfig() error {
	slog.Debug("check config from mandatory flags")

	if d.SSHUser == "" {
		return fmt.Errorf(errorMandatoryOption, "SSH user", "--fsas-ssh-user")
	}
	if d.SSHPassword == "" {
		return fmt.Errorf(errorMandatoryOption, "SSH password", "--fsas-ssh-password")
	}
	if d.ComputeConditionsJson == "" {
		return fmt.Errorf(errorMandatoryOption, "Compute conditions (JSON)", "--fsas-compute-conditions-json")
	}
	if d.NetworkProvisionPort == -1 {
		return fmt.Errorf(errorMandatoryOption, "Provisioning subnet LAN port", "--fsas-network-provision-port")
	}
	if d.NetworkProvisionUUID == "" {
		return fmt.Errorf(errorMandatoryOption, "Provisioning subnet UUID", "--fsas-network-provision-uuid")
	}
	if d.NetworkProvisionDefaultGW == "" {
		return fmt.Errorf(errorMandatoryOption, "Provisioning subnet Default GW", "fsas-network-provision-default-gw")
	}
	if d.DevicesSpecJson == "" {
		return fmt.Errorf(errorMandatoryOption, "Devices specification (JSON)", "--fsas-devices-spec-json")
	}
	if err := fm.CheckDeviceSpecJson(d.DevicesSpecJson); err != nil {
		return err
	}
	if d.OsImageName == "" {
		return fmt.Errorf(errorMandatoryOption, "OS image name", "--fsas-os-image-name")
	}

	if err := d.FabricManager.ValidateTenant(d.TenantUuid, d.Keycloak.GetToken()); err != nil {
		slog.Error("tenant_uuid validation unsuccessful: ", "err", err)
		return err
	}
	slog.Debug("Driver ", "tenant_uuid validation successful", d.TenantUuid)

	if d.OsImageSshHostPubKey == "" {
		return fmt.Errorf(errorMandatoryOption, "OS image ssh host public key", "--fsas-image-os-ssh-host-pub-key")
	}

	if d.SlesRegistrationCode != "" {
		if d.SlesRegistrationEmail == "" {
			return fmt.Errorf("when SLES registration code is not empty then SLES registration email must also not be empty. Fill in param %s", "--fsas-sles-registration-email")
		} else {

			if _, err := mail.ParseAddress(d.SlesRegistrationEmail); err != nil {
				return fmt.Errorf("Email address is not valid: %s", d.SlesRegistrationEmail)
			}
		}
	}
	return nil
}

// Create a host using the driver's config
func (d *Driver) Create() error {
	if err := d.innerCreate(); err != nil {
		slog.Error("Error encountered during instance creation: ", "err", err)
		slog.Info("Attempting to remove partially created machine: ", "machineUUID", d.MachineUUID)
		if removalErr := d.Remove(); removalErr != nil {
			slog.Error("The attempt to remove partially provisioned machine failed: ", "err", removalErr)
			return fmt.Errorf("error during Create: '%s'; followed by error during Remove: '%s'", err.Error(), removalErr.Error())
		}
		return err
	}

	return nil
}

func (d *Driver) innerCreate() error {
	slog.Debug("Attempting to create FSAS CDI machine instance.")
	slog.Debug(fmt.Sprintf("BaseDriver struct: %+v", d.BaseDriver))
	slog.Debug(fmt.Sprintf("Driver struct: %+v", d))

	slog.Info("Logging content of cloud config file during Create")
	logContentOfCloudConfigFile(d.UserDataFile)

	if err := d.initClients(); err != nil {
		return err
	}

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson:     d.ComputeConditionsJson,
		DevicesSpecJson:           d.DevicesSpecJson,
		NetworkBaremetalPort:      d.NetworkBaremetalPort,
		NetworkBaremetalUUID:      d.NetworkBaremetalUUID,
		NetworkBaremetalDefaultGW: d.NetworkBaremetalDefaultGW,
		NetworkProvisionPort:      d.NetworkProvisionPort,
		NetworkProvisionUUID:      d.NetworkProvisionUUID,
		NetworkProvisionDefaultGW: d.NetworkProvisionDefaultGW,
		NtpServer:                 d.NtpUrl,
		DnsServer:                 d.DnsIp,
	}

	machineUUID, err := d.FabricManager.CreateMachine(d.MachineName, d.TenantUuid, machineSpecArgs, d.Keycloak.GetToken())
	if err != nil {
		return err
	}

	d.MachineUUID = machineUUID
	slog.Info("Successfully filled MachineUUID: ", "MachineUUID", d.MachineUUID)

	slog.Info("Waiting for status: ", "status", ACTIVE_POFF)
	if err := d.waitForStatus(ACTIVE_POFF, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_TIMEOUT); err != nil {
		return err
	}

	_, bootSsdId, _, err := d.FabricManager.GetMachineDetails(d.TenantUuid, d.MachineUUID, d.Keycloak.GetToken())
	if err != nil {
		return err
	}

	if err := d.FabricManager.ImageInstall(d.TenantUuid, bootSsdId, d.OsImageName, d.Keycloak.GetToken()); err != nil {
		return err
	}

	slog.Info("Waiting for the installation of the operating system: ", "status", OS_INSTALLING)
	if err := d.waitForStatus(OS_INSTALLING, WAIT_FOR_STATUS_STEP_FOR_INSTALLATION, WAIT_FOR_STATUS_TIMEOUT); err != nil {
		return err
	}

	slog.Info("Installing operating system: ", "OS", d.OsImageName)

	slog.Info("Waiting for operating system installation to complete: ", "status", ACTIVE_POFF)
	if err := d.waitForStatus(ACTIVE_POFF, WAIT_FOR_STATUS_INSTALL_STEP, WAIT_FOR_STATUS_INSTALL_TIMEOUT); err != nil {
		return err
	}

	if err := d.Start(); err != nil {
		return err
	}

	if err := d.assignIpAddresses(); err != nil {
		return err
	}

	hostName, err := d.GetSSHHostname()
	if err != nil {
		slog.Error("Could not acquire target SSH hostname because of an error: ", "err", err)
		return err
	}
	slog.Info("Acquired ssh hostname: ", "hostname", hostName)

	if !d.SshManager.IsInit() {
		sshManager, err := sshutils.NewStandardSshManager(hostName, d.GetSSHUsername(), d.SSHPassword, d.GetSSHKeyPath(), d.OsImageSshHostPubKey)
		if err != nil {
			slog.Error("error while initializing Standard SSH Manager: ", "err", err)
			return err
		}
		d.SshManager = sshManager
	}

	if err := d.SshManager.ExchangeKeys(); err != nil {
		return err
	}

	if err := d.SshManager.RegisterOS(d.SlesRegistrationCode, d.SlesRegistrationEmail); err != nil {
		slog.Error("Failed to register OS via SSH using SUSEConnect: ", "err", err, "email", d.SlesRegistrationEmail)
		return err
	}

	if !d.CfgManager.IsInit() {
		cfgManager := cfgutils.NewStandardCfgManager(d.DevicesSpecJson, d.UserDataFile)
		d.CfgManager = cfgManager
	}

	// write sample data using modified userdata file
	d.CfgManager.ExtendUserdataRunCmd([]string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
		`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`,
	})

	items := []cfgutils.CloudConfigItem{
		cfgutils.NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
		cfgutils.NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files-2.log", "Cloud config succeeded for write_files part 2"),
	}
	d.CfgManager.ExtendUserdataWriteFiles(items)

	slog.Info("Logging content of cloud config userdata file after extending it")
	logContentOfCloudConfigFile(d.UserDataFile)

	// Prepare scripts execution parameters
	scriptPath := "" // Random paths
	removeOnFinish := true
	runSudo := true

	// Generate script content for RKE2 setup
	overrideProviderIdScriptContent := d.CfgManager.PrepareRke2ConfigScript("100-fsas-providerid", d.MachineUUID)

	if err := d.SshManager.ExecuteScript(scriptPath, overrideProviderIdScriptContent, removeOnFinish, runSudo); err != nil {
		return err
	}

	if err := d.applyCloudInit(d.GetMachineName()); err != nil {
		return err
	}

	if err := d.SshManager.DisablePasswordSSHLogin(); err != nil {
		slog.Error("Failed to disable password login: ", "err", err)
		return err
	}

	return nil
}

var osReadFile = os.ReadFile

// applyCloudInit Save user-data and meta-data files on remote machine
func (d *Driver) applyCloudInit(sshHostName string) error {
	userdataPath := filepath.Join(cloudInitDirPath, "user-data")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")

	if d.UserDataFile != "" {
		userDataFileContent, err := osReadFile(d.UserDataFile)
		if err != nil {
			return err
		}

		if err := d.SshManager.WriteFileOnRemoteMachine(userdataPath, string(userDataFileContent), 0700); err != nil {
			return err
		}
	}
	metadataContent := d.CfgManager.PrepareMetadata(d.MachineUUID, sshHostName)

	if err := d.SshManager.WriteFileOnRemoteMachine(metadataPath, metadataContent, 0700); err != nil {
		return err
	}

	if err := d.SshManager.RebootCloudInit(); err != nil {
		slog.Error("Potential error while rebooting cloud init: ", "err", err)
		return err
	}

	// Wait for the machine to reach the Running state
	slog.Info("Waiting for the machine to reach the Running state")
	statusClock.Sleep(WAIT_FOR_START_AFTER_REBOOT)
	return nil
}

// GetSSHHostname returns hostname for use with ssh
func (d *Driver) GetSSHHostname() (string, error) {
	slog.Debug("SSH host name as Machine name: ", "name", d.MachineName)
	// Machine name sometimes is not written to the machine
	return d.GetIP()
}

// GetIP returns IP to use in communication
func (d *Driver) GetIP() (string, error) {
	slog.Info("IP ", "address", d.IPAddress)
	if d.IPAddress == "" {
		return "", fmt.Errorf("IPAddress is empty")
	}
	return d.IPAddress, nil
}

// GetState returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (state.State, error) {
	cdiState, err := d.getCdiState()
	if err != nil {
		return state.Error, err
	}
	// Map status code to state
	machineState := d.mapMachineStatusToState(cdiState)
	return machineState, nil
}

// getCdiState returns the state that the FSAS host is in (ACTIVE_PON, BOOTING, etc)
func (d *Driver) getCdiState() (CdiMachineState, error) {
	slog.Debug("Try to get state of the host")

	// error when MachineUUID is empty, return state.Error
	if d.MachineUUID == "" {
		slog.Error("Machine's UUID was unexpectedly empty: ", "machine_name", d.MachineName)
		return ERROR, fmt.Errorf("machine uuid is empty")
	}

	// init Fabric Manager and Keycloak
	if err := d.initClients(); err != nil {
		return ERROR, err
	}

	// Retrieve status code of Machine from Fabric Manager
	_, _, machineStatus, err :=
		d.FabricManager.GetMachineDetails(d.TenantUuid, d.MachineUUID, d.Keycloak.GetToken())

	if err != nil {
		slog.Error("Could not get Machine status: ", "err", err)
		return ERROR, err
	}

	return CdiMachineState(machineStatus), nil
}

// mapMachineStatusToState Converts FSAS host state into Rancher state
func (d *Driver) mapMachineStatusToState(cdiState CdiMachineState) state.State {
	slog.Debug("Map FM machineStatus code to State code")

	switch cdiState {
	case BUILDING, BUILDING_BEFORE_QUEUE, BOOTING:
		return state.Starting
	case ACTIVE_PON:
		return state.Running
	case POWERING_OFF:
		return state.Stopping
	case ACTIVE_POFF, UNBUILDED, UNBUILDING, UNBUILDING_WAIT:
		return state.Stopped
	case OS_INSTALLING:
		return state.Paused
	case ERROR:
		return state.Error
	default:
		slog.Warn("Unrecognized machine status: ", "machineStatus", cdiState)
		return state.None
	}
}

// GetURL returns a Docker compatible host URL for connecting to this host
// e.g. tcp://1.2.3.4:2376
func (d *Driver) GetURL() (string, error) {
	slog.Debug("Try to get url of the host")
	ip, err := d.GetIP()
	slog.Info("ip= " + ip)

	if err != nil {
		slog.Error("Error: ", "err", err)
		return "", err
	}

	return fmt.Sprintf("tcp://%s:%d", ip, 2376), nil
}

// waitForStatus Wait for host status
func (d *Driver) waitForStatus(expectedState CdiMachineState, step, timeout time.Duration) error {
	startTime := statusClock.Now()

	for {
		currentState, err := d.getCdiState()
		if err != nil {
			slog.Error("Error while checking state: ", "err", err)
			return fmt.Errorf("error getting state: %w", err)
		}

		if currentState == ERROR {
			slog.Error("Received ERROR state")
			return fmt.Errorf("received ERROR state error state: %d", ERROR)
		}

		if currentState == expectedState {
			slog.Debug("Successfully received the required status: ", "status", currentState)
			return nil
		}

		if statusClock.Since(startTime) >= timeout {
			slog.Error("Required status was not achieved within the specified time: ", "expected state", expectedState, "current state", currentState, "timeout", WAIT_FOR_STATUS_TIMEOUT)
			return fmt.Errorf("error: required status was not achieved within the specified time")
		}

		slog.Debug("Required status is not equal to received status, another attempt will occur: ", "expected state", expectedState, "current state", currentState)
		statusClock.Sleep(step)
	}
}

// Kill stops a host forcefully
func (d *Driver) Kill() error {
	slog.Debug("Try to kill host forcefully")

	// in case of not initialized Fabric Manager caused by e.g. method UnmarshalJSON verify init again
	// Fabric Manager needs also keycloak client then init both
	if err := d.initClients(); err != nil {
		return err
	}

	if d.MachineUUID == "" {
		slog.Error("Machine's UUID was unexpectedly empty: ", "machine_name", d.MachineName)
		return fmt.Errorf("machine uuid is empty")
	}

	if err := d.FabricManager.PowerOff(d.MachineUUID, d.TenantUuid,
		d.Keycloak.GetToken()); err != nil {
		slog.Error("Could not kill Machine: ", "machineUUID", d.MachineUUID, "err", err)
		return err
	}

	slog.Info("Waiting for status: ", "status", ACTIVE_POFF)
	if err := d.waitForStatus(ACTIVE_POFF, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_STOPPED_TIMEOUT); err != nil {
		slog.Error("Error while waiting for status: ", "status", ACTIVE_POFF, "err", err)
		return err
	}

	slog.Info("Successfully killed Machine: ", "machineUUID", d.MachineUUID)
	return nil
}

// Remove a host
func (d *Driver) Remove() error {
	slog.Debug("Attempting to remove host")
	slog.Debug(fmt.Sprintf("BaseDriver struct: %+v", d.BaseDriver))
	slog.Debug(fmt.Sprintf("Driver struct: %+v", d))

	if d.MachineUUID == "" {
		slog.Warn("Machine's UUID was unexpectedly empty: ", "machine_name", d.MachineName)
		/*
			The return value must be 'nil' instead of 'err' because when creating machine procedure fails and
			machine uuid is empty then Rancher tries to remove machine endlessly.
		*/
		return nil
	}

	// in case of not initialized Fabric Manager caused by e.g. method UnmarshalJSON verify init again
	// Fabric Manager needs also keycloak client then init both
	if err := d.initClients(); err != nil {
		return err
	}

	hostName, err := d.GetSSHHostname()
	if err != nil {
		// Similar as above - we must ignore hostname acquisition error to avoid perpetual loop
		slog.Warn("Could not acquire target SSH hostname because of an error: ", "err", err)
	} else {
		slog.Info("Acquired SSH hostname: ", "hostname", hostName)
		if !d.SshManager.IsInit() {
			sshManager, err := sshutils.NewStandardSshManager(hostName, d.GetSSHUsername(), d.SSHPassword, d.GetSSHKeyPath(), d.OsImageSshHostPubKey)
			if err != nil {
				slog.Error("error while initializing Standard SSH Manager: ", "err", err)
				return err
			}
			d.SshManager = sshManager
		}
		if err := d.SshManager.DeregisterOS(); err != nil {
			slog.Warn("Could not deregister SLES OS, manual action might be required: ", "err", err)
		}
	}

	if err := d.FabricManager.RemoveMachine(d.MachineUUID, d.TenantUuid, d.Keycloak.GetToken()); err != nil {
		slog.Error("Could not remove Machine: ", "machineUUID", d.MachineUUID, "err", err)
		return err
	}

	slog.Info("Waiting for status: ", "status", UNBUILDED)
	if err := d.waitForStatus(UNBUILDED, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_NOT_FOUND_TIMEOUT); err != nil {
		slog.Error("Error while waiting for status: ", "status", UNBUILDED, "err", err)
		return err
	}

	slog.Info("Successfully removed Machine: ", "machineUUID", d.MachineUUID)
	return nil
}

// Restart a host. This may just call Stop(); Start() if the provider does not
// have any special restart behaviour.
func (d *Driver) Restart() error {
	slog.Debug("Restarting host: ", "machineName", d.MachineName)

	slog.Debug("Attempting to Stop: ", "machineName", d.MachineName)
	if err := d.Stop(); err != nil {
		slog.Error("Issue during attempt to Stop the host: ", "err", err)
		return err
	}

	slog.Debug("Attempting to Start: ", "machineName", d.MachineName)
	if err := d.Start(); err != nil {
		slog.Error("Issue during attempt to Start the host: ", "err", err)
		return err
	}

	slog.Info("Successfully restarted host: ", "machineName", d.MachineName)
	return nil
}

// Start a host
func (d *Driver) Start() error {
	slog.Debug("Try to start the host")
	slog.Debug(fmt.Sprintf("BaseDriver struct: %+v", d.BaseDriver))
	slog.Debug(fmt.Sprintf("Driver struct: %+v", d))

	if err := d.initClients(); err != nil {
		return err
	}

	// error MachineUUID is empty
	if d.MachineUUID == "" {
		slog.Error("Machine's UUID was unexpectedly empty: ", "machine_name", d.MachineName)
		return fmt.Errorf("machine uuid is empty")
	}

	// Power on the machine
	if err := d.FabricManager.PowerOn(d.MachineUUID, d.TenantUuid, d.Keycloak.GetToken()); err != nil {
		slog.Error("Could not Power On the machine: ", "err", err)
		return err
	}

	// Wait for the machine to reach the Running state
	slog.Info("Waiting for status: ", "status", ACTIVE_PON)
	if err := d.waitForStatus(ACTIVE_PON, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_TIMEOUT); err != nil {
		slog.Error("Error occured during waitForStatus execution: ", "err", err)
		return err
	}

	slog.Info("Successfully started machine")
	return nil
}

// Stop a host gracefully
func (d *Driver) Stop() error {
	slog.Debug("Try to stop host gracefully")

	if !d.FabricManager.IsInit() {
		if err := d.initFabricManager(); err != nil {
			slog.Error("error while initializing Fabric Manager: ", "err", err)
			return err
		}
	}

	slog.Info("requesting graceful shutdown for machine: ", "machine_uuid", d.MachineUUID)
	if err := d.FabricManager.GracefulShutdown(d.MachineUUID, d.TenantUuid, d.Keycloak.GetToken()); err != nil {
		slog.Error("Graceful shutdown failed for: ", "machine_uuid", d.MachineUUID, "err", err)
		return err
	}

	slog.Info("Waiting for status: ", "status", ACTIVE_POFF)
	if err := d.waitForStatus(ACTIVE_POFF, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_STOPPED_TIMEOUT); err != nil {
		slog.Error("Error while waiting for status: ", "status", ACTIVE_POFF, "err", err)
		return err
	}

	slog.Info("Successfully stopped Machine: ", "machineUUID", d.MachineUUID)
	return nil

}

// PreCreateCheck allows for pre-create operations to make sure a driver is ready for creation
func (d *Driver) PreCreateCheck() error {
	slog.Debug("Checks before creating host")
	return nil
}

// GetSSHUsername Returns username for use with ssh
func (d *Driver) GetSSHUsername() string {
	slog.Info("SSH ", "username", d.SSHUser)
	return d.SSHUser
}

// setTokenToEmptySTring invalidates token.
// Token is definitely expired after one hour, and this method enables other ways of authentication.
func (d *Driver) setTokenToEmptySTring() {
	slog.Debug("Set token to empty string")

}

func (d *Driver) assignIpAddresses() error {
	slog.Debug("Trying to assign IP Address")
	lanports, _, _, err := d.FabricManager.GetMachineDetails(d.TenantUuid, d.MachineUUID, d.Keycloak.GetToken())
	if err != nil {
		return err
	}

	for idx, lanport := range lanports {
		slog.Debug(fmt.Sprintf("lanport[%d].SubnetUUID=%s", idx, lanport.SubnetUUID))
		if lanport.SubnetUUID == d.NetworkProvisionUUID && d.IPAddress == "" {
			d.IPAddress = lanport.IPAddress
			slog.Info("Successfully filled IP Address: ", "IP", d.IPAddress)
		}
		if lanport.SubnetUUID == d.NetworkBaremetalUUID && d.PrivateIPAddress == "" {
			d.PrivateIPAddress = lanport.IPAddress
			slog.Info("Successfully filled Private IP Address: ", "IP", d.PrivateIPAddress)
		}
	}

	// d.IPAddress is mandatory in the machine creation process
	if d.IPAddress == "" {
		return fmt.Errorf("IPAddress must not be empty")
	}

	return nil
}

func logContentOfCloudConfigFile(cloudConfigFilePath string) {
	if cloudConfigFilePath == "" {
		slog.Error("cloud config file not set (empty string)")
		return
	}
	slog.Info("user-data cloud config file: ", "ccf", cloudConfigFilePath)

	if _, err := os.Stat(cloudConfigFilePath); os.IsNotExist(err) {
		slog.Error("Provided cloud config file does not exist: ", "path", cloudConfigFilePath, "err", err)
		slog.Error("Current host: ", "hostname",
			func() string { h, _ := os.Hostname(); return h }())
		slog.Error("Current working dir: ", "cwd",
			func() string { c, _ := os.Getwd(); return c }())
		return
	}

	content, err := os.ReadFile(cloudConfigFilePath)
	if err != nil {
		slog.Error("Error while reading cloud config file: ", "path", cloudConfigFilePath, "err", err)
	}
	slog.Debug("Cloud config file content: ")
	slog.Debug(string(content))
}
