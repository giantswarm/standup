package gsclient

import (
	"context"

	"github.com/giantswarm/microerror"
)

func (c *Client) CreateKubeconfig(ctx context.Context, clusterID, kubeconfigPath string) error {
	err := c.authenticate(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.runWithGsctl(ctx, "create", "kubeconfig", "--cluster", clusterID, "--certificate-organizations", "system:masters", "--force", "--self-contained", kubeconfigPath)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
