package models

const (
	PostMachinesRequestExpected                        = `{"tenants":{"tenant_uuid":"b3b65e79-ad41-4367-89d6-e4e7315141ef","machines":[{"mach_name":"test-machine-001","resources":[{"res_specs":[{"res_type":"compute","res_num":1,"res_spec":{"condition":[{"column":"model","operator":"eq","value":"PRIMERGY-RX2540M6"}]}},{"res_type":"storage","res_num":2,"res_spec":{"condition":[{"column":"type","operator":"eq","value":"NVMe"}]}},{"res_type":"network","res_num":1,"res_spec":{"condition":[{"column":"name","operator":"eq","value":"baremetal-mgmt"}]},"network":{"nic_type":1,"subnets":[{"subnet_uuid":"75e6b24f-c1cc-4009-a871-b5828a468f4f","lanport_idx":1,"default_gw":"192.168.1.1","lease_time":"86400s","ntp":"ntp.example.com","dns":"8.8.8.8"}]}},{"res_type":"network","res_num":2,"res_spec":{"condition":[{"column":"name","operator":"eq","value":"provisioning-net"}]},"network":{"nic_type":2,"subnets":[{"subnet_uuid":"5dc4769c-eef2-407f-b729-fec926ec9eda","lanport_idx":2,"default_gw":"10.0.0.1","lease_time":"1209600000000000s","ntp":"time.google.com"}]}}]}]}]}}`
	CreateMachineRequestExpected                       = `{"tenants":{"tenant_uuid":"b3b65e79-ad41-4367-89d6-e4e7315141ef","machines":[{"mach_name":"test_machine_001","resources":[{"res_specs":[{"res_type":"compute","res_num":1,"res_spec":{"condition":[{"column":"model","operator":"eq","value":"PRIMERGY-RX2540M6"}]},"network":{"nic_type":1,"subnets":[{"subnet_uuid":"5dc4769c-eef2-407f-b729-fec926ec9eda","lanport_idx":1,"default_gw":"192.168.0.1","ntp":"192.168.0.1","dns":"8.8.8.8"},{"subnet_uuid":"75e6b24f-c1cc-4009-a871-b5828a468f4f","lanport_idx":2,"default_gw":"172.0.0.1","ntp":"192.168.0.1","dns":"8.8.8.8"}]}},{"res_type":"storage","res_num":1,"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]}]}]}}`
	CreateMachineRequestOneProvisioningNetworkExpected = `{"tenants":{"tenant_uuid":"b3b65e79-ad41-4367-89d6-e4e7315141ef","machines":[{"mach_name":"test_machine_001","resources":[{"res_specs":[{"res_type":"compute","res_num":1,"res_spec":{"condition":[{"column":"model","operator":"eq","value":"PRIMERGY-RX2540M6"}]},"network":{"nic_type":1,"subnets":[{"subnet_uuid":"5dc4769c-eef2-407f-b729-fec926ec9eda","lanport_idx":2,"default_gw":"192.168.0.1","ntp":"192.168.0.1","dns":"8.8.8.8"}]}},{"res_type":"storage","res_num":1,"res_spec":{"condition":[{"column":"vendor","operator":"eq","value":"samsung"}]}},{"res_type":"gpu","res_num":2,"res_spec":{"condition":[{"column":"gpu_model","operator":"eq","value":"NVIDIA Tesla T4"}]}}]}]}]}}`
	PostMachinesResponseExample                        = `{
		"data":{
			"machines":[
			{
				"mach_uuid":"59756ed2-6a42-47f2-bc54-117bcf6bdce3",
				"mach_id":1,
				"mach_name":"machine-01",
				"mach_owner":"domzalskis",
				"resources":[
				{
					"res_uuid":"c1a4e32f-ea8f-4eff-8c8c-55d473deb1a0",
					"res_name":"cpu-01",
					"res_type":"compute",
					"res_status":1,
					"res_op_status":"1",
					"res_spec":{
					"condition":[
						{
						"column":"model",
						"operator":"eq",
						"value":"PRIMERGRYRX2540M4"
						}
					]
					}
				},
				{
					"res_uuid":"29e3d171-2441-4df9-9cc3-44c928daf41e",
					"res_name":"storage-01",
					"res_type":"storage",
					"res_status":1,
					"res_op_status":"1",
					"res_spec":{
					"condition":[
						{
						"column":"capacity",
						"operator":"eq",
						"value":"16TB"
						}
					]
					}
				},
				{
					"res_uuid":"6b5f0567-921f-4ef3-a6e5-11f7e2609857",
					"res_name":"network-provisioning",
					"res_type":"network",
					"res_status":1,
					"res_op_status":"1",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"provisioning"
						}
					]
					},
					"network":{
					"nic_type":1,
					"subnets":[
						{
						"subnet_uuid":"6b5f0567-921f-4ef3-a6e5-11f7e2609857",
						"lanport_idx":1,
						"default_gw":"gateway-address",
						"lease_time":"",
						"ntp":"",
						"dns":"",
						"fqdn":""
						}
					]
					}
				},
				{
					"res_uuid":"991fbfd5-3521-4098-880e-1d9d2c8d2705",
					"res_name":"network-cluster",
					"res_type":"network",
					"res_status":2,
					"res_op_status":"2",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"cluster"
						}
					]
					},
					"network":{
					"nic_type":2,
					"subnets":[
						{
						"subnet_uuid":"991fbfd5-3521-4098-880e-1d9d2c8d2705",
						"lanport_idx":2,
						"default_gw":"gateway-address",
						"lease_time":"",
						"ntp":"",
						"dns":"",
						"fqdn":""
						}
					]
					}
				}
				]
			}
			]
		}
	}`
	GetMachineResponseExample = `{
		"data":{
			"machines":[
			{
				"fabric_uuid":"58f4c0f8-6c74-4e86-a560-95ed13daaa46",
				"fabric_id":1,
				"mach_uuid":"a1b2c3d4-e5f6-7890-1234-567890abcdef",
				"mach_id_nonliqid":12345,
				"mach_id":67890,
				"mach_name":"example-machine-01",
				"mach_status":1,
				"mach_op_status":"00",
				"mach_status_detail":"Running",
				"mach_owner":"user123",
				"grp_uuid":"f0e9d8c7-b6a5-4321-0987-6543210fedcb",
				"boot_ssd":"e9d8c7b6-a543-2109-8765-43210fedcba0",
				"lanports":[
				{
					"lanport_uuid":"d8c7b6a5-4321-0987-6543-210fedcba098",
					"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
					"mac_address":"00:11:22:33:44:55",
					"lanport_idx":1,
					"nw_class_cu":null,
					"ip_address":"192.168.2.100"
				},
				{
					"lanport_uuid":"01085c2c-15c4-4957-9ad3-7d1ee481f082",
					"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
					"mac_address":"00:11:22:33:44:66",
					"lanport_idx":2,
					"nw_class_cu":null,
					"ip_address":"192.168.2.150"
				},
				{
					"lanport_uuid":"c7b6a543-2109-8765-4321-0fedcba09876",
					"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
					"mac_address":"00:11:22:33:44:77",
					"lanport_idx":3,
					"nw_class_cu":null,
					"ip_address":"10.0.0.100"
				},
				{
					"lanport_uuid":"a7d09755-d5c9-49ae-8f8c-7f53a3ae4f69",
					"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
					"mac_address":"11:11:11:11:11:11",
					"lanport_idx":4,
					"nw_class_cu":null,
					"ip_address":"10.0.0.200"
				},
				{
					"lanport_uuid":"c7b6a543-2109-8765-4321-0fedcba09876",
					"subnet_uuid":"03aa247b-dd21-4dd0-943c-1b878bb6cccc",
					"mac_address":"22:22:22:22:22:22",
					"nw_class_cu":null,
					"lanport_idx":0,
					"ip_address":""
				}
				],
				"resources":[
				{
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
				},
				{
					"res_uuid":"a5432109-8765-4321-0fed-cba098765432",
					"res_name":"storage-resource-1",
					"res_type":"storage",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"storage_size",
						"operator":"gt",
						"value":"100"
						}
					]
					}
				},
				{
					"res_uuid":"bbb32109-8765-4321-0fed-cba098765432",
					"res_name":"storage-resource-2",
					"res_type":"storage",
					"res_status":1,
					"res_op_status":"0",
					"tags":{"is_bootstorage":true},
					"res_spec":{
					"condition":[
						{
						"column": "model",
                    	"operator": "eq",
                    	"value": "ssd"
						}
					]
					}
				},
				{
					"res_uuid":"43210987-6543-210f-edcb-a09876543210",
					"res_name":"gpu-resource-1",
					"res_type":"gpu",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"gpu_model",
						"operator":"eq",
						"value":"NVIDIA Tesla T4"
						}
					]
					}
				},
				{
					"res_uuid":"21098765-4321-0fed-cba0-987654321098",
					"res_name":"network-resource-1",
					"res_type":"network",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"provisioning"
						}
					]
					},
					"network":{
					"nic_type":1,
					"subnets":[
						{
						"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
						"lanport_idx":1,
						"default_gw":"192.168.2.1",
						"lease_time":"86400",
						"ntp":"192.168.2.2",
						"dns":"8.8.8.8",
						"fqdn":"host1.example.com"
						}
					]
					}
				},
				{
					"res_uuid":"09876543-210f-edcb-a098-765432109876",
					"res_name":"network-resource-2",
					"res_type":"network",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"cluster"
						}
					]
					},
					"network":{
					"nic_type":1,
					"subnets":[
						{
						"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
						"lanport_idx":2,
						"default_gw":"10.0.0.1",
						"lease_time":"3600",
						"ntp":"10.0.0.2",
						"dns":"1.1.1.1",
						"fqdn":"host2.example.com"
						}
					]
					}
				}
				]
			}
			]
		}
	}`
	GetMachineResponseExampleWithTypoInStorageResSpec = `{
		"data":{
			"machines":[
			{
				"fabric_uuid":"58f4c0f8-6c74-4e86-a560-95ed13daaa46",
				"fabric_id":1,
				"mach_uuid":"a1b2c3d4-e5f6-7890-1234-567890abcdef",
				"mach_id_nonliqid":12345,
				"mach_id":67890,
				"mach_name":"example-machine-01",
				"mach_status":1,
				"mach_op_status":"00",
				"mach_status_detail":"Running",
				"mach_owner":"user123",
				"grp_uuid":"f0e9d8c7-b6a5-4321-0987-6543210fedcb",
				"boot_ssd":"e9d8c7b6-a543-2109-8765-43210fedcba0",
				"lanports":[
				{
					"lanport_uuid":"d8c7b6a5-4321-0987-6543-210fedcba098",
					"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
					"mac_address":"00:11:22:33:44:55",
					"lanport_idx":1,
					"nw_class_cu":null,
					"ip_address":"192.168.2.100"
				},
				{
					"lanport_uuid":"01085c2c-15c4-4957-9ad3-7d1ee481f082",
					"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
					"mac_address":"00:11:22:33:44:66",
					"lanport_idx":2,
					"nw_class_cu":null,
					"ip_address":"192.168.2.150"
				},
				{
					"lanport_uuid":"c7b6a543-2109-8765-4321-0fedcba09876",
					"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
					"mac_address":"00:11:22:33:44:77",
					"lanport_idx":3,
					"nw_class_cu":null,
					"ip_address":"10.0.0.100"
				},
				{
					"lanport_uuid":"a7d09755-d5c9-49ae-8f8c-7f53a3ae4f69",
					"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
					"mac_address":"11:11:11:11:11:11",
					"lanport_idx":4,
					"nw_class_cu":null,
					"ip_address":"10.0.0.200"
				},
				{
					"lanport_uuid":"c7b6a543-2109-8765-4321-0fedcba09876",
					"subnet_uuid":"03aa247b-dd21-4dd0-943c-1b878bb6cccc",
					"mac_address":"22:22:22:22:22:22",
					"nw_class_cu":null,
					"lanport_idx":0,
					"ip_address":""
				}
				],
				"resources":[
				{
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
				},
				{
					"res_uuid":"a5432109-8765-4321-0fed-cba098765432",
					"res_name":"storage-resource-1",
					"res_type":"storage",
					"res_status":1,
					"res_op_status":"0",
					"res_spcec":{
					"condition":[
						{
						"column":"storage_size",
						"operator":"gt",
						"value":"100"
						}
					]
					}
				},
				{
					"res_uuid":"bbb32109-8765-4321-0fed-cba098765432",
					"res_name":"storage-resource-2",
					"res_type":"storage",
					"res_status":1,
					"res_op_status":"0",
					"tags":{"is_bootstorage":true},
					"res_spcec":{
					"condition":[
						{
						"column": "model",
                    	"operator": "eq",
                    	"value": "ssd"
						}
					]
					}
				},
				{
					"res_uuid":"43210987-6543-210f-edcb-a09876543210",
					"res_name":"gpu-resource-1",
					"res_type":"gpu",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"gpu_model",
						"operator":"eq",
						"value":"NVIDIA Tesla T4"
						}
					]
					}
				},
				{
					"res_uuid":"21098765-4321-0fed-cba0-987654321098",
					"res_name":"network-resource-1",
					"res_type":"network",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"provisioning"
						}
					]
					},
					"network":{
					"nic_type":1,
					"subnets":[
						{
						"subnet_uuid":"123e4567-e89b-12d3-a456-426614174000",
						"lanport_idx":1,
						"default_gw":"192.168.2.1",
						"lease_time":"86400",
						"ntp":"192.168.2.2",
						"dns":"8.8.8.8",
						"fqdn":"host1.example.com"
						}
					]
					}
				},
				{
					"res_uuid":"09876543-210f-edcb-a098-765432109876",
					"res_name":"network-resource-2",
					"res_type":"network",
					"res_status":1,
					"res_op_status":"0",
					"res_spec":{
					"condition":[
						{
						"column":"name",
						"operator":"eq",
						"value":"cluster"
						}
					]
					},
					"network":{
					"nic_type":1,
					"subnets":[
						{
						"subnet_uuid":"78901234-5678-9abc-def0-1234567890ab",
						"lanport_idx":2,
						"default_gw":"10.0.0.1",
						"lease_time":"3600",
						"ntp":"10.0.0.2",
						"dns":"1.1.1.1",
						"fqdn":"host2.example.com"
						}
					]
					}
				}
				]
			}
			]
		}
	}`
	GetMachineResponseExampleDeletedMachine = `{
		"data": {
			"machines": [
				{
					"mach_uuid": "c8aa9fa9-dd08-4735-bc96-c4c417f6aabb",
					"mach_id_nonliqid": null,
					"mach_id": null,
					"mach_name": null,
					"mach_status": 17,
					"mach_op_status": "00",
					"mach_status_detail": " DELETED",
					"mach_owner": "AnonymousUser",
					"tenant_uuid": "284a6a91-48d6-4fd0-a2a2-4413874c6a01",
					"boot_ssd": null,
					"lanports": [],
					"resources": []
				}
			]
		}
	}`
	DeleteMachineResponseExample = `{
		"data":{
			"machines":[
			{
				"mach_uuid":"c7b6a543-2109-8765-4321-0fedcba09876"
			}
			]
		}
	}`
	ImageInstallPutPayloadExample = `{
		"resources":{
			"res_uuid_ssd":"c7b6a543-2109-8765-4321-0fedcba09876",
			"bootimg_filename":"boot-image-linux-01"
		}
	}`

	ImageInstallPutResponseExample = `{"resources":{"res_uuid_ssd":"c7b6a543-2109-8765-4321-0fedcba09876","bootimg_filename":"my-boot-image-02"}}`

	AccessTokenExample = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"

	TestApiUrl                                = "http://localhost"
	TestPort                                  = "8080"
	TestRealm                                 = "cdi-test"
	TestUserNameNotAllowedToCreateCluster     = "alice"
	TestUserPasswordNotAllowedToCreateCluster = "alice"
	TestUserNameAllowedToCreateCluster        = "james"
	TestUserPasswordAllowedToCreateCluster    = "james"
	TestClientId                              = "test-client"
	TestClientSecret                          = "test-client"

	TestBearerTokenRequestBody = "client_id=test-client&client_secret=test-client&grant_type=password&password=alice&response=id_token+token&scope=openid&username=alice"
	// TestAccessTokenExpected    = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"
	TestAccessTokenExpected  = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"
	TestRefreshTokenExpected = "eyJhbGciOiJIUzUxMiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjZGY0OWRmMy1jNWE0LTRlYjItYmUxOS1jMTA1ZjdlMWUxMDMifQ.eyJleHAiOjE3MzE1MTE1MjMsImlhdCI6MTczMTUwOTcyMywianRpIjoiOWI2NmYxYmItOTc1Ni00YjRmLTkzN2QtYjZmYjk5MmQyOTEwIiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvcmVhbG1zL21hc3RlciIsInN1YiI6IjEzYTkyMTdmLWRiMjMtNDhhNi05ZWZhLWFiY2RhYzgxZmZlZCIsInR5cCI6IlJlZnJlc2giLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIn0.bQXuYIWqKdiiLQ6hpHFwT24c_BYUWgn3UUk8hehCJKeDOjk12VKIotyOnHeEUmeYS10hotUpxMhTzFAUVSMG1w"
	TestBearerTokenResponse  = `{
		"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw",
		"expires_in": 60,
		"refresh_expires_in": 1800,
		"refresh_token": "eyJhbGciOiJIUzUxMiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjZGY0OWRmMy1jNWE0LTRlYjItYmUxOS1jMTA1ZjdlMWUxMDMifQ.eyJleHAiOjE3MzE1MTE1MjMsImlhdCI6MTczMTUwOTcyMywianRpIjoiOWI2NmYxYmItOTc1Ni00YjRmLTkzN2QtYjZmYjk5MmQyOTEwIiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvcmVhbG1zL21hc3RlciIsInN1YiI6IjEzYTkyMTdmLWRiMjMtNDhhNi05ZWZhLWFiY2RhYzgxZmZlZCIsInR5cCI6IlJlZnJlc2giLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIn0.bQXuYIWqKdiiLQ6hpHFwT24c_BYUWgn3UUk8hehCJKeDOjk12VKIotyOnHeEUmeYS10hotUpxMhTzFAUVSMG1w",
		"token_type": "Bearer",
		"not-before-policy": 0,
		"session_state": "d06f8d5f-4146-4a77-916d-50584666dcab",
		"scope": "profile email"
		}`
	TestBearerTokenResponseWithoutKeyAccessToken = `{
			"non": "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw",
			"expires_in": 60,
			"refresh_expires_in": 1800,
			"refresh_token": "eyJhbGciOiJIUzUxMiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjZGY0OWRmMy1jNWE0LTRlYjItYmUxOS1jMTA1ZjdlMWUxMDMifQ.eyJleHAiOjE3MzE1MTE1MjMsImlhdCI6MTczMTUwOTcyMywianRpIjoiOWI2NmYxYmItOTc1Ni00YjRmLTkzN2QtYjZmYjk5MmQyOTEwIiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvcmVhbG1zL21hc3RlciIsInN1YiI6IjEzYTkyMTdmLWRiMjMtNDhhNi05ZWZhLWFiY2RhYzgxZmZlZCIsInR5cCI6IlJlZnJlc2giLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIn0.bQXuYIWqKdiiLQ6hpHFwT24c_BYUWgn3UUk8hehCJKeDOjk12VKIotyOnHeEUmeYS10hotUpxMhTzFAUVSMG1w",
			"token_type": "Bearer",
			"not-before-policy": 0,
			"session_state": "d06f8d5f-4146-4a77-916d-50584666dcab",
			"scope": "profile email"
			}`
	TestBearerTokenResponseWithoutTokenKeys = `{
			"non": "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw",
			"expires_in": 60,
			"refresh_expires_in": 1800,
			"token_type": "Bearer",
			"not-before-policy": 0,
			"session_state": "d06f8d5f-4146-4a77-916d-50584666dcab",
			"scope": "profile email"
			}`

	TestPgcdiPrivilegesTenant              = "cdi-test"
	TestPgcdiPrivilegesRequestBody         = "client_id=test-client&client_secret=test-client&token="
	TestPgcdiPrivilegesResponseInvalidRole = `{
		"exp": 1730928693,
		"iat": 1730926943,
		"jti": "02400af7-58cf-41df-adcc-0d6a95f5b839",
		"iss": "http://localhost:8080/realms/cdi-test",
		"aud": [
			"realm-management",
			"account"
		],
		"sub": "2b6c6b37-09c6-42a4-bf8c-9c55ebf48f36",
		"typ": "Bearer",
		"azp": "test-client",
		"session_state": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"acr": "1",
		"resource_access": {
			"realm-management": {
				"roles": [
					"view-clients",
					"query-clients"
				]
			},
			"account": {
				"roles": [
					"manage-account",
					"manage-account-links",
					"view-profile"
				]
			}
		},
		"scope": "openid email pgcdi_privileges profile",
		"sid": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"email_verified": false,
		"name": "Alice Liddel",
		"pgcdi_privileges": {
			"roles": [
				"cluster_manager"
			],
			"clusters": [
				"cluster_a",
				"cluster_b"
			],
			"tenant": "cdi-test"
		},
		"preferred_username": "alice",
		"given_name": "Alice",
		"family_name": "Liddel",
		"email": "alice@keycloak.org",
		"client_id": "test-client",
		"username": "alice",
		"token_type": "Bearer",
		"active": true
	}`
	TestPgcdiPrivilegesResponseValidRole = `{
		"exp": 1730928693,
		"iat": 1730926943,
		"jti": "02400af7-58cf-41df-adcc-0d6a95f5b839",
		"iss": "http://localhost:8080/realms/cdi-test",
		"aud": [
			"realm-management",
			"account"
		],
		"sub": "2b6c6b37-09c6-42a4-bf8c-9c55ebf48f36",
		"typ": "Bearer",
		"azp": "test-client",
		"session_state": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"acr": "1",
		"resource_access": {
			"realm-management": {
				"roles": [
					"view-clients",
					"query-clients"
				]
			},
			"account": {
				"roles": [
					"manage-account",
					"manage-account-links",
					"view-profile"
				]
			}
		},
		"scope": "openid email pgcdi_privileges profile",
		"sid": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"email_verified": false,
		"name": "James Kirk",
		"pgcdi_privileges": {
			"roles": [
				"system_manager"
			],
			"clusters": [
				"cluster_b",
				"cluster_c"
			],
			"tenant": "cdi-test"
		},
		"preferred_username": "james",
		"given_name": "James",
		"family_name": "Kirk",
		"email": "james@keycloak.org",
		"client_id": "test-client",
		"username": "james",
		"token_type": "Bearer",
		"active": true
	}`

	TestPgcdiPrivilegesResponseValidRoleWithoutPriviledges = `{
		"exp": 1730928693,
		"iat": 1730926943,
		"jti": "02400af7-58cf-41df-adcc-0d6a95f5b839",
		"iss": "http://localhost:8080/realms/cdi-test",
		"aud": [
			"realm-management",
			"account"
		],
		"sub": "2b6c6b37-09c6-42a4-bf8c-9c55ebf48f36",
		"typ": "Bearer",
		"azp": "test-client",
		"session_state": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"acr": "1",
		"resource_access": {
			"realm-management": {
				"roles": [
					"view-clients",
					"query-clients"
				]
			},
			"account": {
				"roles": [
					"manage-account",
					"manage-account-links",
					"view-profile"
				]
			}
		},
		"scope": "openid email pgcdi_privileges profile",
		"sid": "3c8257b1-1517-4c9a-898f-5be3f25b2300",
		"email_verified": false,
		"name": "James Kirk",
		"preferred_username": "james",
		"given_name": "James",
		"family_name": "Kirk",
		"email": "james@keycloak.org",
		"client_id": "test-client",
		"username": "james",
		"token_type": "Bearer",
		"active": true
	}`

	ProviderIdDropInCreationScriptExpected = `
#!/bin/sh
for d in k3s rke2; do
mkdir -p /etc/rancher/${d}/config.yaml.d
cat << EOF > /etc/rancher/${d}/config.yaml.d/100-kubelet-provider-id.yaml
kubelet-arg+: "provider-id=fsas://cdd792f2-5591-4c18-a8bd-1c39e55dedfa"
EOF
done
`

	DeviceSpecsValid = `
	[
		{
			"res_type": "storage",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "hdd"
					}
				]
			}
		},
		{
			"res_type": "storage",
			"res_num": 1,
			"tags": {
				"is_bootstorage": true
			},
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "ssd"
					}
				]
			}
		}
	]`

	DeviceSpecsInvalidNoFieldTagsIsBootStorage = `
	[
		{
			"res_type": "storage",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "ssd"
					}
				]
			}
		},
		{
			"res_type": "storage",
			"res_num": 1,
			"tags": {},
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "ssd"
					}
				]
			}
		}
	]`

	DeviceSpecsInCorrectValueForTagsIsBootStorage = `
	[
		{
			"res_type": "storage",
			"res_num": 1,
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "ssd"
					}
				]
			}
		},
		{
			"res_type": "storage",
			"res_num": 1,
			"tags": {
				"is_bootstorage": false
			},
			"res_spec": {
				"condition": [
					{
						"column": "model",
						"operator": "eq",
						"value": "ssd"
					}
				]
			}
		}
	]`
)

var ExpectedLanports = []Lanport{
	{
		LanportUUID: "d8c7b6a5-4321-0987-6543-210fedcba098",
		SubnetUUID:  "123e4567-e89b-12d3-a456-426614174000",
		MacAddress:  "00:11:22:33:44:55",
		LanportIdx:  1,
		IPAddress:   "192.168.2.100",
	},
	{
		LanportUUID: "01085c2c-15c4-4957-9ad3-7d1ee481f082",
		SubnetUUID:  "123e4567-e89b-12d3-a456-426614174000",
		MacAddress:  "00:11:22:33:44:66",
		LanportIdx:  2,
		IPAddress:   "192.168.2.150",
	},
	{
		LanportUUID: "c7b6a543-2109-8765-4321-0fedcba09876",
		SubnetUUID:  "78901234-5678-9abc-def0-1234567890ab",
		MacAddress:  "00:11:22:33:44:77",
		LanportIdx:  3,
		IPAddress:   "10.0.0.100",
	},
	{
		LanportUUID: "a7d09755-d5c9-49ae-8f8c-7f53a3ae4f69",
		SubnetUUID:  "78901234-5678-9abc-def0-1234567890ab",
		MacAddress:  "11:11:11:11:11:11",
		LanportIdx:  4,
		IPAddress:   "10.0.0.200",
	},
	{
		LanportUUID: "c7b6a543-2109-8765-4321-0fedcba09876",
		SubnetUUID:  "03aa247b-dd21-4dd0-943c-1b878bb6cccc",
		MacAddress:  "22:22:22:22:22:22",
		LanportIdx:  0,
		IPAddress:   "",
	},
}
