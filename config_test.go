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
				AccessKey:    "SCWAXXXXXXXXXXXXXXXX",
				SecretKey:    "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization: "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:      "e0660b65-9dce-4f25-854d-1161a1aa96a9",
				Zone:         "fr-par-1",
				ServerTypes:  []string{"PRO2-XS", "PRO2-S"},
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
				AccessKey:    "SCWXXXXXXXXXXXXXXXXX",
				SecretKey:    "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization: "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:      "e0660b65-9dce-4f25-854d-1161a1aa96a9",
				Zone:         "fr-par-1",
				ServerTypes:  []string{"PRO2-XS", "PRO2-S"},
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
				AccessKey:    "SCWAXXXXXXXXXXXXXXXX",
				SecretKey:    "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization: "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:      "e0660b65-9dce-4f25-854d-1161a1aa96a9",
				Zone:         "fr-par-1",
				ServerTypes:  []string{"PRO2-XS", "PRO2-S"},
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
			name: "cloud init",
			group: InstanceGroup{
				Name:          "fleeting",
				AccessKey:     "SCWAXXXXXXXXXXXXXXXX",
				SecretKey:     "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization:  "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:       "e0660b65-9dce-4f25-854d-1161a1aa96a9",
				Zone:          "fr-par-1",
				ServerTypes:   []string{"PRO2-XS", "PRO2-S"},
				Image:         "1fa98915-fc85-40d9-95ea-65a06ca8b396",
				CloudInit:     "dummy",
				CloudInitFile: "dummy",
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "mutually exclusive plugin config provided: cloud_init, cloud_init_file", err.Error())
			},
		},
		{
			name: "volume size",
			group: InstanceGroup{
				Name:         "fleeting",
				AccessKey:    "SCWAXXXXXXXXXXXXXXXX",
				SecretKey:    "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2",
				Organization: "3ff93173-96c1-4f5f-8cf6-7441efc1070f",
				Project:      "e0660b65-9dce-4f25-854d-1161a1aa96a9",
				Zone:         "fr-par-1",
				ServerTypes:  []string{"PRO2-XS", "PRO2-S"},
				Image:        "1fa98915-fc85-40d9-95ea-65a06ca8b396",
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
			t.Setenv("SCW_SECRET_KEY", "")
			t.Setenv("SCW_ORGANIZATION_ID", "")
			t.Setenv("SCW_PROJECT_ID", "")
			t.Setenv("SCW_DEFAULT_ZONE", "")
			t.Setenv("SCW_API_URL", "")

			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			err := testCase.group.validate()
			testCase.assert(t, testCase.group, err)
		})
	}
}

func TestPopulateCloudInit(t *testing.T) {
	tmp := t.TempDir()
	cloudInitFile := path.Join(tmp, "user-data.yml")
	require.NoError(t, os.WriteFile(cloudInitFile, []byte("my-user-data"), 0644))

	group := InstanceGroup{
		Name:          "fleeting",
		CloudInitFile: cloudInitFile,
	}

	require.NoError(t, group.populate())
	require.Equal(t, "my-user-data", group.CloudInit)
}
