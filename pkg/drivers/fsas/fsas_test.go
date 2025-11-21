package fsas

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"

	"os"

	"testing"
	"time"

	"github.com/fujitsu/docker-machine-driver-fsas/cfgutils"
	cfgMock "github.com/fujitsu/docker-machine-driver-fsas/cfgutils/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/fm"
	fmmock "github.com/fujitsu/docker-machine-driver-fsas/fm/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/keycloak"
	keycloakMock "github.com/fujitsu/docker-machine-driver-fsas/keycloak/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	sshMock "github.com/fujitsu/docker-machine-driver-fsas/sshutils/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/timeutils"
	timeutilsmock "github.com/fujitsu/docker-machine-driver-fsas/timeutils/mock"

	"github.com/rancher/machine/libmachine/drivers"
	"github.com/rancher/machine/libmachine/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {

	/* setup code here */

	originalLogger := slog.Default()
	// Suppress slog output in test
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	exitCode := m.Run() // run tests

	/* tear-down code here */
	statusClock = timeutils.RealClock{}

	// Restore original logger
	slog.SetDefault(originalLogger)

	os.Exit(exitCode)
}

func TestDriverName(t *testing.T) {
	driver := NewDriver()

	observed := driver.DriverName()
	expected := "fsas"

	assert.Equal(t, expected, observed, "Incorrect driver name")
}

func TestGetSSHHostname(t *testing.T) {
	driver := NewDriver()
	ipAddress := "192.168.122.41"
	driver.IPAddress = ipAddress
	driver.MachineName = "sdo-test-new-mask-02-poolu-9vfml-dtsnf"

	observed, _ := driver.GetSSHHostname()
	expected := ipAddress

	assert.Equal(t, expected, observed, "Incorrect ssh hostname")
}

func TestGetIP(t *testing.T) {
	driver := NewDriver()
	ipAddress := "192.168.122.41"
	driver.IPAddress = ipAddress

	observed, _ := driver.GetIP()
	expected := ipAddress

	assert.Equal(t, expected, observed, "Incorrect IP address")
}

func TestGetIPIPAddressEmpty(t *testing.T) {
	driver := NewDriver()
	driver.IPAddress = ""

	observed, err := driver.GetIP()
	expected := ""

	errorData := "IPAddress is empty"
	mockError := errors.New(errorData)

	assert.Equal(t, expected, observed, "Incorrect IP address")
	assert.Error(t, err)
	assert.EqualError(t, err, mockError.Error())
}

func TestGetStateRunning(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("GetMachineDetails", "cdi-test", driver.MachineUUID, mockKeycloak.GetToken()).Return(
		models.ExpectedLanports,
		"902cc002-3775-4be0-be00-535a677b2ab4",
		13,
		nil)

	observed, _ := driver.GetState()
	expected := state.Running

	assert.Equal(t, expected, observed, "Incorrect state")
}

func TestGetStateUnknownStatus(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("GetMachineDetails", "cdi-test", driver.MachineUUID, mockKeycloak.GetToken()).Return(
		models.ExpectedLanports,
		"902cc002-3775-4be0-be00-535a677b2ab4",
		987,
		nil)

	observed, err := driver.GetState()
	expected := state.None

	assert.Equal(t, expected, observed, "Incorrect state")
	assert.NoError(t, err)
}

func TestGetStateEmptyMachineUuid(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = ""

	observed, err := driver.GetState()
	expected := state.Error

	assert.Equal(t, expected, observed, "Incorrect state")
	assert.Error(t, err)
}

func TestGetURLEmptyIP(t *testing.T) {
	driver := NewDriver()

	observed, _ := driver.GetURL()
	expected := ""

	assert.Equal(t, expected, observed, "Incorrect url")
}

func TestGetURLSuccess(t *testing.T) {
	// Arrange: Create a Driver instance with a valid IP address.
	driver := NewDriver()
	driver.IPAddress = "192.168.122.41"

	// Act: Call the GetURL method.
	url, err := driver.GetURL()

	// Assert: Check for expected results.
	assert.NoError(t, err)
	assert.Equal(t, "tcp://192.168.122.41:2376", url)
}

func TestCheckConfigTenantSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	hostPublicKey := "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="

	driver := &Driver{
		BaseDriver:                &drivers.BaseDriver{},
		FabricManager:             mockFM,
		Keycloak:                  mockKeycloak,
		SSHPassword:               "pass",
		ComputeConditionsJson:     "test",
		NetworkBaremetalPort:      1,
		NetworkBaremetalUUID:      "test",
		NetworkBaremetalDefaultGW: "192.168.0.254",
		NetworkProvisionPort:      1,
		NetworkProvisionUUID:      "test",
		NetworkProvisionDefaultGW: "192.168.0.254",
		DevicesSpecJson:           models.DeviceSpecsValid,
		ApiUrl:                    "http://192.168.0.1",
		TenantUuid:                "cdi-test",
		OsImageName:               "Ubuntu",
		UserDataFile:              "userData.json",
		OsImageSshHostPubKey:      hostPublicKey,
	}
	driver.SSHUser = "user"

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("ValidateTenant", "cdi-test", mockKeycloak.GetToken()).Return(nil)

	err := driver.checkConfig()
	assert.NoError(t, err)
	mockFM.AssertCalled(t, "ValidateTenant", "cdi-test", models.AccessTokenExample)
}

func TestCheckConfig_SlesParamsFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	hostPublicKey := "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNlLkDgzQ7FWYLi7wl3ljvaF/n0FEpSrML23hJjvv3HfEvNJxNbjm1GomnefDM9/qYV2pRAganbMMnCG8gs7KD8="

	driver := &Driver{
		BaseDriver:                &drivers.BaseDriver{},
		FabricManager:             mockFM,
		Keycloak:                  mockKeycloak,
		SSHPassword:               "pass",
		ComputeConditionsJson:     "test",
		NetworkBaremetalPort:      1,
		NetworkBaremetalUUID:      "test",
		NetworkBaremetalDefaultGW: "192.168.0.254",
		NetworkProvisionPort:      1,
		NetworkProvisionUUID:      "test",
		NetworkProvisionDefaultGW: "192.168.0.254",
		DevicesSpecJson:           models.DeviceSpecsValid,
		ApiUrl:                    "http://192.168.0.1",
		TenantUuid:                "cdi-test",
		OsImageName:               "Ubuntu",
		UserDataFile:              "userData.json",
		OsImageSshHostPubKey:      hostPublicKey,
		SlesRegistrationCode:      "123",
	}
	driver.SSHUser = "user"

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("ValidateTenant", "cdi-test", mockKeycloak.GetToken()).Return(nil)

	testCases := []struct {
		name     string
		input    func()
		expected string
	}{
		{name: "registration code is present but no email",
			input: func() {
				driver.SlesRegistrationEmail = ""
			},
			expected: "when SLES registration code is not empty then SLES registration email must also not be empty. Fill in param --fsas-sles-registration-email",
		},
		{name: "registration code is present but invalid email",
			input: func() {
				driver.SlesRegistrationEmail = "some#invalid.email"
			},
			expected: "Email address is not valid: some#invalid.email",
		},
		{name: "registration code is present but invalid email 2",
			input: func() {
				driver.SlesRegistrationEmail = "alice@from.wonder,land"
			},
			expected: "Email address is not valid: alice@from.wonder,land",
		},
		{name: "registration code is present but invalid email 3",
			input: func() {
				driver.SlesRegistrationEmail = "alice@"
			},
			expected: "Email address is not valid: alice@",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input()
			err := driver.checkConfig()
			assert.ErrorContains(t, err, tc.expected)
		})
	}
	mockFM.AssertCalled(t, "ValidateTenant", "cdi-test", models.AccessTokenExample)
}

func TestCheckConfigTenantFailed(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:                &drivers.BaseDriver{},
		FabricManager:             mockFM,
		Keycloak:                  mockKeycloak,
		SSHPassword:               "pass",
		ComputeConditionsJson:     "test",
		NetworkBaremetalPort:      1,
		NetworkBaremetalUUID:      "test",
		NetworkBaremetalDefaultGW: "192.168.0.254",
		NetworkProvisionPort:      1,
		NetworkProvisionUUID:      "test",
		NetworkProvisionDefaultGW: "192.168.0.254",
		DevicesSpecJson:           models.DeviceSpecsValid,
		ApiUrl:                    "http://192.168.0.1",
		TenantUuid:                "cdi-test",
		OsImageName:               "Ubuntu",
		UserDataFile:              "userData.json",
	}
	driver.SSHUser = "user"

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("ValidateTenant", "cdi-test", mockKeycloak.GetToken()).Return(fmt.Errorf("request failed"))

	err := driver.checkConfig()
	assert.Error(t, err)
	assert.EqualError(t, err, "request failed")
	mockFM.AssertCalled(t, "ValidateTenant", "cdi-test", models.AccessTokenExample)
}

func TestCheckConfigSSHUserFailed(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		SSHPassword:           "pass",
		ComputeConditionsJson: "test",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "test",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "test",
		DevicesSpecJson:      "test",
		ApiUrl:               "http://192.168.0.1",
		TenantUuid:           "cdi-test",
		OsImageName:          "Ubuntu",
	}

	err := driver.checkConfig()
	assert.Error(t, err)
	assert.EqualError(t, err, "SSH user must be specified using the CLI option --fsas-ssh-user")
}

func TestStartSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	driver.MachineUUID = "59756ed2-6a42-47f2-bc54-117bcf6bdce3"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport(nil), "", 13, nil)
	mockFM.On("PowerOn", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)

	err := driver.Start()
	assert.NoError(t, err)
}

// Start Error - fm client returns an error
func TestStartFailFmClientError(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	driver.MachineUUID = "59756ed2-6a42-47f2-bc54-117bcf6bdce3"

	errorData := `{"status": 500, "detail": {"code": "E020001", "message": "Exception occured, "data": {}}}`

	mockError := errors.New(errorData)

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	mockFM.On("PowerOn", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(mockError)

	err := driver.Start()

	assert.Error(t, err)
	assert.EqualError(t, err, mockError.Error())
}

// Start Error - keycloak init fails
func TestStartFailKeycloakInitError(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	driver.MachineUUID = "59756ed2-6a42-47f2-bc54-117bcf6bdce3"

	errorData := "none of the arguments can be empty; neither 'Realm', 'User', 'Password', 'BaseURI' or 'Port'"

	mockError := errors.New(errorData)

	mockKeycloak.On("IsInit").Return(false)

	err := driver.Start()

	assert.Error(t, err)
	assert.EqualError(t, err, mockError.Error())
}

// Start Error - GetState returns an error
func TestStartGetStateError(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	driver.MachineUUID = "59756ed2-6a42-47f2-bc54-117bcf6bdce3"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockFM.On("PowerOn", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(
		models.ExpectedLanports,
		"902cc002-3775-4be0-be00-535a677b2ab4",
		987,
		nil)

	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)
	mockClock.On("Since", mock_now_time).Return(WAIT_FOR_STATUS_TIMEOUT + time.Microsecond*100)

	err := driver.Start()

	errorData := "error: required status was not achieved within the specified time"
	mockError := errors.New(errorData)
	assert.Error(t, err)
	assert.EqualError(t, err, mockError.Error())
}

// Start Error - MachineUUID empty
func TestStartFail(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	driver.MachineUUID = ""

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)

	err := driver.Start()

	assert.Error(t, err)
	assert.EqualError(t, err, "machine uuid is empty")
}

func TestRemoveSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSshManager := sshMock.NewMockSshManager(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{IPAddress: "192.168.122.55"},
		FabricManager: mockFM,
		SshManager:    mockSshManager,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockSshManager.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockSshManager.On("DeregisterOS").Return(nil)
	mockFM.On("GetMachineDetails", "cdi-test", driver.MachineUUID, models.AccessTokenExample).Return(
		models.ExpectedLanports,
		"902cc002-3775-4be0-be00-535a677b2ab4",
		17,
		nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, "cdi-test", models.AccessTokenExample).Return(nil)

	err := driver.Remove()
	assert.NoError(t, err)

}

func TestRemoveClientsNotInitializedFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
		MachineUUID:   "0e3e2a7c-f0d7-43fb-ade0-a835c326864b",
	}

	mockKeycloak.On("IsInit").Return(false)
	err := driver.Remove()
	assert.ErrorIs(t, err, keycloak.ErrNoneOfConstructorArgsCanBeEmpty)
}

func TestRemoveMachineUUIDIsEmptyFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = ""

	err := driver.Remove()
	assert.NoError(t, err)
}

func TestRemoveFMRemoveMachineFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSshManager := sshMock.NewMockSshManager(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{IPAddress: "192.168.122.55"},
		FabricManager: mockFM,
		SshManager:    mockSshManager,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockSshManager.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockSshManager.On("DeregisterOS").Return(nil)
	expectedError := fmt.Errorf("request failed")
	mockFM.On("RemoveMachine", driver.MachineUUID, "cdi-test", models.AccessTokenExample).Return(expectedError)

	err := driver.Remove()
	assert.ErrorIs(t, err, expectedError)
}

func TestRemoveDeregisterOSFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSshManager := sshMock.NewMockSshManager(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{IPAddress: "192.168.122.55"},
		FabricManager: mockFM,
		SshManager:    mockSshManager,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockSshManager.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockError := fmt.Errorf("Deregister mock fail")
	// This error should only notify via WARN log as not removing machine can be disastrous
	mockSshManager.On("DeregisterOS").Return(mockError)
	mockFM.On("GetMachineDetails", "cdi-test", driver.MachineUUID, models.AccessTokenExample).Return(
		models.ExpectedLanports,
		"902cc002-3775-4be0-be00-535a677b2ab4",
		17,
		nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, "cdi-test", models.AccessTokenExample).Return(nil)

	err := driver.Remove()
	assert.NoError(t, err)
}

func TestInitClientsSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)

	err := driver.initClients()
	assert.NoError(t, err)
}

func TestInitClientsFailKeycloak(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(false)

	err := driver.initClients()
	assert.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrNoneOfConstructorArgsCanBeEmpty)
}

func TestInitClientsFailFM(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(false)

	err := driver.initClients()
	assert.Error(t, err)
	assert.ErrorContains(t, err, fm.ErrMissingParams)
}

func TestInitKeycloakSuccess(t *testing.T) {
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{},
		Keycloak:   mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true)
	err := driver.initKeycloak()
	assert.NoError(t, err)
}

func TestInitKeycloakFail(t *testing.T) {
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{},
		Keycloak:   mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(false)
	err := driver.initKeycloak()
	assert.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrNoneOfConstructorArgsCanBeEmpty)
}

func TestInitFabricManagerPass(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
	}

	mockFM.On("IsInit").Return(true)

	err := driver.initFabricManager()
	assert.NoError(t, err)
}

func TestInitFabricManagerFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
	}

	mockFM.On("IsInit").Return(false)

	err := driver.initFabricManager()
	assert.Error(t, err)
	assert.ErrorContains(t, err, fm.ErrMissingParams)
}

func TestWaitForStatusCorrectStatus(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		TenantUuid:    "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
	}

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return("token")
	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(
		models.ExpectedLanports,
		"3129cbdf-345c-43a9-b4dc-34880ceed63d",
		13,
		nil)
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)

	err := driver.waitForStatus(ACTIVE_PON, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_TIMEOUT)

	mockClock.AssertCalled(t, "Now")
	mockClock.AssertNotCalled(t, "Since", mock.Anything)
	assert.NoError(t, err)
}

func TestWaitForStatusCorrectSecondCallStatus(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		TenantUuid:    "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
	}

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return("token")
	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 13, nil).Once()
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mock_duration := time.Millisecond * 100
	mockClock.On("Now").Return(mock_now_time)
	mockClock.On("Since", mock_now_time).Return(mock_duration)
	mockClock.On("Sleep", WAIT_FOR_STATUS_STEP).Return(nil)

	err := driver.waitForStatus(ACTIVE_PON, WAIT_FOR_STATUS_STEP, WAIT_FOR_STATUS_TIMEOUT)

	assert.NoError(t, err)
	mockClock.AssertCalled(t, "Now")
	mockClock.AssertCalled(t, "Since", mock_now_time)
	mockClock.AssertCalled(t, "Sleep", WAIT_FOR_STATUS_STEP)
	mockClock.AssertNumberOfCalls(t, "Since", 1)
	mockClock.AssertExpectations(t)
}

func TestWaitForStatusTimeout(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		TenantUuid:    "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
	}

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return("token")
	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil)
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mock_time_step := time.Second * 1
	mockClock.On("Now").Return(mock_now_time)
	mockClock.On("Since", mock_now_time).Return(time.Millisecond * 100).Once()
	mockClock.On("Since", mock_now_time).Return(time.Millisecond*100 + mock_time_step*1).Once()
	mockClock.On("Since", mock_now_time).Return(time.Millisecond*100 + mock_time_step*2).Once()
	mockClock.On("Sleep", mock_time_step)

	err := driver.waitForStatus(ACTIVE_PON, mock_time_step, 2*mock_time_step)

	assert.EqualError(t, err, "error: required status was not achieved within the specified time")
	mockClock.AssertExpectations(t)
	mockClock.AssertNumberOfCalls(t, "Since", 3)
	mockClock.AssertNumberOfCalls(t, "Sleep", 2)
}

func TestWaitForStatusError(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		TenantUuid:    "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
	}
	mockError := fmt.Errorf("Request GET /machines/a1b2c3d4-e5f6-7890-1234-567890abcdef failed")

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return("token")
	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return([]models.Lanport(nil), "", 0, mockError)

	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)

	err := driver.waitForStatus(ACTIVE_PON, time.Duration(1*time.Second), time.Duration(2*time.Second))

	assert.EqualError(t, err, "error getting state: Request GET /machines/a1b2c3d4-e5f6-7890-1234-567890abcdef failed")
	mockClock.AssertExpectations(t)
	mockClock.AssertNotCalled(t, "Since", mock_now_time)
}

func TestCreate(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		CfgManager:            mockCfg,
		MachineUUID:           testMachineUUID,
		UserDataFile:          "custom-user-data.yaml",
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
		SlesRegistrationCode:  "somecode010101",
		SlesRegistrationEmail: "hoge@example.com",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockCfg.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,
		NetworkBaremetalPort:  driver.NetworkBaremetalPort,
		NetworkBaremetalUUID:  driver.NetworkBaremetalUUID,
		NetworkProvisionPort:  driver.NetworkProvisionPort,
		NetworkProvisionUUID:  driver.NetworkProvisionUUID,
		NtpServer:             driver.NtpUrl,
		DnsServer:             driver.DnsIp,
	}

	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	//waitForStatus
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	mockSSH.On("RegisterOS", driver.SlesRegistrationCode, driver.SlesRegistrationEmail).Return(nil)
	mockSSH.On("ExchangeKeys").Return(nil)
	mockRKE2ScriptContent := "script-content-rke2"
	mockCfg.On("PrepareRke2ConfigScript", "100-fsas-providerid", "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f").Return(mockRKE2ScriptContent)
	mockSSH.On("ExecuteScript", "", mockRKE2ScriptContent, true, true).Return(nil).Once()
	// applyCloudInit
	userdataPath := filepath.Join(cloudInitDirPath, "user-data")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")
	mockSSH.On("WriteFileOnRemoteMachine", userdataPath, mockRKE2ScriptContent, fs.FileMode(0700)).Return(nil)
	mockCfg.On("PrepareMetadata", testMachineUUID, driver.MachineName).Return("")
	mockSSH.On("WriteFileOnRemoteMachine", metadataPath, "", fs.FileMode(0700)).Return(nil)
	mockSSH.On("RebootCloudInit").Return(nil)
	mockSSH.On("DisablePasswordSSHLogin").Return(nil)
	mockClock.On("Sleep", WAIT_FOR_START_AFTER_REBOOT).Return(nil)
	applyMockForExtendedUserMethods(mockCfg)

	// Mock implementation of os.ReadFile
	originalOsReadFile := osReadFile
	defer func() { osReadFile = originalOsReadFile }()
	osReadFile = func(path string) ([]byte, error) {
		return []byte("script-content-rke2"), nil
	}

	err := driver.Create()
	assert.NoError(t, err)
}

func applyMockForExtendedUserMethods(mockCfg *cfgMock.MockCfgManager) {
	mockCfg.On("ExtendUserdataRunCmd", []string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
		`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`,
	}).Return(nil).Once()
	items := []cfgutils.CloudConfigItem{
		cfgutils.NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
		cfgutils.NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files-2.log", "Cloud config succeeded for write_files part 2"),
	}
	mockCfg.On("ExtendUserdataWriteFiles", items).Return(nil).Once()
}

func TestCreateCloudInitFail(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		CfgManager:            mockCfg,
		MachineUUID:           testMachineUUID,
		UserDataFile:          "custom-user-data.yaml",
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
		SlesRegistrationCode:  "somecode010101",
		SlesRegistrationEmail: "hoge@example.com",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockCfg.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,
		NetworkBaremetalPort:  driver.NetworkBaremetalPort,
		NetworkBaremetalUUID:  driver.NetworkBaremetalUUID,
		NetworkProvisionPort:  driver.NetworkProvisionPort,
		NetworkProvisionUUID:  driver.NetworkProvisionUUID,
		NtpServer:             driver.NtpUrl,
		DnsServer:             driver.DnsIp,
	}

	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	//waitForStatus
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	mockSSH.On("RegisterOS", driver.SlesRegistrationCode, driver.SlesRegistrationEmail).Return(nil)
	mockSSH.On("ExchangeKeys").Return(nil)
	mockRKE2ScriptContent := "script-content-rke2"
	mockCfg.On("PrepareRke2ConfigScript", "100-fsas-providerid", "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f").Return(mockRKE2ScriptContent)
	mockSSH.On("ExecuteScript", "", mockRKE2ScriptContent, true, true).Return(nil).Once()
	// applyCloudInit
	userdataPath := filepath.Join(cloudInitDirPath, "user-data")
	mockSSH.On("WriteFileOnRemoteMachine", userdataPath, "custom-user-data.yaml", fs.FileMode(0700)).Return(fmt.Errorf("WriteFileOnRemoteMachine failed"))
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 17, nil).Once()
	applyMockForExtendedUserMethods(mockCfg)

	// Mock implementation of os.ReadFile
	originalOsReadFile := osReadFile
	defer func() { osReadFile = originalOsReadFile }()
	osReadFile = func(path string) ([]byte, error) {
		return []byte("custom-user-data.yaml"), nil
	}

	err := driver.Create()
	assert.EqualError(t, err, errors.New("WriteFileOnRemoteMachine failed").Error())
}

func TestCreateInitClientsFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(false)

	err := driver.Create()

	assert.EqualError(t, err, "none of the arguments can be empty; neither 'Realm', 'User', 'Password', 'BaseURI' or 'Port'")
	mockKeycloak.AssertNumberOfCalls(t, "IsInit", 1) // UUID is empty, Remove will finish without error, with warn
}

func TestCreateMachineFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{IPAddress: "192.168.122.55"},
		FabricManager:         mockFM,
		SshManager:            mockSSH,
		Keycloak:              mockKeycloak,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}

	testError := fmt.Errorf("CreateMachine unsucessfull")
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return("", testError)
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, "", int(UNBUILDED), nil)

	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)

	err := driver.Create()
	expectedError := "CreateMachine unsucessfull"
	assert.EqualError(t, err, expectedError)
}

func TestCreateWaitForStatusFail(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", int(UNBUILDED), nil)

	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)
	mockClock.On("Since", mock_now_time).Return(WAIT_FOR_STATUS_TIMEOUT + time.Microsecond*100)

	err := driver.Create()

	assert.ErrorContains(t, err, "error: required status was not achieved within the specified time")
	mockClock.AssertNumberOfCalls(t, "Since", 1)
}

func TestCreateGetMachineDetailsFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// waitForStatus call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	testError := fmt.Errorf("GetMachineDetails unsucessfull")
	// bootSSD call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, testError).Once()
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// waitForStatus call in RemoveMachine
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, testError.Error())
}

func TestCreateImageInstallFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// Create's 1st waitForStatus and 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	testError := fmt.Errorf("ImageInstall unsucessfull")
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(testError)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// Call in Remove
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, testError.Error())
}

func TestCreateStartFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st waitForStatus and 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 3rd waitForStatus after (OS_INSTALLING)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	// 4th waitForStatus after OS is installed
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	testError := fmt.Errorf("PowerOn unsucessfull")
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(testError)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// last waitForStatus in Remove
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, testError.Error())

}

func TestCreateGetMachineDetailsFailSecond(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",

		NetworkBaremetalPort: 1,
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",

		NetworkProvisionPort: 1,
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
		NtpUrl:               "test",
		DnsIp:                "test",
	}
	driver.MachineName = "machineNameTest"

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st call after Create and 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 2 OS installation calls (installed and installed check)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// PowerOn waitForStatus
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Once()
	// IP addresses call
	testError := fmt.Errorf("GetMachineDetails unsucessfull")
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, testError).Once()
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// Remove call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, testError.Error())
}

func TestCreateExchangeKeysFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st call after Create, 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 2 OS installation related checks
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// PowerOn waitForStatus check && Lanports reading check
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	testError := fmt.Errorf("ExchangeKeys unsuccessful")
	mockSSH.On("IsInit").Return(true)
	mockSSH.On("ExchangeKeys").Return(testError)
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// waitForStatus in Remove call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, testError.Error())
}

func TestCreateOSRegistrationFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		CfgManager:            mockCfg,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
		SlesRegistrationCode:  "somecode010101",
		SlesRegistrationEmail: "hoge@example.com",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st call after Create, 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 2 OS installation related checks
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// PowerOn waitForStatus check && Lanports reading check
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	mockSSH.On("ExchangeKeys").Return(nil)
	mockError := fmt.Errorf("Registration failed")
	mockSSH.On("RegisterOS", driver.SlesRegistrationCode, driver.SlesRegistrationEmail).Return(mockError)
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// waitForStatus in Remove call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)

	err := driver.Create()
	assert.EqualError(t, err, mockError.Error())
}

func TestCreateExecuteScriptFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		CfgManager:            mockCfg,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
		SlesRegistrationCode:  "somecode010101",
		SlesRegistrationEmail: "hoge@example.com",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockCfg.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st call after Create, 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 2 OS installation related checks
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// PowerOn waitForStatus check && Lanports reading check
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	mockSSH.On("RegisterOS", driver.SlesRegistrationCode, driver.SlesRegistrationEmail).Return(nil)
	mockSSH.On("ExchangeKeys").Return(nil)
	mockRKE2ScriptContent := "test RKE2 script content"
	mockCfg.On("PrepareRke2ConfigScript", "100-fsas-providerid", testMachineUUID).Return(mockRKE2ScriptContent)
	mockError := fmt.Errorf("ExecuteScript unsuccessful")
	mockSSH.On("ExecuteScript", "", mockRKE2ScriptContent, true, true).Return(mockError)
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// waitForStatus in Remove call
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return([]models.Lanport{}, "", 17, nil)
	applyMockForExtendedUserMethods(mockCfg)

	err := driver.Create()
	assert.EqualError(t, err, mockError.Error())
}

func TestCreateFailRemoveFail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)
	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:            &drivers.BaseDriver{},
		FabricManager:         mockFM,
		Keycloak:              mockKeycloak,
		SshManager:            mockSSH,
		CfgManager:            mockCfg,
		MachineUUID:           testMachineUUID,
		TenantUuid:            "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		ComputeConditionsJson: "testJsnn",
		DevicesSpecJson:       "testJson",
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NetworkProvisionPort:  1,
		NetworkProvisionUUID:  "123e4567-e89b-12d3-a456-426614174000",
		NtpUrl:                "test",
		DnsIp:                 "test",
		SlesRegistrationCode:  "somecode010101",
		SlesRegistrationEmail: "hoge@example.com",
	}
	driver.MachineName = "machineNameTest"

	mockSSH.On("IsInit").Return(true)
	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("IsInit").Return(true)
	mockCfg.On("IsInit").Return(true)

	machineSpecArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: driver.ComputeConditionsJson,
		DevicesSpecJson:       driver.DevicesSpecJson,

		NetworkBaremetalPort: driver.NetworkBaremetalPort,
		NetworkBaremetalUUID: driver.NetworkBaremetalUUID,

		NetworkProvisionPort: driver.NetworkProvisionPort,
		NetworkProvisionUUID: driver.NetworkProvisionUUID,
		NtpServer:            driver.NtpUrl,
		DnsServer:            driver.DnsIp,
	}
	mockFM.On("CreateMachine", driver.MachineName, driver.TenantUuid, machineSpecArgs, models.AccessTokenExample).Return(testMachineUUID, nil)
	// 1st call after Create, 2nd call for bootSSD
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Twice()
	mockFM.On("ImageInstall", driver.TenantUuid, bootSsdUUID, driver.OsImageName, models.AccessTokenExample).Return(nil)
	// 2 OS installation related checks
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 18, nil).Once()
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 15, nil).Once()
	mockFM.On("PowerOn", testMachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)
	// PowerOn waitForStatus check && Lanports reading check
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, bootSsdUUID, 13, nil).Twice()
	mockSSH.On("RegisterOS", driver.SlesRegistrationCode, driver.SlesRegistrationEmail).Return(nil)
	mockSSH.On("ExchangeKeys").Return(nil)
	mockRKE2ScriptContent := "test RKE2 script content"
	mockCfg.On("PrepareRke2ConfigScript", "100-fsas-providerid", testMachineUUID).Return(mockRKE2ScriptContent)
	mockError := fmt.Errorf("ExecuteScript unsuccessful")
	mockSSH.On("ExecuteScript", "", mockRKE2ScriptContent, true, true).Return(mockError)
	removeError := fmt.Errorf("Remove after failed inner Create failed as well")
	mockSSH.On("DeregisterOS").Return(nil)
	mockFM.On("RemoveMachine", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(removeError)
	applyMockForExtendedUserMethods(mockCfg)

	err := driver.Create()
	assert.EqualError(t, err, "error during Create: 'ExecuteScript unsuccessful'; followed by error during Remove: 'Remove after failed inner Create failed as well'")
}

func TestKill_success(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("PowerOff", driver.MachineUUID, "cdi-test", models.AccessTokenExample).Return(nil)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil)

	err := driver.Kill()
	assert.NoError(t, err)
}

func TestKill_machineUUID_isEmpty_fail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = ""

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)

	err := driver.Kill()
	assert.ErrorContains(t, err, "machine uuid is empty")
}

func TestKill_FMPowerOff_fail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		TenantUuid:    "cdi-test",
	}
	driver.MachineUUID = "ddb3e14d-b9c8-4500-8377-073ad43a5ff7"

	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	expectedError := fmt.Errorf("request failed")
	mockFM.On("PowerOff", driver.MachineUUID, "cdi-test", models.AccessTokenExample).Return(expectedError)

	err := driver.Kill()
	assert.ErrorIs(t, err, expectedError)
}

func TestStop_success(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil)

	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(nil)

	err := driver.Stop()
	assert.NoError(t, err)

}

func TestStop_GracefulShutdown_failed(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true).Maybe()
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	shutdownErr := fmt.Errorf("FM graceful shutdown failed")
	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(shutdownErr).Once()

	err := driver.Stop()
	assert.ErrorIs(t, err, shutdownErr)
}

func TestStop_waitForStatus_failed(t *testing.T) {

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}

	mockKeycloak.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	mockFM.On("IsInit").Return(true)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 99, nil)

	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(nil)

	err := driver.Stop()
	assert.ErrorContains(t, err, "required status was not achieved within the specified time")
}

func TestRestartSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	// Stop
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil).Once()
	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(nil)

	// Start
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 13, nil).Once()
	mockFM.On("PowerOn", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)

	err := driver.Restart()
	assert.NoError(t, err)

}

func TestRestartFail_Stop(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	// No Start on Stop Failure

	// Stop Fail - FM
	shutdownError := fmt.Errorf("FM graceful shutdown failed")
	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(shutdownError).Once()

	err := driver.Restart()
	assert.ErrorIs(t, err, shutdownError)

	// Stop Fail - Status
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 99, nil).Once()
	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(nil).Once()

	err = driver.Restart()
	assert.ErrorContains(t, err, "required status was not achieved within the specified time")

}

func TestRestartFail_Start(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)

	driver := &Driver{
		BaseDriver: &drivers.BaseDriver{
			IPAddress: "10.1.2.3",
			SSHUser:   "user-1",
		},
		SSHPassword:   "password1",
		MachineUUID:   "cdd792f2-5591-4c18-a8bd-1c39e55dedfa",
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
	}
	mockKeycloak.On("IsInit").Return(true)
	mockFM.On("IsInit").Return(true)
	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)

	// Stop
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, mockKeycloak.GetToken()).Return(models.ExpectedLanports, "3129cbdf-345c-43a9-b4dc-34880ceed63d", 15, nil)
	// Normal UUID
	mockFM.On("GracefulShutdown", driver.MachineUUID, "", models.AccessTokenExample).Return(nil).Once()
	// Empty UUID
	mockFM.On("GracefulShutdown", "", "", models.AccessTokenExample).Return(nil).Maybe()

	// Start Fail - Status
	// mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(
	// 	models.ExpectedLanports,
	// 	"902cc002-3775-4be0-be00-535a677b2ab4",
	// 	987,
	// 	nil)
	mockFM.On("PowerOn", driver.MachineUUID, driver.TenantUuid, models.AccessTokenExample).Return(nil)

	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock
	mock_now_time := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(mock_now_time)
	mockClock.On("Since", mock_now_time).Return(WAIT_FOR_STATUS_TIMEOUT + time.Microsecond*100)

	err := driver.Restart()

	assert.Error(t, err)
	errorData := "error: required status was not achieved within the specified time"
	assert.EqualError(t, err, errorData)

	// Start Fail - Empty UUID
	driver.MachineUUID = ""

	err = driver.Restart()

	assert.Error(t, err)
	assert.ErrorContains(t, err, "machine uuid is empty")

}

func TestAssignIpAddressesSuccess(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:           &drivers.BaseDriver{},
		FabricManager:        mockFM,
		Keycloak:             mockKeycloak,
		TenantUuid:           "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		NetworkBaremetalUUID: "78901234-5678-9abc-def0-1234567890ab",
		NetworkProvisionUUID: "123e4567-e89b-12d3-a456-426614174000",
	}

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(
		models.ExpectedLanports, bootSsdUUID, 13, nil)

	err := driver.assignIpAddresses()

	assert.Equal(t, "192.168.2.100", driver.IPAddress)
	assert.Equal(t, "10.0.0.100", driver.PrivateIPAddress)
	assert.NoError(t, err)
}

func TestAssignIpAddressesFailed(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	bootSsdUUID := "3129cbdf-345c-43a9-b4dc-34880ceed63d"
	driver := &Driver{
		BaseDriver:           &drivers.BaseDriver{},
		FabricManager:        mockFM,
		Keycloak:             mockKeycloak,
		TenantUuid:           "4a9587f0-e7da-4824-8127-d5ca5ddf8c34",
		NetworkBaremetalUUID: "6aaf2935-6a66-4f29-8dcc-1367688960ea",
		NetworkProvisionUUID: "f7294e52-228a-4ef1-b9ca-3d3402e49cf6",
	}

	mockKeycloak.On("GetToken").Return(models.AccessTokenExample)
	mockFM.On("GetMachineDetails", driver.TenantUuid, driver.MachineUUID, models.AccessTokenExample).Return(
		models.ExpectedLanports, bootSsdUUID, 13, nil)

	errorData := "IPAddress must not be empty"
	mockError := errors.New(errorData)

	err := driver.assignIpAddresses()

	assert.Error(t, err)
	assert.EqualError(t, err, mockError.Error())
}

func Test_applyCloudInit_success(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		CfgManager:    mockCfg,
		SshManager:    mockSSH,
		MachineUUID:   testMachineUUID,
		UserDataFile:  "",
	}

	mockCfg.On("PrepareMetadata", testMachineUUID, "a20-pool1-d5h97-lmjkr").Return("")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")
	mockSSH.On("WriteFileOnRemoteMachine", metadataPath, "", fs.FileMode(0700)).Return(nil)
	mockSSH.On("RebootCloudInit").Return(nil)
	mockClock.On("Sleep", WAIT_FOR_START_AFTER_REBOOT).Return(nil)

	testhostname := "a20-pool1-d5h97-lmjkr"
	err := driver.applyCloudInit(testhostname)
	assert.NoError(t, err)
}

func Test_applyCloudInit_fail_write_file(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		CfgManager:    mockCfg,
		SshManager:    mockSSH,
		MachineUUID:   testMachineUUID,
		UserDataFile:  "",
	}

	mockCfg.On("PrepareMetadata", testMachineUUID, "a20-pool1-d5h97-lmjkr").Return("")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")
	mockSSH.On("WriteFileOnRemoteMachine", metadataPath, "", fs.FileMode(0700)).Return(fmt.Errorf("WriteFileOnRemoteMachine failed"))

	testhostname := "a20-pool1-d5h97-lmjkr"
	err := driver.applyCloudInit(testhostname)
	assert.EqualError(t, err, errors.New("WriteFileOnRemoteMachine failed").Error())
}

func Test_applyCloudInit_fail_reboot_cloudinit(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		CfgManager:    mockCfg,
		SshManager:    mockSSH,
		MachineUUID:   testMachineUUID,
		UserDataFile:  "",
	}

	mockCfg.On("PrepareMetadata", testMachineUUID, "a20-pool1-d5h97-lmjkr").Return("")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")
	mockSSH.On("WriteFileOnRemoteMachine", metadataPath, "", fs.FileMode(0700)).Return(nil)
	mockSSH.On("RebootCloudInit").Return(fmt.Errorf("RebootCloudInit failed"))

	testhostname := "a20-pool1-d5h97-lmjkr"
	err := driver.applyCloudInit(testhostname)
	assert.EqualError(t, err, errors.New("RebootCloudInit failed").Error())
}

func Test_applyCloudInit_success_with_userdata(t *testing.T) {
	mockClock := timeutilsmock.NewMockClock(t)
	statusClock = mockClock

	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		CfgManager:    mockCfg,
		SshManager:    mockSSH,
		MachineUUID:   testMachineUUID,
		UserDataFile:  "test_user_data",
	}

	userdataPath := filepath.Join(cloudInitDirPath, "user-data")
	mockSSH.On("WriteFileOnRemoteMachine", userdataPath, "script-content-rke2", fs.FileMode(0700)).Return(nil)
	mockCfg.On("PrepareMetadata", testMachineUUID, "a20-pool1-d5h97-lmjkr").Return("")
	metadataPath := filepath.Join(cloudInitDirPath, "meta-data")
	mockSSH.On("WriteFileOnRemoteMachine", metadataPath, "", fs.FileMode(0700)).Return(nil)
	mockSSH.On("RebootCloudInit").Return(nil)
	mockClock.On("Sleep", WAIT_FOR_START_AFTER_REBOOT).Return(nil)

	originalOsReadFile := osReadFile
	defer func() { osReadFile = originalOsReadFile }()
	osReadFile = func(path string) ([]byte, error) {
		return []byte("script-content-rke2"), nil
	}

	testhostname := "a20-pool1-d5h97-lmjkr"
	err := driver.applyCloudInit(testhostname)
	assert.NoError(t, err)
}

func Test_applyCloudInit_success_with_userdata_fail(t *testing.T) {
	mockFM := fmmock.NewMockFabricManager(t)
	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	mockSSH := sshMock.NewMockSshManager(t)
	mockCfg := cfgMock.NewMockCfgManager(t)

	testMachineUUID := "ff3a4a18-1ef9-4e17-9c8d-eec35b3c638f"
	driver := &Driver{
		BaseDriver:    &drivers.BaseDriver{},
		FabricManager: mockFM,
		Keycloak:      mockKeycloak,
		CfgManager:    mockCfg,
		SshManager:    mockSSH,
		MachineUUID:   testMachineUUID,
		UserDataFile:  "test_user_data",
	}

	userdataPath := filepath.Join(cloudInitDirPath, "user-data")
	mockSSH.On("WriteFileOnRemoteMachine", userdataPath, "script-content-rke2", fs.FileMode(0700)).Return(fmt.Errorf("WriteFileOnRemoteMachine failed"))

	originalOsReadFile := osReadFile
	defer func() { osReadFile = originalOsReadFile }()
	osReadFile = func(path string) ([]byte, error) {
		return []byte("script-content-rke2"), nil
	}

	testhostname := "a20-pool1-d5h97-lmjkr"
	err := driver.applyCloudInit(testhostname)
	assert.EqualError(t, err, errors.New("WriteFileOnRemoteMachine failed").Error())
}
