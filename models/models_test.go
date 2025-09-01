package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateMachineRequestToJSON(t *testing.T) {
	request := CreateMachineRequest{
		Tenants: CreateMachineTenantsRequest{
			TenantUUID: "b3b65e79-ad41-4367-89d6-e4e7315141ef",
			Machines: []CreateMachineSpec{
				{
					Machine: "test-machine-001",
					Resources: []ResSpecs{
						{
							ResourceSpecifications: []Resource{
								{
									ResourceType: "compute",
									ResourceNum:  1,
									ResourceSpec: &ResSpec{
										Condition: []Condition{
											{Column: "model", Operator: "eq", Value: "PRIMERGY-RX2540M6"},
										},
									},
								},
								{
									ResourceType: "storage",
									ResourceNum:  2, // Added a second storage resource
									ResourceSpec: &ResSpec{
										Condition: []Condition{
											{Column: "type", Operator: "eq", Value: "NVMe"},
										},
									},
								},
								{
									ResourceType: "network",
									ResourceNum:  1,
									ResourceSpec: &ResSpec{
										Condition: []Condition{
											{Column: "name", Operator: "eq", Value: "baremetal-mgmt"},
										},
									},
									Network: &Network{
										NicType: 1,
										Subnets: []Subnet{
											{
												SubnetUUID: "75e6b24f-c1cc-4009-a871-b5828a468f4f",
												LanportIdx: 1,
												DefaultGW:  "192.168.1.1",     // Realistic gateway
												LeaseTime:  "86400s",          // Lease time in seconds
												Ntp:        "ntp.example.com", // NTP server
												Dns:        "8.8.8.8",         // DNS server
											},
										},
									},
								},
								{
									ResourceType: "network",
									ResourceNum:  2,
									ResourceSpec: &ResSpec{
										Condition: []Condition{
											{Column: "name", Operator: "eq", Value: "provisioning-net"},
										},
									},
									Network: &Network{
										NicType: 2, // Different NIC type
										Subnets: []Subnet{
											{
												SubnetUUID: "5dc4769c-eef2-407f-b729-fec926ec9eda",
												LanportIdx: 2,
												DefaultGW:  "10.0.0.1",                                // Different gateway
												LeaseTime:  fmt.Sprintf("%ds", int(time.Hour*24*7*2)), // Two weeks lease time
												Ntp:        "time.google.com",                         // Different NTP server
												Dns:        "",                                        // Different DNS server
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal the request to JSON
	rawJSON, err := json.Marshal(request)
	assert.NoError(t, err, "Marshaling JSON should not return an error")
	assert.Equal(t, PostMachinesRequestExpected, string(rawJSON))
}

func TestUnmarshalPostMachinesResponse(t *testing.T) {
	var postResponse MachinesRequestResponse
	err := json.Unmarshal([]byte(PostMachinesResponseExample), &postResponse)
	assert.NoError(t, err)

	machine := postResponse.Data.Machines[0]

	assert.Equal(t, "", machine.FabricUUID) // should be default nil value
	assert.Equal(t, 0, machine.FabricID)    // should be default nil value
	assert.Equal(t, "59756ed2-6a42-47f2-bc54-117bcf6bdce3", machine.MachineUUID)
	assert.Equal(t, 1, machine.MachineID)
	assert.Equal(t, "machine-01", machine.MachineName)
	assert.Equal(t, "domzalskis", machine.MachineOwner)

	assert.Equal(t, 4, len(machine.Resources))

	// Compute resource assertions
	computeResource := machine.Resources[0]
	assert.Equal(t, "c1a4e32f-ea8f-4eff-8c8c-55d473deb1a0", computeResource.ResourceUUID)
	assert.Equal(t, "cpu-01", computeResource.ResourceName)
	assert.Equal(t, "compute", computeResource.ResourceType)
	assert.Equal(t, 1, computeResource.ResourceStatus)
	assert.Equal(t, "1", computeResource.ResourceOpStatus)
	assert.Equal(t, "PRIMERGRYRX2540M4", computeResource.ResourceSpec.Condition[0].Value)

	// Storage resource assertions
	storageResource := machine.Resources[1]
	assert.Equal(t, "29e3d171-2441-4df9-9cc3-44c928daf41e", storageResource.ResourceUUID)
	assert.Equal(t, "storage-01", storageResource.ResourceName)
	assert.Equal(t, "storage", storageResource.ResourceType)
	assert.Equal(t, "16TB", storageResource.ResourceSpec.Condition[0].Value)

	// Network resource (provisioning) assertions
	networkProvisioningResource := machine.Resources[2]
	assert.Equal(t, "6b5f0567-921f-4ef3-a6e5-11f7e2609857", networkProvisioningResource.ResourceUUID)
	assert.Equal(t, "network-provisioning", networkProvisioningResource.ResourceName)
	assert.Equal(t, "network", networkProvisioningResource.ResourceType)
	assert.Equal(t, "provisioning", networkProvisioningResource.ResourceSpec.Condition[0].Value)
	assert.NotNil(t, networkProvisioningResource.Network)
	assert.Equal(t, 1, networkProvisioningResource.Network.NicType)
	assert.Equal(t, 1, len(networkProvisioningResource.Network.Subnets))
	assert.Equal(t, "6b5f0567-921f-4ef3-a6e5-11f7e2609857", networkProvisioningResource.Network.Subnets[0].SubnetUUID)
	assert.Equal(t, 1, networkProvisioningResource.Network.Subnets[0].LanportIdx)
	assert.Equal(t, "gateway-address", networkProvisioningResource.Network.Subnets[0].DefaultGW)

	// Network resource (cluster) assertions
	networkClusterResource := machine.Resources[3]
	assert.Equal(t, "991fbfd5-3521-4098-880e-1d9d2c8d2705", networkClusterResource.ResourceUUID)
	assert.Equal(t, "network-cluster", networkClusterResource.ResourceName)
	assert.Equal(t, "network", networkClusterResource.ResourceType)
	assert.Equal(t, "cluster", networkClusterResource.ResourceSpec.Condition[0].Value)
	assert.NotNil(t, networkClusterResource.Network)
	assert.Equal(t, 2, networkClusterResource.Network.NicType)
	assert.Equal(t, 1, len(networkClusterResource.Network.Subnets))
	assert.Equal(t, "991fbfd5-3521-4098-880e-1d9d2c8d2705", networkClusterResource.Network.Subnets[0].SubnetUUID)
	assert.Equal(t, 2, networkClusterResource.Network.Subnets[0].LanportIdx)
	assert.Equal(t, "gateway-address", networkClusterResource.Network.Subnets[0].DefaultGW)
}

func TestUnmarshalGetMachineResponse(t *testing.T) {
	var getResponse MachinesRequestResponse
	err := json.Unmarshal([]byte(GetMachineResponseExample), &getResponse)
	assert.NoError(t, err)

	machine := getResponse.Data.Machines[0]
	// Assertions for GET response
	assert.Equal(t, "58f4c0f8-6c74-4e86-a560-95ed13daaa46", machine.FabricUUID)
	assert.Equal(t, 1, machine.FabricID)
	assert.Equal(t, "a1b2c3d4-e5f6-7890-1234-567890abcdef", machine.MachineUUID)
	assert.Equal(t, 12345, machine.MachineIDNonLiqid)
	assert.Equal(t, 67890, machine.MachineID)
	assert.Equal(t, "example-machine-01", machine.MachineName)
	assert.Equal(t, 1, machine.MachineStatus)
	assert.Equal(t, "00", machine.MachineOpStatus)
	assert.Equal(t, "Running", machine.MachineStatusDetail)
	assert.Equal(t, "user123", machine.MachineOwner)
	assert.Equal(t, "f0e9d8c7-b6a5-4321-0987-6543210fedcb", machine.GroupUUID)
	assert.Equal(t, "e9d8c7b6-a543-2109-8765-43210fedcba0", machine.BootSSD)

	// Lanports assertions
	assert.Equal(t, 5, len(machine.Lanports))
	lanport1 := machine.Lanports[0]
	assert.Equal(t, "d8c7b6a5-4321-0987-6543-210fedcba098", lanport1.LanportUUID)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", lanport1.SubnetUUID)
	assert.Equal(t, "00:11:22:33:44:55", lanport1.MacAddress)
	assert.Equal(t, 1, lanport1.LanportIdx)
	assert.Equal(t, "192.168.2.100", lanport1.IPAddress)

	lanport2 := machine.Lanports[1] // Assertions for the second lanport
	assert.Equal(t, "01085c2c-15c4-4957-9ad3-7d1ee481f082", lanport2.LanportUUID)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", lanport2.SubnetUUID)
	assert.Equal(t, "00:11:22:33:44:66", lanport2.MacAddress)
	assert.Equal(t, 2, lanport2.LanportIdx)
	assert.Equal(t, "192.168.2.150", lanport2.IPAddress)

	// Resources assertions
	assert.Equal(t, 6, len(machine.Resources))

	// Example assertions for the first resource (compute)
	resourceCPU := machine.Resources[0]
	assert.Equal(t, "b6a54321-0987-6543-210f-edcba0987654", resourceCPU.ResourceUUID)
	assert.Equal(t, "compute-resource-1", resourceCPU.ResourceName)
	assert.Equal(t, "compute", resourceCPU.ResourceType)
	assert.Equal(t, 1, resourceCPU.ResourceStatus)
	assert.Equal(t, "0", resourceCPU.ResourceOpStatus)
	assert.Equal(t, "cpu_cores", resourceCPU.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "eq", resourceCPU.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "4", resourceCPU.ResourceSpec.Condition[0].Value)

	resourceStorage1 := machine.Resources[1]
	assert.Equal(t, "a5432109-8765-4321-0fed-cba098765432", resourceStorage1.ResourceUUID)
	assert.Equal(t, "storage-resource-1", resourceStorage1.ResourceName)
	assert.Equal(t, "storage", resourceStorage1.ResourceType)
	assert.Equal(t, 1, resourceStorage1.ResourceStatus)
	assert.Equal(t, "0", resourceStorage1.ResourceOpStatus)
	assert.Equal(t, "storage_size", resourceStorage1.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "gt", resourceStorage1.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "100", resourceStorage1.ResourceSpec.Condition[0].Value)

	resourceStorage2 := machine.Resources[2]
	assert.Equal(t, "bbb32109-8765-4321-0fed-cba098765432", resourceStorage2.ResourceUUID)
	assert.Equal(t, "storage-resource-2", resourceStorage2.ResourceName)
	assert.Equal(t, "storage", resourceStorage2.ResourceType)
	assert.Equal(t, 1, resourceStorage2.ResourceStatus)
	assert.Equal(t, "0", resourceStorage2.ResourceOpStatus)
	assert.Equal(t, true, resourceStorage2.Tags.IsBootStorage)
	assert.Equal(t, "model", resourceStorage2.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "eq", resourceStorage2.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "ssd", resourceStorage2.ResourceSpec.Condition[0].Value)

	resourceGPU := machine.Resources[3]
	assert.Equal(t, "43210987-6543-210f-edcb-a09876543210", resourceGPU.ResourceUUID)
	assert.Equal(t, "gpu-resource-1", resourceGPU.ResourceName)
	assert.Equal(t, "gpu", resourceGPU.ResourceType)
	assert.Equal(t, 1, resourceGPU.ResourceStatus)
	assert.Equal(t, "0", resourceGPU.ResourceOpStatus)
	assert.Equal(t, "gpu_model", resourceGPU.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "eq", resourceGPU.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "NVIDIA Tesla T4", resourceGPU.ResourceSpec.Condition[0].Value)

	resourceNetworkProvisioning := machine.Resources[4]
	assert.Equal(t, "21098765-4321-0fed-cba0-987654321098", resourceNetworkProvisioning.ResourceUUID)
	assert.Equal(t, "network-resource-1", resourceNetworkProvisioning.ResourceName)
	assert.Equal(t, "network", resourceNetworkProvisioning.ResourceType)
	assert.Equal(t, 1, resourceNetworkProvisioning.ResourceStatus)
	assert.Equal(t, "0", resourceNetworkProvisioning.ResourceOpStatus)
	assert.Equal(t, "name", resourceNetworkProvisioning.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "eq", resourceNetworkProvisioning.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "provisioning", resourceNetworkProvisioning.ResourceSpec.Condition[0].Value)
	assert.NotNil(t, resourceNetworkProvisioning.Network)
	assert.Equal(t, 1, resourceNetworkProvisioning.Network.NicType)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", resourceNetworkProvisioning.Network.Subnets[0].SubnetUUID)
	assert.Equal(t, 1, resourceNetworkProvisioning.Network.Subnets[0].LanportIdx)
	assert.Equal(t, "192.168.2.1", resourceNetworkProvisioning.Network.Subnets[0].DefaultGW)
	assert.Equal(t, "86400", resourceNetworkProvisioning.Network.Subnets[0].LeaseTime)
	assert.Equal(t, "192.168.2.2", resourceNetworkProvisioning.Network.Subnets[0].Ntp)
	assert.Equal(t, "8.8.8.8", resourceNetworkProvisioning.Network.Subnets[0].Dns)
	assert.Equal(t, "host1.example.com", resourceNetworkProvisioning.Network.Subnets[0].Fqdn)

	resourceNetworkCluster := machine.Resources[5]
	assert.Equal(t, "09876543-210f-edcb-a098-765432109876", resourceNetworkCluster.ResourceUUID)
	assert.Equal(t, "network-resource-2", resourceNetworkCluster.ResourceName)
	assert.Equal(t, "network", resourceNetworkCluster.ResourceType)
	assert.Equal(t, 1, resourceNetworkCluster.ResourceStatus)
	assert.Equal(t, "0", resourceNetworkCluster.ResourceOpStatus)
	assert.Equal(t, "name", resourceNetworkCluster.ResourceSpec.Condition[0].Column)
	assert.Equal(t, "eq", resourceNetworkCluster.ResourceSpec.Condition[0].Operator)
	assert.Equal(t, "cluster", resourceNetworkCluster.ResourceSpec.Condition[0].Value)
	assert.NotNil(t, resourceNetworkCluster.Network)
	assert.Equal(t, 1, resourceNetworkCluster.Network.NicType)
	assert.Equal(t, "78901234-5678-9abc-def0-1234567890ab", resourceNetworkCluster.Network.Subnets[0].SubnetUUID)
	assert.Equal(t, "78901234-5678-9abc-def0-1234567890ab", resourceNetworkCluster.Network.Subnets[0].SubnetUUID)
	assert.Equal(t, 2, resourceNetworkCluster.Network.Subnets[0].LanportIdx)
	assert.Equal(t, "10.0.0.1", resourceNetworkCluster.Network.Subnets[0].DefaultGW)
	assert.Equal(t, "3600", resourceNetworkCluster.Network.Subnets[0].LeaseTime)
	assert.Equal(t, "10.0.0.2", resourceNetworkCluster.Network.Subnets[0].Ntp)
	assert.Equal(t, "1.1.1.1", resourceNetworkCluster.Network.Subnets[0].Dns)
	assert.Equal(t, "host2.example.com", resourceNetworkCluster.Network.Subnets[0].Fqdn)
}

func TestUnmarshalDeleteMachineResponse(t *testing.T) {
	var deleteResponse MachinesRequestResponse
	err := json.Unmarshal([]byte(DeleteMachineResponseExample), &deleteResponse)
	assert.NoError(t, err)

	// Assertions for DELETE response
	assert.Equal(t, "c7b6a543-2109-8765-4321-0fedcba09876", deleteResponse.Data.Machines[0].MachineUUID)
}

func TestUnmarshalImageInstallJSON(t *testing.T) {
	var imageInstallResponse ImageInstallation
	err := json.Unmarshal([]byte(ImageInstallPutPayloadExample), &imageInstallResponse)
	assert.NoError(t, err)

	// Assertions for image installation response
	assert.Equal(t, "c7b6a543-2109-8765-4321-0fedcba09876", imageInstallResponse.Resources.SSDResourceUUID)
	assert.Equal(t, "boot-image-linux-01", imageInstallResponse.Resources.BootImageName)
}

func TestMarshalImageInstallToJSON(t *testing.T) {
	request := ImageInstallation{
		Resources: BootResource{
			SSDResourceUUID: "c7b6a543-2109-8765-4321-0fedcba09876",
			BootImageName:   "my-boot-image-02",
		},
	}

	rawJSON, err := json.Marshal(request)
	assert.NoError(t, err)

	assert.Equal(t, ImageInstallPutResponseExample, string(rawJSON))
}

func Test_customUnmarshallerForResourceStruct(t *testing.T) {
	testCases := []struct {
		name       string
		jsonString string
		expected   *ResSpec
	}{
		{name: "valid response without typo",
			jsonString: `{
					"res_uuid":"b6a54321-0987-6543-210f-edcba0987654",
					"res_name":"compute-resource-1",
					"res_type":"compute",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"cpu_cores",
						"operator":"eq",
						"value":"4"
						}
					]
					}
				}`,
			expected: &ResSpec{[]Condition{{Column: "cpu_cores", Operator: "eq", Value: "4"}}}},

		{name: "not valid response with typo",
			jsonString: `{
					"res_uuid":"b6a54321-0987-6543-210f-edcba0987654",
					"res_name":"compute-resource-1",
					"res_type":"compute",
					"res_status":1,
					"res_op_status":"0",
					"res_spcec":{
					"condition":[
						{
						"column":"cpu_cores",
						"operator":"eq",
						"value":"4"
						}
					]
					}
				}`,
			expected: &ResSpec{[]Condition{{Column: "cpu_cores", Operator: "eq", Value: "4"}}}},

		{name: "response without res_spec field at all, neither valid nor invalid",
			jsonString: `{
					"res_uuid":"b6a54321-0987-6543-210f-edcba0987654",
					"res_name":"compute-resource-1",
					"res_type":"compute",
					"res_status":1,
					"res_op_status":"0"
				}`,
			expected: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resource Resource

			err := json.Unmarshal([]byte(tc.jsonString), &resource)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, resource.ResourceSpec)
		})
	}

}
