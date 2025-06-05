package scaleway

import (
	"errors"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
)

func (g *InstanceGroup) validate() error {
	errs := []error{}

	// Defaults
	if g.settings.Protocol == "" {
		g.settings.Protocol = provider.ProtocolSSH
	}

	if g.settings.Username == "" {
		g.settings.Username = "root"
	}

	// Environment variables
	{
		value, err := envutil.LookupEnvWithFile("SCW_ACCESS_KEY")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.AccessKey = value
		}
	}
	{
		value, err := envutil.LookupEnvWithFile("SCW_SECRET_KEY")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.SecretKey = value
		}
	}
	{
		value, err := envutil.LookupEnvWithFile("SCW_ORGANIZATION_ID")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.Organization = value
		}
	}
	{
		value, err := envutil.LookupEnvWithFile("SCW_PROJECT_ID")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.Project = value
		}
	}
	{
		value, err := envutil.LookupEnvWithFile("SCW_DEFAULT_ZONE")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.Zone = value
		}
	}
	{
		value, err := envutil.LookupEnvWithFile("SCW_API_URL")
		if err != nil {
			errs = append(errs, err)
		} else if value != "" {
			g.Endpoint = value
		}
	}

	// Checks
	if g.Name == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: name"))
	}

	if g.AccessKey == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: access_key"))
	}

	if g.SecretKey == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: secret_key"))
	}

	if g.Organization == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: organization"))
	}

	if g.Project == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: project"))
	}

	if g.Zone == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: zone"))
	}

	if g.ServerType == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: server_type"))
	}

	if g.Image == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: image"))
	}

	if g.VolumeSize != 0 && g.VolumeSize < 10 {
		errs = append(errs, fmt.Errorf("invalid plugin config value: volume_size must be >= 10"))
	}

	if g.UserData != "" && g.UserDataFile != "" {
		errs = append(errs, fmt.Errorf("mutually exclusive plugin config provided: user_data, user_data_file"))
	}

	if g.settings.Protocol == provider.ProtocolWinRM {
		errs = append(errs, fmt.Errorf("unsupported connector config protocol: %s", g.settings.Protocol))
	}

	return errors.Join(errs...)
}

func (g *InstanceGroup) populate() error {
	if g.UserDataFile != "" {
		userData, err := os.ReadFile(g.UserDataFile)
		if err != nil {
			return fmt.Errorf("failed to read user data file: %w", err)
		}
		g.UserData = string(userData)
	}

	g.labels = map[string]string{
		"managed-by": Version.Name,
	}

	return nil
}
