package cluster

import (
	"bytes"
	"context"
	"text/template"

	"github.com/giantswarm/apiextensions/v2/pkg/id"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/standup/cmd/create/cluster/templates/azure"
	"github.com/giantswarm/standup/pkg/key"
)

func (r *runner) createCapzCluster(ctx context.Context, ctrl client.Client, organization string) (string, error) {
	var err error

	data := struct {
		KubernetesVersion  string
		Namespace          string
		Owner              string
		Version            string
		StorageAccountType string
		ClusterDescription string
		ClusterName        string

		ControlPlaneVMSize string

		NodePoolDescription string
		NodePoolMaxSize     string
		NodePoolMinSize     string
		NodePoolName        string
		NodePoolReplicas    string
		NodePoolVMSize      string
	}{
		KubernetesVersion:  "v1.19.8",
		Namespace:          key.OrganizationNamespaceFromName(organization),
		Owner:              organization,
		Version:            "20.0.0",
		StorageAccountType: "Premium_LRS",
		ClusterDescription: "e2e Test cluster",
		ClusterName:        id.Generate(),

		ControlPlaneVMSize: "Standard_D4s_v3",

		NodePoolDescription: "np1",
		NodePoolMaxSize:     "3",
		NodePoolMinSize:     "10",
		NodePoolName:        id.Generate(),
		NodePoolReplicas:    "3",
		NodePoolVMSize:      "Standard_D4s_v3",
	}

	var compiled []string
	for _, tmpl := range azure.GetTemplates() {
		var tpl bytes.Buffer
		t := template.Must(template.New("cluster.yaml").Parse(tmpl))
		err = t.Execute(&tpl, data)
		if err != nil {
			return "", microerror.Mask(err)
		}

		compiled = append(compiled, tpl.String())
	}

	err = r.applyTemplates(ctx, compiled, ctrl)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return data.ClusterName, nil
}
