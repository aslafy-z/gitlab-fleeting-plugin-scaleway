package scaleway

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name   string
		group  InstanceGroup
		env    map[string]string
		assert func(t *testing.T, group InstanceGroup, err error)
	}{
		{
			name: "valid",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "dummy",
				SecretKey:    "dummy",
				Organization: "dummy",
				Project:      "dummy",
				Zone:         "hel1",
				ServerType:   "PRO2-XS",
				Image:        "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				VolumeSize:   15,
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.NoError(t, err)
				assert.Equal(t, provider.ProtocolSSH, group.settings.Protocol)
				assert.Equal(t, "root", group.settings.Username)
			},
		},
		{
			name: "valid with env",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "dummy",
				SecretKey:    "dummy",
				Organization: "dummy",
				Project:      "dummy",
				Zone:         "hel1",
				ServerType:   "PRO2-XS",
				Image:        "1fa98915-fc85-40d9-95ea-65a06ca8b396",
			},
			env: map[string]string{
				"SCW_ACCESS_KEY": "value",
				"SCW_API_URL":    "value",
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "value", group.AccessKey)
				assert.Equal(t, "value", group.Endpoint)
			},
		},
		{
			name:  "empty",
			group: InstanceGroup{},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, `missing required plugin config: name
missing required plugin config: access_key
missing required plugin config: secret_key
missing required plugin config: organization
missing required plugin config: project
missing required plugin config: zone
missing required plugin config: server_type
missing required plugin config: image`, err.Error())
			},
		},
		{
			name: "winrm",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "dummy",
				SecretKey:    "dummy",
				Organization: "dummy",
				Project:      "dummy",
				Zone:         "hel1",
				ServerType:   "PRO2-XS",
				Image:        "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				settings: provider.Settings{
					ConnectorConfig: provider.ConnectorConfig{
						Protocol: "winrm",
					},
				},
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "unsupported connector config protocol: winrm", err.Error())
			},
		},
		{
			name: "user data",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "dummy",
				SecretKey:    "dummy",
				Organization: "dummy",
				Project:      "dummy",
				Zone:         "hel1",
				ServerType:   "PRO2-XS",
				UserData:     "dummy",
				UserDataFile: "dummy",
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "mutually exclusive plugin config provided: user_data, user_data_file", err.Error())
			},
		},
		{
			name: "volume size",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "dummy",
				SecretKey:    "dummy",
				Organization: "dummy",
				Project:      "dummy",
				Zone:         "hel1",
				ServerType:   "PRO2-XS",
				VolumeSize:   8,
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "invalid plugin config value: volume_size must be >= 10", err.Error())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("SCW_ACCESS_KEY", "")
			t.Setenv("SCW_API_URL", "")

			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			err := testCase.group.validate()
			testCase.assert(t, testCase.group, err)
		})
	}
}

func TestPopulateUserData(t *testing.T) {
	tmp := t.TempDir()
	userDataFile := path.Join(tmp, "user-data.yml")
	require.NoError(t, os.WriteFile(userDataFile, []byte("my-user-data"), 0644))

	group := InstanceGroup{
		Name:         "fleeting",
		UserDataFile: userDataFile,
	}

	require.NoError(t, group.populate())
	require.Equal(t, "my-user-data", group.UserData)
}
