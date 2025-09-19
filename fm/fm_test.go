package fm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"testing"

	httputils "github.com/fujitsu/docker-machine-driver-fsas/httputils/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {

	/* setup code here */

	originalLogger := slog.Default()
	// Suppress slog output in test
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	exitCode := m.Run() // run tests

	/* tear-down code here */
	// Restore original logger
	slog.SetDefault(originalLogger)

	os.Exit(exitCode)
}

// Successful scenario
func TestCreateMachine_Success(t *testing.T) {

	expectedMachineUUID := "59756ed2-6a42-47f2-bc54-117bcf6bdce3"
	machineName := "test-machine-001"
	tenantId := "b3b65e79-ad41-4367-89d6-e4e7315141ef"

	machineSpecsArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: `[{"column": "model","operator": "eq","value": "PRIMERGY-RX2540M6"}]`,
		DevicesSpecJson:       `[{"res_type":"storage","res_num":1,"tags":{"is_bootstorage":true},"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`,
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "75e6b24f-c1cc-4009-a871-b5828a468f4f",
		NetworkProvisionPort:  2,
		NetworkProvisionUUID:  "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NtpServer:             "ntp.example.com",
		DnsServer:             "8.8.8.8",
	}

	getEntryData := func(index int) models.MachinesRequestResponse {
		var entryData models.MachinesRequestResponse
		json.Unmarshal([]byte(models.PostMachinesResponseExample), &entryData)

		switch index {
		case 0:
			// happy path
			return entryData
		case 1:
			return func() models.MachinesRequestResponse {
				entryData.Data.Machines[0].MachineUUID = ""
				return entryData
			}()
		case 2:
			return func() models.MachinesRequestResponse {
				entryData.Data.Machines = []models.MachineDetails{}
				return entryData
			}()
		case 3:
			return func() models.MachinesRequestResponse {
				entryData.Data.Machines = nil
				return entryData
			}()

		default:
			return entryData
		}
	}

	testCases := []struct {
		name         string
		entryData    models.MachinesRequestResponse
		err          error
		expectedUUID string
	}{
		{name: "happy path",
			entryData:    getEntryData(0),
			err:          nil,
			expectedUUID: expectedMachineUUID,
		},
		{name: "machines slice without MachineUUID field",
			entryData:    getEntryData(1),
			err:          ErrGetMachineUUIDFromPostResponse,
			expectedUUID: "",
		},
		{name: "machines slice is empty",
			entryData:    getEntryData(2),
			err:          ErrGetMachineUUIDFromPostResponse,
			expectedUUID: "",
		},
		{name: "machines slice is nil",
			entryData:    getEntryData(3),
			err:          ErrGetMachineUUIDFromPostResponse,
			expectedUUID: "",
		},
	}

	for _, tc := range testCases {
		mockClient := httputils.NewMockCdiHTTPClient(t)
		fmc := &FabricManagerClient{cdiClient: mockClient}

		expectedPayload, _ := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecsArgs)
		expectedJSONPayload, _ := json.Marshal(expectedPayload)
		expectedQuery := map[string]string{"tenant_uuid": expectedPayload.Tenants.TenantUUID}
		expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
		var responseData models.MachinesRequestResponse

		helperSetResponseMachineUUID := func(payload []byte, endpoint string, queryParams map[string]string, response interface{}, headers map[string]string) {
			resp := response.(*models.MachinesRequestResponse)
			resp.Data = tc.entryData.Data
		}

		response := &responseData

		mockClient.EXPECT().
			Post(expectedJSONPayload, "/machines", expectedQuery, response, expectedHeaders).
			Run(helperSetResponseMachineUUID).
			Return(http.StatusOK, nil)

		t.Run(tc.name, func(t *testing.T) {
			machineUuid, err := fmc.CreateMachine(machineName, tenantId, machineSpecsArgs, models.AccessTokenExample)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.expectedUUID, machineUuid)
		})
	}
}

// CreateMachine - Error
func TestCreateMachine_PostError(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	machineName := "test-machine-001"
	tenantId := "b3b65e79-ad41-4367-89d6-e4e7315141ef"

	machineSpecsArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: `[{"column": "model","operator": "eq","value": "PRIMERGY-RX2540M6"}]`,
		DevicesSpecJson:       `[{"res_type":"storage","res_num":1,"tags":{"is_bootstorage":true},"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`,
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "75e6b24f-c1cc-4009-a871-b5828a468f4f",
		NetworkProvisionPort:  2,
		NetworkProvisionUUID:  "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NtpServer:             "ntp.example.com",
		DnsServer:             "8.8.8.8",
	}

	expectedPayload, _ := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecsArgs)

	expectedJSONPayload, _ := json.Marshal(expectedPayload)
	expectedQuery := map[string]string{"tenant_uuid": expectedPayload.Tenants.TenantUUID}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}

	var responseData, entryData models.MachinesRequestResponse

	json.Unmarshal([]byte(models.PostMachinesResponseExample), &entryData)

	response := &responseData

	errorData := `{"status": 500, "detail": {"code": "E020001", "message": "Exception occured, "data": {}}}`

	mockError := errors.New(errorData)
	mockClient.
		EXPECT().
		Post(expectedJSONPayload, "/machines", expectedQuery, response, expectedHeaders).
		Return(http.StatusInternalServerError, mockError)

	machineUuid, err := fmc.CreateMachine(machineName, tenantId, machineSpecsArgs, models.AccessTokenExample)

	assert.Error(t, err)
	assert.Equal(t, "", machineUuid)
	assert.Equal(t, mockError, err)
}

// Successful scenario
func TestPopulateCreateMachineRequest_Success(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	machineName := "test-machine-001"
	tenantId := "b3b65e79-ad41-4367-89d6-e4e7315141ef"

	machineSpecsArgs := models.MachineSpecsArgs{
		ComputeConditionsJson:     `[{"column": "model","operator": "eq","value": "PRIMERGY-RX2540M6"}]`,
		DevicesSpecJson:           `[{"res_type":"storage","res_num":1,	"tags": {"is_bootstorage":true},"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`,
		NetworkProvisionUUID:      "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NetworkProvisionPort:      1,
		NetworkProvisionDefaultGW: "192.168.0.1",
		NetworkBaremetalUUID:      "75e6b24f-c1cc-4009-a871-b5828a468f4f",
		NetworkBaremetalPort:      2,
		NetworkBaremetalDefaultGW: "172.0.0.1",
		NtpServer:                 "192.168.0.1",
		DnsServer:                 "8.8.8.8",
	}

	createMachineRequest, err := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecsArgs)

	assert.NoError(t, err, "populateCreateMachineRequest should not return an error")

	// Marshal the request to JSON
	rawJSON, err := json.Marshal(createMachineRequest)
	assert.NoError(t, err, "Marshaling JSON should not return an error")
	assert.Equal(t, models.CreateMachineRequestExpected, string(rawJSON))
}

func TestPopulateCreateMachineRequestOneProvisioningNetwork_Success(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	machineName := "test-machine-001"
	tenantId := "b3b65e79-ad41-4367-89d6-e4e7315141ef"

	machineSpecsArgs := models.MachineSpecsArgs{
		ComputeConditionsJson:     `[{"column": "model","operator": "eq","value": "PRIMERGY-RX2540M6"}]`,
		DevicesSpecJson:           `[{"res_type":"storage","res_num":1,	"tags": {"is_bootstorage":true},"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`,
		NetworkProvisionUUID:      "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NetworkProvisionPort:      2,
		NetworkProvisionDefaultGW: "192.168.0.1",
		NtpServer:                 "192.168.0.1",
		DnsServer:                 "8.8.8.8",
	}

	createMachineRequest, err := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecsArgs)

	assert.NoError(t, err, "populateCreateMachineRequest should not return an error")

	// Marshal the request to JSON
	rawJSON, err := json.Marshal(createMachineRequest)
	assert.NoError(t, err, "Marshaling JSON should not return an error")
	assert.Equal(t, models.CreateMachineRequestOneProvisioningNetworkExpected, string(rawJSON))
}

func TestCheckDeviceSpecJson(t *testing.T) {
	testCases := []struct {
		name       string
		deviceSpec string
		err        error
	}{
		{name: "happy path",
			deviceSpec: models.DeviceSpecsValid,
			err:        nil},

		{name: "field 'tags' does not contain field 'is_bootstorage'",
			deviceSpec: models.DeviceSpecsInvalidNoFieldTagsIsBootStorage,
			err:        ErrBootStorageTags},

		{name: "field 'tags' contains incorrect value for field 'is_bootstorage'",
			deviceSpec: models.DeviceSpecsInCorrectValueForTagsIsBootStorage,
			err:        ErrBootStorageTags},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDeviceSpecJson(tc.deviceSpec)
			// assert.ErrorIs(t, err, ErrBootStorageTags)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func getSpecArgs(tagsValue string) models.MachineSpecsArgs {
	deviceSpecJson := fmt.Sprintf(`[{"res_type":"storage","res_num":2, "res_spec":{"condition":	[{"column":"vendor","operator":"eq","value":"corsair"}]}},
	{"res_type":"storage","res_num":1,
		%s
		"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`, tagsValue)

	return models.MachineSpecsArgs{
		ComputeConditionsJson: `[{"column": "model","operator": "eq","value": "PRIMERGY-RX2540M6"}]`,
		DevicesSpecJson:       deviceSpecJson,
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "75e6b24f-c1cc-4009-a871-b5828a468f4f",
		NetworkProvisionPort:  2,
		NetworkProvisionUUID:  "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NtpServer:             "ntp.example.com",
		DnsServer:             "8.8.8.8",
	}

}

func getResourcesForTest() []models.Resource {
	var mrr models.MachinesRequestResponse
	err := json.Unmarshal([]byte(models.GetMachineResponseExampleWithTypoInStorageResSpec), &mrr)
	if err != nil {
		slog.Error("Error unmarshalling devices specification from JSON: ", "err", err)
		log.Fatalf("error while unmarshallig struct: %+v", err)
	}

	return mrr.Data.Machines[0].Resources
}

func getBootStorageConditionForTest() []models.Condition {
	return []models.Condition{
		{Column: "model", Operator: "eq", Value: "ssd"},
	}
}

func Test_getSsdId(t *testing.T) {

	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	testCases := []struct {
		name          string
		res           []models.Resource
		condition     []models.Condition
		expectedSsdId string
		expectedError error
	}{
		{name: "happy path",
			res:           getResourcesForTest(),
			condition:     getBootStorageConditionForTest(),
			expectedSsdId: "bbb32109-8765-4321-0fed-cba098765432",
			expectedError: nil,
		},

		{name: "add items to condition",
			condition: func(c []models.Condition) []models.Condition {
				additionalConditions := models.Condition{
					Column: "model", Operator: "eq", Value: "hdd",
				}
				c = append(c, additionalConditions)
				return c
			}(getBootStorageConditionForTest()),
			res:           getResourcesForTest(),
			expectedSsdId: "",
			expectedError: ErrSsdIdNotFound,
		},

		{name: "No field of type 'storage' was found",
			condition: getBootStorageConditionForTest(),
			res: func(res []models.Resource) []models.Resource {
				for i, r := range res {
					if r.ResourceType == "storage" {
						res[i].ResourceType = "not-a-storage"
					}
				}
				return res
			}(getResourcesForTest()),
			expectedSsdId: "",
			expectedError: ErrSsdIdNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmc.bootStorageCondition = tc.condition
			ssdId, err := fmc.getSsdId(tc.res)
			assert.Equal(t, tc.expectedSsdId, ssdId)

			if err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Invalid input string scenario
func TestPopulateCreateMachineRequest_Error(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	machineName := "test-machine-001"
	tenantId := "b3b65e79-ad41-4367-89d6-e4e7315141ef"

	machineSpecsArgs := models.MachineSpecsArgs{
		ComputeConditionsJson: "<invalid_string>",
		DevicesSpecJson:       `[{"res_type":"storage","res_num":1,"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]`,
		NetworkBaremetalPort:  1,
		NetworkBaremetalUUID:  "75e6b24f-c1cc-4009-a871-b5828a468f4f",
		NetworkProvisionPort:  2,
		NetworkProvisionUUID:  "5dc4769c-eef2-407f-b729-fec926ec9eda",
		NtpServer:             "ntp.example.com",
		DnsServer:             "8.8.8.8",
	}

	_, err := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecsArgs)

	assert.Error(t, err)
}

func TestNewFabricManagerClientError(t *testing.T) {
	_, err := NewFabricManagerClient("", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMissingParams)

	_, err = NewFabricManagerClient("", "/fabric_manager/api/v1/", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMissingParams)

	_, err = NewFabricManagerClient("https://192.168.122.1", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMissingParams)
}

func TestNewFabricManagerClientSuccess(t *testing.T) {
	fmc, err := NewFabricManagerClient("https://192.168.122.1", "/fabric_manager/api/v1/", models.DeviceSpecsValid)
	require.NoError(t, err)
	assert.NotNil(t, fmc.cdiClient)
}

func TestValidateTenantSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenant_id := "12345678-1234-1234-1234-123456789012"
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedQuery := map[string]string{"tenant_uuid": tenant_id}
	endpoint := fmt.Sprintf("/tenants/%s", tenant_id)
	mockClient.EXPECT().Get(endpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.ValidateTenant(tenant_id, models.AccessTokenExample)

	assert.NoError(t, err)
}

func TestValidateTenantIdFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenant_id := "12345678-1234-1234-1234-123456789012"

	expectedQuery := map[string]string{"tenant_uuid": tenant_id}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}

	mockError := errors.New("Request GET /tenants/cdi-test failed")
	endpoint := fmt.Sprintf("/tenants/%s", tenant_id)
	mockClient.EXPECT().Get(endpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusNotFound, mockError)

	err := fmc.ValidateTenant(tenant_id, models.AccessTokenExample)

	assert.Error(t, err)
	assert.EqualError(t, err, "Request GET /tenants/cdi-test failed")
}

func TestValidateTenantRequestFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenant_id := "12345678-1234-1234-1234-123456789012"

	expectedQuery := map[string]string{"tenant_uuid": tenant_id}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}

	mockError := errors.New("Request GET /tenants/cdi-test failed")
	endpoint := fmt.Sprintf("/tenants/%s", tenant_id)
	mockClient.EXPECT().Get(endpoint, expectedQuery, nil, expectedHeaders).Return(int(http.DefaultClient.Timeout), mockError)

	err := fmc.ValidateTenant(tenant_id, models.AccessTokenExample)

	assert.Error(t, err)
	assert.EqualError(t, err, "Request GET /tenants/cdi-test failed")
}

func TestPowerOnSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/pon", machineId)
	expectedPayload := []byte{}

	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.PowerOn(machineId, tenantId, models.AccessTokenExample)
	assert.NoError(t, err)
}

func TestPowerOnFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/pon", machineId)
	expectedPayload := []byte{}

	mockError := errors.New("Request failed")
	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusNotFound, mockError)

	err := fmc.PowerOn(machineId, tenantId, models.AccessTokenExample)
	assert.Error(t, err)
	assert.EqualError(t, err, "Request failed")
}

func TestPowerOffSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/poff", machineId)
	expectedPayload := []byte{}

	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.PowerOff(machineId, tenantId, models.AccessTokenExample)
	assert.NoError(t, err)
}

func TestPowerOffFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/poff", machineId)
	expectedPayload := []byte{}

	mockError := errors.New("Request failed")
	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusNotFound, mockError)

	err := fmc.PowerOff(machineId, tenantId, models.AccessTokenExample)
	assert.Error(t, err)
	assert.EqualError(t, err, "Request failed")
}

func TestGracefulShutdownSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": tenantId}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/graceful", machineId)
	expectedPayload := []byte{}

	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.GracefulShutdown(machineId, tenantId, models.AccessTokenExample)
	assert.NoError(t, err)
}

func TestGracefulShutdownFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": tenantId}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s/graceful", machineId)
	expectedPayload := []byte{}

	mockError := errors.New("Request failed")
	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusInternalServerError, mockError)

	err := fmc.GracefulShutdown(machineId, tenantId, models.AccessTokenExample)
	assert.Error(t, err)
	assert.EqualError(t, err, "Request failed")
}

func TestImageInstallSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	ssdId := "ssd_uuid_test"
	imageFilename := "image_filename"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := "/resources/ssd_uuid_test/imginstall"

	resource := models.BootResource{
		SSDResourceUUID: ssdId,
		BootImageName:   imageFilename,
	}
	payload := models.ImageInstallation{
		Resources: resource,
	}
	expectedPayload, _ := json.Marshal(payload)
	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.ImageInstall(tenantId, ssdId, imageFilename, models.AccessTokenExample)
	assert.NoError(t, err)
}

func TestImageInstallFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	ssdId := "ssd_uuid_test"
	imageFilename := "image_filename"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Content-Type": "application/json", "Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := "/resources/ssd_uuid_test/imginstall"

	resource := models.BootResource{
		SSDResourceUUID: ssdId,
		BootImageName:   imageFilename,
	}
	payload := models.ImageInstallation{
		Resources: resource,
	}
	expectedPayload, _ := json.Marshal(payload)
	mockError := errors.New("Request failed")
	mockClient.EXPECT().Put(expectedPayload, expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusNotFound, mockError)

	err := fmc.ImageInstall(tenantId, ssdId, imageFilename, models.AccessTokenExample)
	assert.Error(t, err)
	assert.EqualError(t, err, "Request failed")
}

func TestRemoveMachineSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s", machineId)

	mockClient.EXPECT().Delete(expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusOK, nil)

	err := fmc.RemoveMachine(machineId, tenantId, models.AccessTokenExample)
	assert.NoError(t, err)
}

func TestRemoveMachineFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	tenantId := "cdi-test"
	machineId := "cdd792f2-5591-4c18-a8bd-1c39e55dedfa"

	expectedQuery := map[string]string{"tenant_uuid": "cdi-test"}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s", machineId)

	mockError := errors.New("Request failed")
	mockClient.EXPECT().Delete(expectedEndpoint, expectedQuery, nil, expectedHeaders).Return(http.StatusNotFound, mockError)

	err := fmc.RemoveMachine(machineId, tenantId, models.AccessTokenExample)
	assert.Error(t, err)
	assert.EqualError(t, err, "Request failed")
}

func TestGetMachineDetailsSuccess(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}
	fmc.bootStorageCondition = getBootStorageConditionForTest()
	var responseData, entryData models.MachinesRequestResponse

	json.Unmarshal([]byte(models.GetMachineResponseExampleWithTypoInStorageResSpec), &entryData)

	tenantId := "cdi-test"
	machineUUID := "a1b2c3d4-e5f6-7890-1234-567890abcdef"

	expectedQuery := map[string]string{"tenant_uuid": tenantId}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s", machineUUID)

	helperSetMachineDetails := func(endpoint string, queryParams map[string]string, responseAddress interface{}, headers map[string]string) {
		resp := responseAddress.(*models.MachinesRequestResponse)
		resp.Data = entryData.Data
	}

	response := &responseData

	mockClient.
		EXPECT().
		Get(expectedEndpoint, expectedQuery, response, expectedHeaders).
		Run(helperSetMachineDetails).
		Return(http.StatusOK, nil)

	machineLanports, machineSsd, machineStatus, err := fmc.GetMachineDetails(tenantId, machineUUID, models.AccessTokenExample)
	assert.NoError(t, err)
	assert.Equal(t, models.ExpectedLanports, machineLanports)
	assert.Equal(t, "bbb32109-8765-4321-0fed-cba098765432", machineSsd)
	assert.Equal(t, 1, machineStatus)
}

func TestGetMachineDetailsSuccessDeleted(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}
	fmc.bootStorageCondition = getBootStorageConditionForTest()
	var responseData, entryData models.MachinesRequestResponse

	json.Unmarshal([]byte(models.GetMachineResponseExampleDeletedMachine), &entryData)

	tenantId := "cdi-test"
	machineUUID := "a1b2c3d4-e5f6-7890-1234-567890abcdef"

	expectedQuery := map[string]string{"tenant_uuid": tenantId}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s", machineUUID)

	helperSetMachineDetails := func(endpoint string, queryParams map[string]string, responseAddress interface{}, headers map[string]string) {
		resp := responseAddress.(*models.MachinesRequestResponse)
		resp.Data = entryData.Data
	}

	response := &responseData

	mockClient.
		EXPECT().
		Get(expectedEndpoint, expectedQuery, response, expectedHeaders).
		Run(helperSetMachineDetails).
		Return(http.StatusOK, nil)

	lanports, machineSsd, machineStatus, err := fmc.GetMachineDetails(tenantId, machineUUID, models.AccessTokenExample)

	assert.NoError(t, err)
	assert.Equal(t, []models.Lanport{}, lanports)
	assert.Equal(t, "", machineSsd)
	assert.Equal(t, 17, machineStatus)
}

func TestGetMachineDetailsFailed(t *testing.T) {
	mockClient := httputils.NewMockCdiHTTPClient(t)
	fmc := &FabricManagerClient{cdiClient: mockClient}

	var responseData models.MachinesRequestResponse

	tenantId := "cdi-test"
	machineUUID := "a1b2c3d4-e5f6-7890-1234-567890abcdef"

	expectedQuery := map[string]string{"tenant_uuid": tenantId}
	expectedHeaders := map[string]string{"Authorization": fmt.Sprintf("Bearer %s", models.AccessTokenExample)}
	expectedEndpoint := fmt.Sprintf("/machines/%s", machineUUID)

	response := &responseData

	mockError := fmt.Errorf("Request GET %s failed", expectedEndpoint)
	mockClient.
		EXPECT().
		Get(expectedEndpoint, expectedQuery, response, expectedHeaders).
		Return(http.StatusNotFound, mockError)

	machineLanports, machineSsd, machineStatus, err := fmc.GetMachineDetails(tenantId, machineUUID, models.AccessTokenExample)

	assert.Error(t, err)
	assert.Equal(t, []models.Lanport(nil), machineLanports)
	assert.Equal(t, "", machineSsd)
	assert.Equal(t, 0, machineStatus)
	assert.Equal(t, mockError, err)
}

func Test_getBootStorageCondition(t *testing.T) {

	testCases := []struct {
		name       string
		deviceSpec string
		err        error
		expected   []models.Condition
	}{
		{name: "happy path",
			deviceSpec: models.DeviceSpecsValid,
			err:        nil,
			expected: []models.Condition{
				{Column: "model", Operator: "eq", Value: "ssd"},
			}},

		{name: "lack of field 'storage'",
			deviceSpec: func(ds string) string {
				return strings.ReplaceAll(ds, "storage", "foo")
			}(models.DeviceSpecsValid),
			err:      ErrBootStorageConditionNotFound,
			expected: []models.Condition(nil)},

		{name: "lack of field 'tags'",
			deviceSpec: func(ds string) string {
				return strings.ReplaceAll(ds, "tags", "foo")
			}(models.DeviceSpecsValid),
			err:      ErrBootStorageConditionNotFound,
			expected: []models.Condition(nil)},

		{name: "lack of field 'is_bootstorage'",
			deviceSpec: func(ds string) string {
				return strings.ReplaceAll(ds, "is_bootstorage", "foo")
			}(models.DeviceSpecsValid),
			err:      ErrBootStorageConditionNotFound,
			expected: []models.Condition(nil)},

		{name: "field 'is_bootstorage' has incorrect value",
			deviceSpec: func(ds string) string {
				return strings.ReplaceAll(ds, `"is_bootstorage": true`, `"is_bootstorage": false`)
			}(models.DeviceSpecsValid),
			err:      ErrBootStorageConditionNotFound,
			expected: []models.Condition(nil)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bsc, err := getBootStorageCondition(tc.deviceSpec)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.expected, bsc)
		})
	}
}
