package fm

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"

	"github.com/fujitsu/docker-machine-driver-fsas/httputils"
	"github.com/fujitsu/docker-machine-driver-fsas/models"
)

const ErrMissingParams = "baseURI and port cannot be empty"

var (
	isInit                            = false
	ErrBootStorageTags                = errors.New("mandatory field 'is_bootstorage' must be equal true in devices specification")
	ErrBootStorageConditionNotFound   = errors.New("not found condition in device spec for bootable storage")
	ErrSsdIdNotFound                  = errors.New("ssdId not found in resources")
	ErrGetMachineUUIDFromPostResponse = errors.New("error while getting machine UUID from POST response")
)

// FabricManager interface defines the methods for interacting with the Fabric Manager.
type FabricManager interface {
	IsInit() bool
	ValidateTenant(tenantId, bearerToken string) error
	PowerOn(machineUUID, tenantId, bearerToken string) error
	PowerOff(machineUUID, tenantId, bearerToken string) error
	ImageInstall(tenantId string, ssdId string, imageFilename, bearerToken string) error
	RemoveMachine(machineUUID, tenantId, bearerToken string) error
	CreateMachine(machineName, tenantId string, machineSpecs models.MachineSpecsArgs, bearerToken string) (string, error)
	GetMachineDetails(tenantId, machineUUID, bearerToken string) ([]models.Lanport, string, int, error)
}

// FabricManagerClient struct holds configuration for Fabric Manager interaction.
type FabricManagerClient struct {
	cdiClient            httputils.CdiHTTPClient
	bootStorageCondition []models.Condition
}

// This makes FabricManagerClient implement the FabricManager interface
var _ FabricManager = (*FabricManagerClient)(nil)

// NewFabricManagerClient creates a new FabricManagerClient instance.
func NewFabricManagerClient(baseURI, endpoint, deviceSpecJsonString string) (*FabricManagerClient, error) {
	slog.Debug("Creating FabricManagerClient: ", "baseURI", baseURI, "endpoint", endpoint)
	if baseURI == "" || endpoint == "" {
		return nil, errors.New(ErrMissingParams)
	}
	serverURI := httputils.UrlBuilder(baseURI, endpoint)
	bootStorageCondition, err := getBootStorageCondition(deviceSpecJsonString)
	if err != nil {
		return nil, err
	}

	isInit = true
	return &FabricManagerClient{
		cdiClient:            httputils.NewStandardCdiHTTPClient(serverURI),
		bootStorageCondition: bootStorageCondition,
	}, nil
}

func (fmc *FabricManagerClient) IsInit() bool {
	return isInit
}

func (fmc *FabricManagerClient) ValidateTenant(tenantId, bearerToken string) error {
	endpoint := fmt.Sprintf("/tenants/%s", tenantId)

	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeader(bearerToken)
	_, err := fmc.cdiClient.Get(endpoint, queryParams, nil, headers)

	if err != nil {
		slog.Error("Tenant check failed because of an error: ", "endpoint", endpoint, "err", err)
		return err
	}

	slog.Info("Successfully validated tenant: ", "tenant_id", tenantId)
	return nil
}

func (fmc *FabricManagerClient) PowerOn(machineUUID, tenantId, bearerToken string) error {

	endpoint := fmt.Sprintf("/machines/%s/pon", machineUUID)
	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeaderWithContentType(bearerToken)
	payload := []byte{}

	statusCode, err := fmc.cdiClient.Put(payload, endpoint, queryParams, nil, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request PUT %s failed: ", endpoint), "err", err)
		return err
	}

	slog.Info("Successfully requested machine power on: ", "machine_uuid", machineUUID, "tenant_id", tenantId, "status_code", statusCode)
	return nil
}

func (fmc *FabricManagerClient) PowerOff(machineUUID, tenantId, bearerToken string) error {

	endpoint := fmt.Sprintf("/machines/%s/poff", machineUUID)
	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeaderWithContentType(bearerToken)
	payload := []byte{}

	statusCode, err := fmc.cdiClient.Put(payload, endpoint, queryParams, nil, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request PUT %s failed: ", endpoint), "err", err)
		return err
	}

	slog.Info("Successfully requested machine power off: ", "machine_uuid", machineUUID, "tenant_id", tenantId, "status_code", statusCode)
	return nil
}

func (fmc *FabricManagerClient) ImageInstall(tenantId, ssdId, imageFilename, bearerToken string) error {

	bootResource := models.BootResource{
		SSDResourceUUID: ssdId,
		BootImageName:   imageFilename,
	}

	imageInstallation := models.ImageInstallation{
		Resources: bootResource,
	}

	payload, err := json.Marshal(imageInstallation)
	if err != nil {
		slog.Error("Error marshalling PUT request payload to JSON: ", "err", err)
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	endpoint := fmt.Sprintf("/resources/%s/imginstall", ssdId)
	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeaderWithContentType(bearerToken)

	statusCode, err := fmc.cdiClient.Put(payload, endpoint, queryParams, nil, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request PUT %s failed", endpoint))
		return err
	}

	slog.Info("Successfully requested image installation: ", "tenant_id", tenantId, "ssd_id", ssdId, "status_code", statusCode)
	return nil
}

func (fmc *FabricManagerClient) RemoveMachine(machineUUID, tenantId, bearerToken string) error {

	endpoint := fmt.Sprintf("/machines/%s", machineUUID)
	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeader(bearerToken)

	statusCode, err := fmc.cdiClient.Delete(endpoint, queryParams, nil, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request DELETE %s failed: ", endpoint), "err", err)
		return err
	}

	slog.Info("Successfully removed machine: ", "machine_uuid", machineUUID, "tenant_id", tenantId, "status_code", statusCode)
	return nil
}

// getBootStorageCondition Returns conditions field from device spec
func getBootStorageCondition(devicesSpec string) ([]models.Condition, error) {
	var resources []models.Resource
	if err := json.Unmarshal([]byte(devicesSpec), &resources); err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.ResourceType == "storage" {
			if r.Tags != nil && r.Tags.IsBootStorage == true {
				return r.ResourceSpec.Condition, nil
			}
		}
	}

	return nil, ErrBootStorageConditionNotFound
}

// populateCreateMachineRequest constructs a CreateMachineRequest from machineName, tenantId and parameters from models.MachineSpecsArgs
func (fmc *FabricManagerClient) populateCreateMachineRequest(machineName, tenantId string, machineSpecs models.MachineSpecsArgs) (*models.CreateMachineRequest, error) {

	var devicesSpec []models.Resource
	var computeConditions []models.Condition

	err := json.Unmarshal([]byte(machineSpecs.DevicesSpecJson), &devicesSpec)
	if err != nil {
		slog.Error("Error unmarshalling devices specification from JSON: ", "err", err, "machineSpecs.DevicesSpecJson", machineSpecs.DevicesSpecJson)
		return nil, err
	}

	err = json.Unmarshal([]byte(machineSpecs.ComputeConditionsJson), &computeConditions)
	if err != nil {
		slog.Error("Error unmarshalling compute conditions from JSON: ", "err", err)
		return nil, err
	}

	subnets := []models.Subnet{
		{
			SubnetUUID: machineSpecs.NetworkProvisionUUID,
			LanportIdx: machineSpecs.NetworkProvisionPort,
			Ntp:        machineSpecs.NtpServer,
			Dns:        machineSpecs.DnsServer,
			DefaultGW:  machineSpecs.NetworkProvisionDefaultGW,
		},
	}

	if machineSpecs.NetworkBaremetalUUID != "" {
		subnets = append(subnets, models.Subnet{
			SubnetUUID: machineSpecs.NetworkBaremetalUUID,
			LanportIdx: machineSpecs.NetworkBaremetalPort,
			Ntp:        machineSpecs.NtpServer,
			Dns:        machineSpecs.DnsServer,
			DefaultGW:  machineSpecs.NetworkBaremetalDefaultGW,
		})
	}

	resourceSpecification := []models.Resource{
		{
			ResourceType: "compute",
			ResourceNum:  1,
			ResourceSpec: &models.ResSpec{Condition: computeConditions},
			Network: &models.Network{
				NicType: 1,
				Subnets: subnets,
			},
		},
	}

	resourceSpecification = append(resourceSpecification, devicesSpec...)

	// TODO: temporary change for physical CDI tests; refactor me!
	machineName = strings.ReplaceAll(machineName, "-", "_")

	machine := models.CreateMachineSpec{
		Machine: machineName,
		Resources: []models.ResSpecs{
			{
				ResourceSpecifications: resourceSpecification,
			},
		},
	}
	machines := []models.CreateMachineSpec{machine}

	tenants := models.CreateMachineTenantsRequest{
		TenantUUID: tenantId,
		Machines:   machines,
	}

	return &models.CreateMachineRequest{Tenants: tenants}, nil
}

// CreateMachine sends a POST request to the Fabric Manager's `/machines/` endpoint to create a new machine
func (fmc *FabricManagerClient) CreateMachine(machineName, tenantId string, machineSpecs models.MachineSpecsArgs, bearerToken string) (string, error) {

	createMachineRequest, err := fmc.populateCreateMachineRequest(machineName, tenantId, machineSpecs)
	if err != nil {
		slog.Error("Error while populating machine specs in populateCreateMachineRequest: ", "err", err)
		return "", fmt.Errorf("failed to populate createMachineRequest: %w", err)
	}

	payload, err := json.Marshal(createMachineRequest)
	if err != nil {
		slog.Error("Error marshalling POST /machines/ request payload to JSON: ", "err", err)
		return "", fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	slog.Debug("Request Payload: ", "payload", string(payload))

	var response models.MachinesRequestResponse

	queryParams := map[string]string{"tenant_uuid": createMachineRequest.Tenants.TenantUUID}
	headers := httputils.GetAuthorizationHeaderWithContentType(bearerToken)

	_, err = fmc.cdiClient.Post(payload, "/machines", queryParams, &response, headers)
	if err != nil {
		slog.Error("Request POST /machines failed")
		return "", err
	}

	if !(len(response.Data.Machines) > 0 && response.Data.Machines[0].MachineUUID != "") {
		slog.Error("Error while getting machine UUID from POST response: ", "response", response)
		return "", ErrGetMachineUUIDFromPostResponse
	}
	machineUuid := response.Data.Machines[0].MachineUUID

	slog.Info("New machine successfully created: ", "machineUuid", machineUuid)
	return machineUuid, nil
}

// GetMachineDetails receives status on Machine from the Fabric Manager service.
func (fmc *FabricManagerClient) GetMachineDetails(tenantId, machineUUID, bearerToken string) (lanports []models.Lanport, bootSsd string, status int, _ error) {
	endpoint := fmt.Sprintf("/machines/%s", machineUUID)
	slog.Debug("Getting status on Machine: ", "mach_uuid", machineUUID)

	var responseData models.MachinesRequestResponse

	queryParams := map[string]string{"tenant_uuid": tenantId}
	headers := httputils.GetAuthorizationHeader(bearerToken)

	if _, err := fmc.cdiClient.Get(endpoint, queryParams, &responseData, headers); err != nil {
		slog.Error(fmt.Sprintf("Request GET %s failed: ", endpoint), "err", err)
		return lanports, bootSsd, status, err
	}

	lanports = responseData.Data.Machines[0].Lanports
	bootSsd, err := fmc.getSsdId(responseData.Data.Machines[0].Resources)
	if err != nil {
		return lanports, bootSsd, status, err
	}
	status = responseData.Data.Machines[0].MachineStatus

	slog.Info("Successfully received status on Machine: ",
		"mach_uuid", machineUUID,
		"lanports", lanports,
		"boot_ssd", bootSsd,
		"mach_status", status)

	return lanports, bootSsd, status, nil
}

// getSsdId Returns ssd id as UUID string and error
func (fmc *FabricManagerClient) getSsdId(resource []models.Resource) (string, error) {
	// Do not return error in case resource slice is empty (response after machine is deleted)
	if len(resource) == 0 {
		slog.Info("Not possible to read ssdId because of empty resource list")
		return "", nil
	}
	for _, r := range resource {
		if r.ResourceType == "storage" {
			if slices.Equal(r.ResourceSpec.Condition, fmc.bootStorageCondition) {
				ssdId := r.ResourceUUID
				slog.Info("Successfully found ssdId: ", "ssdId", ssdId)
				return ssdId, nil
			}
		}
	}
	slog.Error(ErrSsdIdNotFound.Error()+";", "conditions", fmc.bootStorageCondition)
	return "", ErrSsdIdNotFound
}

func CheckDeviceSpecJson(ds string) error {

	var devicesSpec []models.Resource

	err := json.Unmarshal([]byte(ds), &devicesSpec)
	if err != nil {
		slog.Error("Error unmarshalling devices specification from JSON: ", "err", err, "deviceSpecJson", ds)
		return err
	}

	var flagIsBootStorageFound = false
	for _, ds := range devicesSpec {
		if ds.ResourceType == "storage" {
			if ds.Tags != nil && ds.Tags.IsBootStorage == true {
				flagIsBootStorageFound = true
				break
			}
		}
	}
	if !flagIsBootStorageFound {
		slog.Error("Mandatory field 'is_bootstorage' must be equal true in devices specification for at least one resource of type 'storage': ", "DevicesSpecJson", ds)
		return ErrBootStorageTags
	}

	return nil
}
