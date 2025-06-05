package testutils

import (
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

var (
	GetServerTypePRO2XSRequest = mockutil.Request{
		Method: "GET",
		Path:   "/instance/v1/zones/fr-par-1/products/servers?page=1",
		Status: 200,
		JSON: scwInstance.ListServersTypesResponse{
			Servers: map[string]*scwInstance.ServerType{
				"PRO2-XS": {
					Arch: "x86_64",
				},
			},
			TotalCount: 1,
		},
	}
	GetServerTypePRO2SRequest = mockutil.Request{
		Method: "GET",
		Path:   "/instance/v1/zones/fr-par-1/products/servers?page=1",
		Status: 200,
		JSON: scwInstance.ListServersTypesResponse{
			Servers: map[string]*scwInstance.ServerType{
				"PRO2-S": {
					Arch: "x86_64",
				},
			},
			TotalCount: 1,
		},
	}
	GetImageUbuntu2404Request = mockutil.Request{
		Method: "GET", Path: "/instance/v1/zones/fr-par-1/images/1fa98915-fc85-40d9-95ea-65a06ca8b396",
		Status: 200,
		JSON: scwInstance.GetImageResponse{
			Image: &scwInstance.Image{
				ID:   "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				Zone: "fr-par-1",
			},
		},
	}
)
