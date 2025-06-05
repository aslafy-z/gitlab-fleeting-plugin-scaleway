package scaleway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"go.uber.org/mock/gomock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/instancegroup"
	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/testutils"
	scwIam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

func TestUploadSSHPublicKey(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server)
	}{
		{
			name: "existing fingerprint",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Path:   "/iam/v1alpha1/ssh-keys",
						Status: 200,
						JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{&sshKey}, TotalCount: 1},
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, "1", result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
		{
			name: "new",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Path:   "/iam/v1alpha1/ssh-keys",
						Status: 200,
						JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{&sshKey}, TotalCount: 1},
					},
					{
						Method: "POST", Path: "/iam/v1alpha1/ssh-keys",
						Want: func(t *testing.T, r *http.Request) {
							body, err := io.ReadAll(r.Body)
							require.NoError(t, err)

							publicKey, err := json.Marshal(sshKey.PublicKey)
							require.NoError(t, err)

							require.JSONEq(t, fmt.Sprintf(`{
								"name": "fleeting",
								"project_id": "dummy",
								"public_key": %s
							}`, publicKey), string(body))
						},
						Status: 200,
						JSON:   scwIam.SSHKey(sshKey),
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, "1", result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
		{
			name: "new with existing name",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Path:   "/iam/v1alpha1/ssh-keys",
						Status: 200,
						JSON:   scwIam.ListSSHKeysResponse{SSHKeys: []*scwIam.SSHKey{&sshKey}, TotalCount: 1},
					},
					{
						Method: "DELETE", Path: "/iam/v1alpha1/ssh-keys/1",
						Status: 204,
					},
					{
						Method: "POST", Path: "/iam/v1alpha1/ssh-keys",
						Want: func(t *testing.T, r *http.Request) {
							body, err := io.ReadAll(r.Body)
							require.NoError(t, err)

							publicKey, err := json.Marshal(sshKey.PublicKey)
							require.NoError(t, err)

							require.JSONEq(t, fmt.Sprintf(`{
								"name": "fleeting",
								"project_id": "dummy",
								"public_key": %s
							}`, publicKey), string(body))
						},
						Status: 200,
						JSON:   scwIam.SSHKey(sshKey),
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, int64(1), result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)

			server := mockutil.NewServer(t, nil)
			client := testutils.MakeTestClient(server.URL)

			group := &InstanceGroup{
				Name:           "fleeting",
				log:            hclog.New(hclog.DefaultOptions),
				settings:       provider.Settings{},
				group:          mock,
				client:         client,
				iamClient:      scwIam.NewAPI(client),
				instanceClient: scwInstance.NewAPI(client),
			}

			testCase.run(t, ctx, group, server)
		})
	}
}
