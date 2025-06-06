package scaleway

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestProvisioning(t *testing.T) {
	if os.Getenv("SCW_ACCESS_KEY") == "" {
		t.Skip("mandatory environment variable SCW_ACCESS_KEY not set")
	}
	if os.Getenv("SCW_SECRET_KEY") == "" {
		t.Skip("mandatory environment variable SCW_SECRET_KEY not set")
	}
	if os.Getenv("SCW_ORGANIZATION_ID") == "" {
		t.Skip("mandatory environment variable SCW_ORGANIZATION_ID not set")
	}
	if os.Getenv("SCW_PROJECT_ID") == "" {
		t.Skip("mandatory environment variable SCW_PROJECT_ID not set")
	}

	ctx := context.Background()

	opts := []scw.ClientOption{
		scw.WithAuth(os.Getenv("SCW_ACCESS_KEY"), os.Getenv("SCW_SECRET_KEY")),
		scw.WithDefaultOrganizationID(os.Getenv("SCW_ORGANIZATION_ID")),
		scw.WithDefaultProjectID(os.Getenv("SCW_PROJECT_ID")),
		scw.WithDefaultZone(scw.Zone(os.Getenv("SCW_DEFAULT_ZONE"))),
		scw.WithUserAgent(fmt.Sprintf("%s (%s)", Version.Name, Version.String())),
	}

	if endpoint := os.Getenv("SCW_API_URL"); endpoint != "" {
		opts = append(opts, scw.WithAPIURL(endpoint))
	}

	client, err := scw.NewClient(opts...)
	require.NoError(t, err)

	instanceClient := scwInstance.NewAPI(client)

	pluginBinary := integration.BuildPluginBinary(t, "cmd/fleeting-plugin-scaleway", "fleeting-plugin-scaleway")

	t.Run("generated credentials", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					AccessKey:    os.Getenv("SCW_ACCESS_KEY"),
					SecretKey:    os.Getenv("SCW_SECRET_KEY"),
					Organization: os.Getenv("SCW_ORGANIZATION_ID"),
					Project:      os.Getenv("SCW_PROJECT_ID"),
					Endpoint:     os.Getenv("SCW_API_URL"),

					Zone:        "fr-par-1",
					ServerTypes: []string{"PRO2-XS", "PRO2-S"},
					Image:       "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, instanceClient, name)
	})

	t.Run("static credentials", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		sshPrivateKey, _, err := sshutil.GenerateKeyPair()
		require.NoError(t, err)

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					AccessKey:    os.Getenv("SCW_ACCESS_KEY"),
					SecretKey:    os.Getenv("SCW_SECRET_KEY"),
					Organization: os.Getenv("SCW_ORGANIZATION_ID"),
					Project:      os.Getenv("SCW_PROJECT_ID"),
					Endpoint:     os.Getenv("SCW_API_URL"),

					Zone:        "fr-par-1",
					ServerTypes: []string{"PRO2-XS", "PRO2-S"},
					Image:       "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,

					UseStaticCredentials: true,
					Username:             "root",
					Key:                  sshPrivateKey,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, instanceClient, name)
	})
}

// Ensure all servers were cleaned.
func ensureNoServers(t *testing.T, ctx context.Context, instanceClient *scwInstance.API, name string) { // nolint: revive
	t.Helper()

	result, err := instanceClient.ListServers(
		&scwInstance.ListServersRequest{
			Tags: []string{"instance-group=" + name},
		},
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	require.NoError(t, err)
	require.Equal(t, int(result.TotalCount), 0)
}
