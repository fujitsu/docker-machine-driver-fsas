package models

import (
	"encoding/json"
)

// VMRequestPayload struct represents the payload for requesting a new VM.
type VMRequestPayload struct {
	MachineName string `json:"machine_name"`
	MachineType string `json:"machine_type"`
}

// Structures necessary to generate payload of POST /machines requests to fabric manager service
type Condition struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type ResSpec struct {
	// TODO: confirm if list below is 1-element only when physical fm environment is available
	Condition []Condition `json:"condition"` // List containing only 1 element by design of fabric manager
}

type Subnet struct {
	SubnetUUID string `json:"subnet_uuid"`
	LanportIdx int    `json:"lanport_idx"`
	DefaultGW  string `json:"default_gw,omitempty"`
	LeaseTime  string `json:"lease_time,omitempty"`
	Ntp        string `json:"ntp"`
	Dns        string `json:"dns,omitempty"`
	Fqdn       string `json:"fqdn,omitempty"`
}

type Network struct {
	NicType int      `json:"nic_type"`
	Subnets []Subnet `json:"subnets"` // Expected 1-element arrays only
}

type ResStorageTags struct {
	IsBootStorage bool `json:"is_bootstorage"`
}

type Resource struct {
	ResourceType     string          `json:"res_type"`
	ResourceNum      int             `json:"res_num,omitempty"` // Present only in payload of POST /machines
	ResourceSpec     *ResSpec        `json:"res_spec,omitempty"`
	Tags             *ResStorageTags `json:"tags,omitempty"`
	Network          *Network        `json:"network,omitempty"`            // It's the only localization where network data are processed
	ResourceUUID     string          `json:"res_uuid,omitempty"`           // Present only in responses
	ResourceName     string          `json:"res_name,omitempty"`           // Present only in responses
	ResourceStatus   int             `json:"res_status,omitempty"`         // Present only in responses
	ResourceOpStatus string          `json:"res_op_status,omitempty"`      // Present only in responses
	MinResourceCount int             `json:"min_resource_count,omitempty"` // GPU field (read-only and optional)
	MaxResourceCount int             `json:"max_resource_count,omitempty"` // GPU field (read-only and optional)
}

// Custom Unmarshaler to handle both "res_spec" and "res_spcec"
/*
	The reason why there is a defined custom unmarshaller is a typo
	in the production Fabric Manager code: response for the Get request
	returns JSON with field: 'res_spcec' instead of 'res_spec'.
	This is only a temporary workaround until the typo is corrected
	and merged to the release branch.
*/
// TODO: Remove this custom unmarshaller when the typo is fixed
func (r *Resource) UnmarshalJSON(data []byte) error {
	// Define a temporary type to avoid recursion during unmarshaling.
	type Alias Resource
	aux := &struct {
		*Alias
		ResSpcec *ResSpec `json:"res_spcec,omitempty"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// If "res_spcec" was present, move its value to ResourceSpec.
	if aux.ResSpcec != nil {
		r.ResourceSpec = aux.ResSpcec
	}

	return nil
}

// Custom Marshaler to omit "tags", "minresourcecount" and "maxresourcecount"
/*
	The reason why a custom marshaller is defined is because some fields
   	are only needed internally - they cannot be serialized in the requests to FM API,
    but are needed to be deserialized from customer's input in GUI
   Specifically:
   - tags: used by the node driver to determine which disk to install the OS on
   - min_resource_count / max_resource_count: GPU fields
     that can be present in customer input JSON, but must never be sent
     to Fabric Manager as they are not part of the API specification.
*/
func (r *Resource) MarshalJSON() ([]byte, error) {
	// Create a temporary struct without the Tags, MinResourceCount and MaxResourceCount fields
	type OutgoingResource Resource
	alias := &OutgoingResource {
		ResourceType:     r.ResourceType,
		ResourceNum:      r.ResourceNum,
		ResourceSpec:     r.ResourceSpec,
		Network:          r.Network,
		ResourceUUID:     r.ResourceUUID,
		ResourceName:     r.ResourceName,
		ResourceStatus:   r.ResourceStatus,
		ResourceOpStatus: r.ResourceOpStatus,
	}

	data, err := json.Marshal(alias)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type ResSpecs struct {
	ResourceSpecifications []Resource `json:"res_specs"` // Many resources allowed & expected
}

type CreateMachineSpec struct {
	Machine   string     `json:"mach_name"`
	Resources []ResSpecs `json:"resources"` // List containing only 1 element by design of fabric manager
}

type CreateMachineRequest struct {
	Tenants CreateMachineTenantsRequest `json:"tenants"`
}

type CreateMachineTenantsRequest struct {
	TenantUUID string              `json:"tenant_uuid"`
	Machines   []CreateMachineSpec `json:"machines"` // Many machines allowed but for node driver it is always one
}

// Structures necessary to deserialize response from POST /machines & GET /machines/<uuid> requests
type Lanport struct {
	LanportUUID string `json:"lanport_uuid"`
	SubnetUUID  string `json:"subnet_uuid"`
	MacAddress  string `json:"mac_address"`
	LanportIdx  int    `json:"lanport_idx"`
	IPAddress   string `json:"ip_address"`
}

type MachineDetails struct {
	FabricUUID          string     `json:"fabric_uuid,omitempty"`
	FabricID            int        `json:"fabric_id,omitempty"`
	MachineUUID         string     `json:"mach_uuid"`
	MachineIDNonLiqid   int        `json:"mach_id_nonliqid,omitempty"`
	MachineID           int        `json:"mach_id,omitempty"`
	MachineName         string     `json:"mach_name,omitempty"`
	MachineStatus       int        `json:"mach_status,omitempty"`
	MachineOpStatus     string     `json:"mach_op_status,omitempty"`
	MachineStatusDetail string     `json:"mach_status_detail,omitempty"`
	MachineOwner        string     `json:"mach_owner,omitempty"`
	GroupUUID           string     `json:"grp_uuid,omitempty"`
	BootSSD             string     `json:"boot_ssd,omitempty"`
	Lanports            []Lanport  `json:"lanports,omitempty"`
	Resources           []Resource `json:"resources,omitempty"`
}

type MachinesResponseData struct {
	Machines []MachineDetails `json:"machines"`
}

type MachinesRequestResponse struct {
	Data MachinesResponseData `json:"data"`
}

// Structures necesary to handle OS image installation
type BootResource struct {
	SSDResourceUUID string `json:"res_uuid_ssd"`
	BootImageName   string `json:"bootimg_filename"`
}

type ImageInstallation struct {
	Resources BootResource `json:"resources"`
}

// MachineSpecsArgs struct holds part of the parameters for populateCreateMachineRequest method
type MachineSpecsArgs struct {
	ComputeConditionsJson     string
	DevicesSpecJson           string
	NetworkBaremetalPort      int
	NetworkProvisionPort      int
	NetworkBaremetalUUID      string
	NetworkProvisionUUID      string
	NetworkBaremetalDefaultGW string
	NetworkProvisionDefaultGW string
	NtpServer                 string
	DnsServer                 string
}
