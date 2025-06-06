package instancegroup

import (
	"testing"

	"github.com/stretchr/testify/require"

	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

func TestInstanceFromServer(t *testing.T) {
	instance := InstanceFromServer(&scwInstance.Server{ID: "1", Name: "fleeting-a", State: scwInstance.ServerStateRunning})
	require.Equal(t, "1", instance.ID)
	require.Equal(t, "fleeting-a", instance.Name)
	require.NotNil(t, instance.Server)
	require.Equal(t, scwInstance.ServerStateRunning, instance.Server.State)
}

func TestInstanceFromIID(t *testing.T) {
	testCases := []struct {
		name     string
		iid      string
		instance *Instance
	}{
		{
			name:     "success",
			iid:      "fleeting-a:1",
			instance: &Instance{Name: "fleeting-a", ID: "1"},
		},
		{
			name:     "fail no separator",
			iid:      "fleeting-a-1",
			instance: nil,
		},
		{
			name:     "fail to many separator",
			iid:      "fleeting:a:1",
			instance: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			instance, err := InstanceFromIID(testCase.iid)
			if testCase.instance == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.instance, instance)
		})
	}
}

func TestInstanceIID(t *testing.T) {
	testCases := []struct {
		name     string
		instance *Instance
		iid      string
	}{
		{
			name:     "success",
			instance: &Instance{Name: "fleeting-a", ID: "1"},
			iid:      "fleeting-a:1",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.iid, testCase.instance.IID())
		})
	}
}
