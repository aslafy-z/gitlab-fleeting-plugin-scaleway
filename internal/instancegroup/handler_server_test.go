package instancegroup

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func TestServerHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "1", Zone: "fr-par-1", Address: net.ParseIP("203.0.113.1")},
				},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "2", Zone: "fr-par-1", Address: net.ParseIP("2001:db8:5678::1")},
				},
			},
			// {
			// 	Method: "GET", Path: "/instance/v1/zones/fr-par-1/products/servers/availability?page=1",
			// 	Status: 201,
			// 	JSON: scwInstance.GetServerTypesAvailabilityResponse{
			// 		Servers: map[string]*scwInstance.GetServerTypesAvailabilityResponseAvailability{
			// 			"PRO2-XS": {
			// 				Availability: scwInstance.ServerTypesAvailabilityAvailable,
			// 			},
			// 		},
			// 	},
			// },
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers",
				Status: 201,
				JSON: scwInstance.CreateServerResponse{
					Server: &scwInstance.Server{ID: "1", Name: "fleeting-a", Zone: scw.Zone("fr-par-1"), Arch: "x86_64", Volumes: map[string]*scwInstance.VolumeServer{"0": {ID: "1", Zone: scw.Zone("fr-par-1")}}},
				},
			},
			{
				Method: "PATCH", Path: "/instance/v1/zones/fr-par-1/servers/1/user_data/cloud-init",
				Status: 204,
			},
			{
				Method: "PATCH", Path: "/block/v1/zones/fr-par-1/volumes/1",
				Status: 204,
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers/1/action",
				Status: 200,
				JSON: scwInstance.ServerActionResponse{
					Task: &scwInstance.Task{
						ID:     "task-1",
						Zone:   scw.Zone("fr-par-1"),
						Status: scwInstance.TaskStatusPending,
					},
				},
			},
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &ServerHandler{}

		require.NoError(t, handler.Create(ctx, group, instance))

		assert.NotNil(t, instance.ID)
		assert.NotNil(t, instance.waitFn)
	})
	t.Run("success with second server type", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "1", Zone: "fr-par-1", Address: net.ParseIP("203.0.113.1")},
				},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "2", Zone: "fr-par-1", Address: net.ParseIP("2001:db8:5678::1")},
				},
			},
			// {
			// 	Method: "GET", Path: "/instance/v1/zones/fr-par-1/products/servers/availability?page=1",
			// 	Status: 201,
			// 	JSON: scwInstance.GetServerTypesAvailabilityResponse{
			// 		Servers: map[string]*scwInstance.GetServerTypesAvailabilityResponseAvailability{
			// 			"PRO2-XS": {
			// 				Availability: scwInstance.ServerTypesAvailabilityShortage,
			// 			},
			// 		},
			// 	},
			// },
			// {
			// 	Method: "GET", Path: "/instance/v1/zones/fr-par-1/products/servers/availability?page=1",
			// 	Status: 201,
			// 	JSON: scwInstance.GetServerTypesAvailabilityResponse{
			// 		Servers: map[string]*scwInstance.GetServerTypesAvailabilityResponseAvailability{
			// 			"PRO2-S": {
			// 				Availability: scwInstance.ServerTypesAvailabilityAvailable,
			// 			},
			// 		},
			// 	},
			// },
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers",
				Status: 412,
				JSON:   scw.ResponseError{Type: "out_of_stock", Resource: "server"},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers",
				Status: 201,
				JSON: scwInstance.CreateServerResponse{
					Server: &scwInstance.Server{ID: "1", Name: "fleeting-a", Zone: scw.Zone("fr-par-1"), Arch: "x86_64", Volumes: map[string]*scwInstance.VolumeServer{"0": {ID: "1", Zone: scw.Zone("fr-par-1")}}},
				},
			},
			{
				Method: "PATCH", Path: "/instance/v1/zones/fr-par-1/servers/1/user_data/cloud-init",
				Status: 204,
			},
			{
				Method: "PATCH", Path: "/block/v1/zones/fr-par-1/volumes/1",
				Status: 204,
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers/1/action",
				Status: 200,
				JSON: scwInstance.ServerActionResponse{
					Task: &scwInstance.Task{
						ID:     "task-1",
						Zone:   scw.Zone("fr-par-1"),
						Status: scwInstance.TaskStatusPending,
					},
				},
			},
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &ServerHandler{}

		require.NoError(t, handler.Create(ctx, group, instance))

		assert.NotNil(t, instance.ID)
		assert.NotNil(t, instance.waitFn)
	})
	t.Run("failure with second server type", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "1", Zone: "fr-par-1", Address: net.ParseIP("203.0.113.1")},
				},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/ips",
				Status: 201,
				JSON: scwInstance.CreateIPResponse{
					IP: &scwInstance.IP{ID: "2", Zone: "fr-par-1", Address: net.ParseIP("2001:db8:5678::1")},
				},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers",
				Status: 412,
				JSON:   scw.ResponseError{Type: "out_of_stock", Resource: "server"},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers",
				Status: 412,
				JSON:   scw.ResponseError{Type: "out_of_stock", Resource: "server"},
			},
			// {
			// 	Method: "GET", Path: "/instance/v1/zones/fr-par-1/products/servers/availability?page=1",
			// 	Status: 201,
			// 	JSON: scwInstance.GetServerTypesAvailabilityResponse{
			// 		Servers: map[string]*scwInstance.GetServerTypesAvailabilityResponseAvailability{
			// 			"PRO2-XS": {
			// 				Availability: scwInstance.ServerTypesAvailabilityShortage,
			// 			},
			// 		},
			// 	},
			// },
			// {
			// 	Method: "GET", Path: "/instance/v1/zones/fr-par-1/products/servers/availability?page=1",
			// 	Status: 201,
			// 	JSON: scwInstance.GetServerTypesAvailabilityResponse{
			// 		Servers: map[string]*scwInstance.GetServerTypesAvailabilityResponseAvailability{
			// 			"PRO2-S": {
			// 				Availability: scwInstance.ServerTypesAvailabilityShortage,
			// 			},
			// 		},
			// 	},
			// },
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &ServerHandler{}

		require.EqualError(t,
			handler.Create(ctx, group, instance),
			"could not request instance creation: scaleway-sdk-go: resource server is out of stock",
		)
	})
}

func TestServerHandlerCleanup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "GET",
				Path:   "/instance/v1/zones/fr-par-1/servers/1",
				Status: 200,
				JSON: scwInstance.GetServerResponse{
					Server: &scwInstance.Server{
						ID:   "1",
						Name: "fleeting-a",
						Zone: scw.Zone("fr-par-1"),
						Arch: "x86_64",
						Volumes: map[string]*scwInstance.VolumeServer{
							"0": {ID: "1", Zone: scw.Zone("fr-par-1")},
						},
						PublicIPs: []*scwInstance.ServerIP{
							{ID: "1", Address: net.ParseIP("203.0.113.1")},
							{ID: "2", Address: net.ParseIP("2001:db8:5678::1")},
						},
					},
				},
			},
			{
				Method: "DELETE", Path: "/instance/v1/zones/fr-par-1/ips/1",
				Status: 204,
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers/1/action",
				Status: 200,
				JSON: scwInstance.ServerActionResponse{
					Task: &scwInstance.Task{
						ID:     "task-1",
						Zone:   scw.Zone("fr-par-1"),
						Status: scwInstance.TaskStatusPending,
					},
				},
			},
			{
				Method: "GET", Path: "/instance/v1/zones/fr-par-1/servers/1",
				Status: 200,
				JSON: scwInstance.GetServerResponse{
					Server: &scwInstance.Server{
						ID:    "1",
						Name:  "fleeting-a",
						Zone:  scw.Zone("fr-par-1"),
						State: scwInstance.ServerStateStopped,
					},
				},
			},
			{
				Method: "POST", Path: "/instance/v1/zones/fr-par-1/servers/1/detach-volume",
				Status: 200,
				JSON: scwInstance.DetachServerVolumeResponse{
					Server: &scwInstance.Server{
						ID:      "1",
						Name:    "fleeting-a",
						Zone:    scw.Zone("fr-par-1"),
						State:   scwInstance.ServerStateStopped,
						Volumes: map[string]*scwInstance.VolumeServer{},
					},
				},
			},
			{
				Method: "DELETE", Path: "/instance/v1/zones/fr-par-1/volumes/1",
				Status: 204,
			},
			{
				Method: "DELETE", Path: "/instance/v1/zones/fr-par-1/servers/1",
				Status: 204,
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: "1"}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, "1", instance.ID)
		assert.NotNil(t, instance.waitFn)
	})

	t.Run("success not found", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "GET",
				Path:   "/instance/v1/zones/fr-par-1/servers/1",
				Status: 404,
				JSON:   scw.ResponseError{Type: "not_found", Resource: "server"},
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: "1"}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, "1", instance.ID)
		assert.Nil(t, instance.waitFn)
	})

	t.Run("passthrough", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{})

		instance := &Instance{Name: "fleeting-a"}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, "", instance.ID)
		assert.Nil(t, instance.waitFn)
	})
}
