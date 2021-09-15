package azure

import (
	_ "embed"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed azure_cluster.yaml.tmpl
var azureCluster string

//go:embed azure_machine_template.yaml.tmpl
var azureMachineTemplate string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed machine_pool.yaml.tmpl
var machinePool string

//go:embed azure_machine_pool.yaml.tmpl
var azureMachinePool string

//go:embed kubeadm_config.yaml.tmpl
var kubeadmConfig string

func GetTemplates() []string {
	return []string{
		cluster,
		azureCluster,
		azureMachineTemplate,
		kubeadmControlPlane,
		machinePool,
		azureMachinePool,
		kubeadmConfig,
	}
}
