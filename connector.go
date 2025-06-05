package scaleway

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"
	scwIam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func (g *InstanceGroup) UploadSSHPublicKey(ctx context.Context, pub []byte) (sshKey *scwIam.SSHKey, err error) {
	fingerprint, err := sshutil.GetPublicKeyFingerprint(pub)
	if err != nil {
		return nil, fmt.Errorf("could not get ssh key fingerprint: %w", err)
	}
	sshKeyRes, err := g.iamClient.ListSSHKeys(
		&scwIam.ListSSHKeysRequest{
			Disabled: scw.BoolPtr(false),
		},
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("could not list ssh keys: %w", err)
	}
	if sshKeyRes.TotalCount > 0 {
		for _, key := range sshKeyRes.SSHKeys {
			if key.Fingerprint == fingerprint {
				g.log.Info("using existing ssh key", "name", key.Name, "fingerprint", key.Fingerprint)
				return key, nil
			}
		}

		for _, key := range sshKeyRes.SSHKeys {
			if key.Name == g.Name {
				g.log.Warn("deleting existing ssh key", "name", key.Name, "fingerprint", key.Fingerprint)
				err = g.iamClient.DeleteSSHKey(
					&scwIam.DeleteSSHKeyRequest{
						SSHKeyID: key.ID,
					},
					scw.WithContext(ctx),
				)
				if err != nil {
					return nil, fmt.Errorf("could not delete existing ssh key: %w", err)
				}
			}
		}
	}

	g.log.Info("uploading ssh key", "name", g.Name, "fingerprint", fingerprint)
	sshKey, err = g.iamClient.CreateSSHKey(
		&scwIam.CreateSSHKeyRequest{
			Name:      g.Name,
			PublicKey: string(pub),
			ProjectID: g.Project,
		},
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("could not upload ssh key: %w", err)
	}

	return sshKey, nil
}
