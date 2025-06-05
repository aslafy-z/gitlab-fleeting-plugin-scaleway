package main

import (
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"

	scaleway "github.com/aslafy-z/gitlab-fleeting-plugin-scaleway"
)

func main() {
	plugin.Main(&scaleway.InstanceGroup{}, scaleway.Version)
}
