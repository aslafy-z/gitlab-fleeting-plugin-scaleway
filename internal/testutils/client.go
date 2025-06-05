package testutils

import (
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func MakeTestClient(endpoint string) *scw.Client {
	opts := []scw.ClientOption{
		scw.WithAPIURL(endpoint),
	}

	client, _ := scw.NewClient(opts...)
	return client
}
