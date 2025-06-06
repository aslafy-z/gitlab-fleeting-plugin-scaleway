package testutils

import (
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func MakeTestClient(endpoint string) *scw.Client {
	opts := []scw.ClientOption{
		scw.WithAPIURL(endpoint),
		scw.WithDefaultProjectID("e0660b65-9dce-4f25-854d-1161a1aa96a9"),
		scw.WithDefaultZone(scw.Zone("fr-par-1")),
	}

	client, _ := scw.NewClient(opts...)
	return client
}
