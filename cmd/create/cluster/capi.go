package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// mimics kubectl apply behaviour using client.Client.
func (r *runner) applyTemplates(ctx context.Context, templates []string, ctrl client.Client) error {
	for _, data := range templates {
		raw := map[string]interface{}{}
		err := yaml.Unmarshal([]byte(data), &raw)

		u := &unstructured.Unstructured{}
		u.Object = raw

		err = ctrl.Create(ctx, u)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
