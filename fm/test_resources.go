package fm

import "github.com/fujitsu/docker-machine-driver-fsas/models"

var (
	deviceSpecStorageAndComposableNetwork = `[
        {
            "res_type": "storage",
            "res_num": 1,
            "tags": {"is_bootstorage": true},
            "res_spec": {
                "condition": [
                    {
                        "column": "vendor",
                        "operator": "eq",
                        "value": "Samsung"
                    }
                ]
            }
        },
        {
            "res_type": "network",
            "res_spec": {
                        "condition": [
                            {
                                "column": "vendor",
                                "operator": "eq",
                                "value": "Mellanox"
                            }
                        ]
            },
            "network": {
                "nic_type": 2,
                "subnets": [
                    {
                        "subnet_uuid": "75e6b24f-baee-baee-baee-b5828a468f4f",
                        "lanport_idx": 1,
                        "dns": "8.8.8.99",
                        "ntp": "192.168.0.99"
                    }
                ]
            },
            "res_num": 1
        }
    ]`

	templateDeviceSpecStorageAndComposableNetwork = []models.Resource{
		{
			ResourceType: "storage",
			ResourceNum:  1,
			ResourceSpec: &models.ResourceSpecification{Condition: []models.Condition{{
				Column:   "vendor",
				Operator: "eq",
				Value:    "Samsung",
			}}},
			Tags: &models.ResStorageTags{IsBootStorage: true},
		},
		{
			ResourceType: "network",
			ResourceNum:  1,
			ResourceSpec: &models.ResourceSpecification{Condition: []models.Condition{{
				Column:   "vendor",
				Operator: "eq",
				Value:    "Mellanox",
			}}},
			Network: &models.Network{
				NicType: models.NicTypeComposable,
				Subnets: []models.Subnet{},
			},
		},
	}

	deviceSpecStorageAndComposableNetwork3rdSubnet = `[
        {
            "res_type": "storage",
            "res_num": 1,
            "tags": {"is_bootstorage": true},
            "res_spec": {
                "condition": [
                    {
                        "column": "vendor",
                        "operator": "eq",
                        "value": "Samsung"
                    }
                ]
            }
        },
        {
            "res_type": "network",
            "res_spec": {
                        "condition": [
                            {
                                "column": "vendor",
                                "operator": "eq",
                                "value": "Mellanox"
                            }
                        ]
            },
            "network": {
                "nic_type": 2,
                "subnets": [
                    {
                        "subnet_uuid": "abc6b24f-3333-3333-3333-b5828a468f4f",
                        "lanport_idx": 1,
                        "dns": "8.8.8.99",
                        "ntp": "192.168.0.99"
                    }
                ]
            },
            "res_num": 1
        }
    ]`

	templateExpectedMachRequest = &models.CreateMachineRequest{
		Tenants: models.CreateMachineRequestBodyTenants{
			TenantUUID: "b3b65e79-ad41-4367-89d6-e4e7315141ef",
			Machines: []models.CreateMachineSpec{
				{
					MachineName: "test_machine_001",
					Resources: []models.CreateMachineResources{
						{
							ResourceSpecifications: []models.Resource{
								{
									ResourceType: "compute",
									ResourceNum:  1,
									ResourceSpec: &models.ResourceSpecification{Condition: []models.Condition{{
										Column:   "model",
										Operator: "eq",
										Value:    "PRIMERGY-RX2540M6",
									}}},
									Network: &models.Network{
										NicType: models.NicTypeOnboard,
										Subnets: []models.Subnet{},
									},
								},
								{
									ResourceType: "storage",
									ResourceNum:  1,
									ResourceSpec: &models.ResourceSpecification{Condition: []models.Condition{{
										Column:   "vendor",
										Operator: "eq",
										Value:    "Samsung",
									}}},
									Tags: &models.ResStorageTags{IsBootStorage: true},
								},
								{
									ResourceType: "network",
									ResourceNum:  1,
									ResourceSpec: &models.ResourceSpecification{Condition: []models.Condition{{
										Column:   "vendor",
										Operator: "eq",
										Value:    "Mellanox",
									}}},
									Network: &models.Network{
										NicType: models.NicTypeComposable,
										Subnets: []models.Subnet{},
									},
								},
							},
						},
					},
				},
			},
		},
	}
)
