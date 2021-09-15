package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *runner) createCapaCluster(ctx context.Context, ctrl client.Client, organization string) (string, error) {
	return "", microerror.Mask(notImplementedError)
}
