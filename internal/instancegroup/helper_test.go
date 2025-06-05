package instancegroup

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/testutils"
)

func mustUnmarshal[T any](t *testing.T, src io.ReadCloser, dest T) {
	body, err := io.ReadAll(src)
	require.NoError(t, err)

	err = json.Unmarshal(body, dest)
	require.NoError(t, err)
}

func setupInstanceGroup(t *testing.T, config Config, requests []mockutil.Request) *instanceGroup {
	t.Helper()

	requests = append(
		[]mockutil.Request{
			testutils.GetServerTypePRO2XSRequest,
			testutils.GetServerTypePRO2SRequest,
			testutils.GetImageUbuntu2404Request,
		},
		requests...,
	)

	server := httptest.NewServer(mockutil.Handler(t, requests))
	client := testutils.MakeTestClient(server.URL)

	log := hclog.New(hclog.DefaultOptions)

	group := &instanceGroup{
		name:           "fleeting",
		config:         config,
		log:            log,
		instanceClient: scwInstance.NewAPI(client),
		blockClient:    scwBlock.NewAPI(client),
	}
	group.randomNameFn = makeRandomNameFn(group.name)

	err := group.Init(context.Background())
	require.NoError(t, err)

	return group
}

func makeRandomNameFn(prefix string) func() string {
	offset := 96
	index := 0
	return func() string {
		index++
		return prefix + "-" + string(byte(offset+index))
	}
}

func TestMakeRandomNameFn(t *testing.T) {
	fn := makeRandomNameFn("fleeting")
	require.Equal(t, "fleeting-a", fn())
	require.Equal(t, "fleeting-b", fn())
	require.Equal(t, "fleeting-c", fn())
	require.Equal(t, "fleeting-d", fn())
}
