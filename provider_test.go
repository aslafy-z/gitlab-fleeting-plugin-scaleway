package scaleway

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"go.uber.org/mock/gomock"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/instancegroup"
	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/testutils"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwIam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

func sshKeyFixture(t *testing.T) ([]byte, scwIam.SSHKey) {
	t.Helper()

	privateKey, publicKey, err := sshutil.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	fingerprint, err := sshutil.GetPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Fatal(err)
	}

	return privateKey, scwIam.SSHKey{ID: "1", Name: "fleeting", ProjectID: "e0660b65-9dce-4f25-854d-1161a1aa96a9", Fingerprint: fingerprint, PublicKey: string(publicKey)}
}

func TestInit(t *testing.T) {
	sshPrivateKey, sshKey := sshKeyFixture(t)

	testCases := []struct {
		name     string
		requests []mockutil.Request
		run      func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings)
	}{
		{name: "generated ssh key upload",
			requests: []mockutil.Request{
				{
					Method: "GET",
					Path:   "/iam/v1alpha1/ssh-keys?disabled=false&order_by=created_at_asc&page=1",
					Status: 200,
					JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{}, TotalCount: 0},
				},
				{
					Method: "POST", Path: "/iam/v1alpha1/ssh-keys",
					Status: 200,
					JSON:   scwIam.SSHKey(sshKey),
				},
				testutils.GetServerTypePRO2XSRequest,
				testutils.GetServerTypePRO2SRequest,
				testutils.GetImageUbuntu2404UUIDRequest,
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "scaleway/fr-par-1/fleeting", info.ID)
			},
		},
		{name: "static ssh key upload",
			requests: []mockutil.Request{
				{
					Method: "GET",
					Path:   "/iam/v1alpha1/ssh-keys?disabled=false&order_by=created_at_asc&page=1",
					Status: 200,
					JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{}, TotalCount: 0},
				},
				{
					Method: "POST", Path: "/iam/v1alpha1/ssh-keys",
					Status: 200,
					JSON:   scwIam.SSHKey(sshKey),
				},
				testutils.GetServerTypePRO2XSRequest,
				testutils.GetServerTypePRO2SRequest,
				testutils.GetImageUbuntu2404UUIDRequest,
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				settings.UseStaticCredentials = true
				settings.Key = sshPrivateKey

				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "scaleway/fr-par-1/fleeting", info.ID)
			},
		},
		{name: "static ssh key existing",
			requests: []mockutil.Request{
				{
					Method: "GET",
					Path:   "/iam/v1alpha1/ssh-keys?disabled=false&order_by=created_at_asc&page=1",
					Status: 200,
					JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{&sshKey}, TotalCount: 1},
				},
				testutils.GetServerTypePRO2XSRequest,
				testutils.GetServerTypePRO2SRequest,
				testutils.GetImageUbuntu2404UUIDRequest,
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				settings.UseStaticCredentials = true
				settings.Key = sshPrivateKey

				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "scaleway/fr-par-1/fleeting", info.ID)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(mockutil.Handler(t, testCase.requests))

			group := &InstanceGroup{
				Name: "fleeting",

				AccessKey:    "SCWAXXXXXXXXXXXXXXXX",
				SecretKey:    "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization: "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:      "e0660b65-9dce-4f25-854d-1161a1aa96a9",

				Endpoint:    server.URL,
				Zone:        "fr-par-1",
				ServerTypes: []string{"PRO2-XS", "PRO2-S"},
				Image:       "1fa98915-fc85-40d9-95ea-65a06ca8b396",
			}
			ctx := context.Background()
			log := hclog.New(hclog.DefaultOptions)
			settings := provider.Settings{}

			testCase.run(t, group, ctx, log, settings)
		})
	}
}

func TestIncrease(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 3

				mock.EXPECT().
					Increase(ctx, 2).
					Return([]string{"fleeting-a:1", "fleeting-b:2"}, nil)

				mock.EXPECT().
					Sanity(ctx).
					Return(nil)

				count, err := group.Increase(ctx, 2)
				require.NoError(t, err)
				require.Equal(t, 2, count)
				require.Equal(t, 5, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 3

				mock.EXPECT().
					Increase(ctx, 2).
					Return([]string{"fleeting-a:1"}, fmt.Errorf("some error"))

				mock.EXPECT().
					Sanity(ctx).
					Return(nil)

				count, err := group.Increase(ctx, 2)
				require.Error(t, err)
				require.Equal(t, 1, count)
				require.Equal(t, 4, group.size)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.New(hclog.DefaultOptions),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestDecrease(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 2

				mock.EXPECT().
					Decrease(ctx, []string{"fleeting-a:1", "fleeting-b:2"}).
					Return([]string{"fleeting-a:1", "fleeting-b:2"}, nil)

				mock.EXPECT().
					Sanity(ctx).
					Return(nil)

				result, err := group.Decrease(ctx, []string{"fleeting-a:1", "fleeting-b:2"})
				require.NoError(t, err)
				require.Equal(t, []string{"fleeting-a:1", "fleeting-b:2"}, result)

				require.Equal(t, 0, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 2

				mock.EXPECT().
					Decrease(ctx, []string{"fleeting-a:1", "fleeting-b:2"}).
					Return([]string{"fleeting-a:1"}, fmt.Errorf("some error"))

				mock.EXPECT().
					Sanity(ctx).
					Return(nil)

				result, err := group.Decrease(ctx, []string{"fleeting-a:1", "fleeting-b:2"})
				require.Error(t, err)
				require.Equal(t, []string{"fleeting-a:1"}, result)

				require.Equal(t, 1, group.size)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.New(hclog.DefaultOptions),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				instance := &instancegroup.Instance{
					Name:   "fleeting-a",
					ID:     "1",
					Server: &scwInstance.Server{State: scwInstance.ServerStateRunning},
				}

				mock.EXPECT().
					List(ctx).
					Return([]*instancegroup.Instance{instance}, nil)

				updateIDs := make([]string, 0)
				err := group.Update(ctx, func(id string, state provider.State) {
					updateIDs = append(updateIDs, id)
				})
				require.NoError(t, err)
				require.Equal(t, []string{"fleeting-a:1"}, updateIDs)
				require.Equal(t, 1, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				mock.EXPECT().
					List(ctx).
					Return(nil, fmt.Errorf("some error"))

				err := group.Update(ctx, func(id string, state provider.State) {
					require.Fail(t, "update should not have been called")
				})
				require.Error(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.New(hclog.DefaultOptions),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestConnectInfo(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.settings.UseStaticCredentials = true
				group.settings.Key = []byte("-----BEGIN OPENSSH PRIVATE KEY-----")

				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(instancegroup.InstanceFromServer(&scwInstance.Server{
						ID:    "1",
						Name:  "fleeting-a",
						State: scwInstance.ServerStateRunning,
						Arch:  scwInstance.ArchX86_64,
						Image: &scwInstance.Image{
							ID:   "1fa98915-fc85-40d9-95ea-65a06ca8b396",
							Name: "Ubuntu 24.04",
							Arch: scwInstance.ArchX86_64,
						},
						PublicIPs: []*scwInstance.ServerIP{
							{
								Address: net.ParseIP("37.1.1.1"),
								Family:  scwInstance.ServerIPIPFamilyInet,
							},
						},
					}), nil)

				result, err := group.ConnectInfo(ctx, "fleeting-a:1")
				require.NoError(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						OS:                   "Ubuntu 24.04",
						Arch:                 "amd64",
						Protocol:             "ssh",
						UseStaticCredentials: true,
						Username:             "root",
						Key:                  []byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
					},
					ID:           "fleeting-a:1",
					ExternalAddr: "37.1.1.1",
				}, result)
			},
		},
		{name: "success ipv6",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.settings.UseStaticCredentials = true
				group.settings.Key = []byte("-----BEGIN OPENSSH PRIVATE KEY-----")

				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(instancegroup.InstanceFromServer(&scwInstance.Server{
						ID:    "1",
						Name:  "fleeting-a",
						State: scwInstance.ServerStateRunning,
						Arch:  scwInstance.ArchX86_64,
						Image: &scwInstance.Image{
							ID:   "1fa98915-fc85-40d9-95ea-65a06ca8b396",
							Name: "Ubuntu 24.04",
							Arch: scwInstance.ArchX86_64,
						},
						PublicIPs: []*scwInstance.ServerIP{
							{
								Address: net.ParseIP("2a01:4f8:1c19:1403::1"),
								Family:  scwInstance.ServerIPIPFamilyInet6,
							},
						},
					}), nil)

				result, err := group.ConnectInfo(ctx, "fleeting-a:1")
				require.NoError(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						OS:                   "Ubuntu 24.04",
						Arch:                 "amd64",
						Protocol:             "ssh",
						UseStaticCredentials: true,
						Username:             "root",
						Key:                  []byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
					},
					ID:           "fleeting-a:1",
					ExternalAddr: "2a01:4f8:1c19:1403::1",
				}, result)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(nil, fmt.Errorf("some error"))

				result, err := group.ConnectInfo(ctx, "fleeting-a:1")
				require.Error(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						Protocol: "ssh",
						Username: "root",
					},
				}, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.New(hclog.DefaultOptions),
				settings: provider.Settings{},
				group:    mock,
			}

			group.settings.Protocol = "ssh"
			group.settings.Username = "root"

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestShutdown(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, group *InstanceGroup, server *mockutil.Server)
	}{
		{name: "success",
			run: func(t *testing.T, group *InstanceGroup, server *mockutil.Server) {
				group.sshKey = &scwIam.SSHKey{ID: "1", Name: "fleeting", ProjectID: "e0660b65-9dce-4f25-854d-1161a1aa96a9"}

				server.Expect([]mockutil.Request{
					{
						Method: "DELETE", Path: "/iam/v1alpha1/ssh-keys/1",
						Status: 204,
					},
				})

				err := group.Shutdown(context.Background())
				require.NoError(t, err)
			},
		},
		{name: "failure",
			run: func(t *testing.T, group *InstanceGroup, server *mockutil.Server) {
				group.sshKey = &scwIam.SSHKey{ID: "1", Name: "fleeting", ProjectID: "e0660b65-9dce-4f25-854d-1161a1aa96a9"}

				server.Expect([]mockutil.Request{
					{
						Method: "DELETE", Path: "/iam/v1alpha1/ssh-keys/1",
						Status: 500,
					},
				})

				err := group.Shutdown(context.Background())
				require.EqualError(t, err, "scaleway-sdk-go: http error 500 Internal Server Error: 500 Internal Server Error")
			},
		},
		{name: "passthrough",
			run: func(t *testing.T, group *InstanceGroup, server *mockutil.Server) {
				server.Expect([]mockutil.Request{})

				err := group.Shutdown(context.Background())
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := mockutil.NewServer(t, nil)
			client := testutils.MakeTestClient(server.URL)

			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:            hclog.New(hclog.DefaultOptions),
				settings:       provider.Settings{},
				group:          mock,
				client:         client,
				iamClient:      scwIam.NewAPI(client),
				blockClient:    scwBlock.NewAPI(client),
				instanceClient: scwInstance.NewAPI(client),
			}

			testCase.run(t, group, server)
		})
	}
}
