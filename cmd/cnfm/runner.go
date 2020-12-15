package cnfm

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/standup/cmd/cnfm/internal/alerts"
	"github.com/giantswarm/standup/pkg/key"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
	kubeconfigPath := key.KubeconfigPath(r.flag.Kubeconfig, r.flag.Provider)

	// Create REST config for the control plane
	var restConfig *rest.Config
	{
		var err error
		restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var controlPlaneK8sClient k8sclient.Interface
	{
		var err error
		controlPlaneK8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: restConfig,
		})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tenantK8sClient k8sclient.Interface
	{
		kubeconfig, err := ioutil.ReadFile(r.flag.Kubeconfig)
		if err != nil {
			return microerror.Mask(err)
		}

		tenantClientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
		if err != nil {
			return microerror.Mask(err)
		}

		restConfig, err := tenantClientConfig.ClientConfig()
		if err != nil {
			return microerror.Mask(err)
		}
		restConfig.Timeout = time.Second * 10

		tenantK8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: restConfig,
		})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		podList, err := tenantK8sClient.K8sClient().CoreV1().Pods("giantswarm").List(ctx, v1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=chart-operator",
		})
		if err != nil {
			return microerror.Mask(err)
		}
		if len(podList.Items) == 0 {
			r.logger.LogCtx(ctx, "message", "didn't find chart-operator pod in the giantswarm namespace")
			os.Exit(1)
		} else if podList.Items[0].Status.Phase != "Running" {
			r.logger.LogCtx(ctx, "message", "chart-operator is not running in the giantswarm namespace")
			os.Exit(1)
		}
	}

	r.logger.LogCtx(ctx, "message", "application tests passed")

	var alertsService *alerts.Service
	{
		var err error
		alertsService, err = alerts.New(alerts.Config{
			Logger:     r.logger,
			RestConfig: controlPlaneK8sClient.RESTConfig(),
			ClusterID:  r.flag.ClusterID,
		})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	alerts, err := alertsService.ListAlerts(ctx, "", true, true)
	if err != nil {
		return microerror.Mask(err)
	}

	var alertCount int
	for _, alert := range alerts {
		severity := string(alert.Alert.Labels["severity"])
		if severity == "page" || severity == "notify" {
			alertCount++
		}
	}

	if alertCount > 0 {
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("alertmanager is reporting %d active page/notify alerts about this cluster", alertCount))
		os.Exit(alertCount)
	}

	r.logger.LogCtx(ctx, "message", "alerts tests passed")

	return nil
}
