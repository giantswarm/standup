package wait

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

func labelsToSelector(labels map[string]string) string {
	selector := ""
	for key, value := range labels {
		selector += fmt.Sprintf("%s=%s,", key, value)
	}
	selector = selector[:len(selector)-1] // trim trailing comma
	return selector

}

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
	kubeconfig, err := ioutil.ReadFile(r.flag.Kubeconfig)
	if err != nil {
		return microerror.Mask(err)
	}

	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return microerror.Mask(err)
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return microerror.Mask(err)
	}
	restConfig.Timeout = time.Second * 10

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		r.logger.LogCtx(ctx, "message", "waiting for tenant cluster API to be reachable")

		o := func() error {
			_, err := k8sClient.CoreV1().Nodes().List(ctx, v1.ListOptions{})
			if tenant.IsAPINotAvailable(err) || IsServerError(err) {
				r.logger.LogCtx(ctx, "message", "API not yet available")
				return microerror.Mask(err)
			} else if err != nil {
				r.logger.LogCtx(ctx, "message", "unexpected error contacting API", "error", err)
				return microerror.Mask(err)
			}
			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "message", "API is now reachable")
	}

	{
		r.logger.LogCtx(ctx, "message", "waiting for nodes to be ready")

		o := func() error {
			nodes, err := k8sClient.CoreV1().Nodes().List(ctx, v1.ListOptions{})
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error listing nodes", "error", err)
				return microerror.Mask(err)
			}
			nodeCount := len(nodes.Items)
			readyCount := 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == "Ready" {
						if condition.Status == "True" {
							readyCount++
						}
						break
					}
				}
			}
			if nodeCount < r.flag.DesiredNodesCount {
				message := fmt.Sprintf("found %d registered nodes, waiting for at least %d", nodeCount, r.flag.DesiredNodesCount)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}
			if readyCount < nodeCount {
				message := fmt.Sprintf("%d out of %d nodes ready", readyCount, nodeCount)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}
			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "message", "nodes are ready")
	}

	{
		r.logger.LogCtx(ctx, "message", "waiting for CoreDNS to be ready")

		// Legacy GS clusters.
		targetLabels := map[string]string{
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            "CoreDNS",
		}

		// CAPI clusters.
		alternateTargetLabels := map[string]string{
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            "KubeDNS",
		}

		o := func() error {
			serviceLabelSelector := labelsToSelector(targetLabels)
			services, err := k8sClient.CoreV1().Services("kube-system").List(ctx, v1.ListOptions{
				LabelSelector: serviceLabelSelector,
			})
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error listing services", "error", err)
				return microerror.Mask(err)
			}
			if len(services.Items) == 0 {
				serviceLabelSelector := labelsToSelector(alternateTargetLabels)
				services, err = k8sClient.CoreV1().Services("kube-system").List(ctx, v1.ListOptions{
					LabelSelector: serviceLabelSelector,
				})
				if err != nil {
					r.logger.LogCtx(ctx, "message", "error listing services", "error", err)
					return microerror.Mask(err)
				}
			}

			if len(services.Items) == 0 {
				message := fmt.Sprintf("CoreDNS service not found using label selectors %#q and %#q", serviceLabelSelector, alternateTargetLabels)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}

			service := services.Items[0]
			podLabelSelector := labelsToSelector(service.Spec.Selector)
			pods, err := k8sClient.CoreV1().Pods("kube-system").List(ctx, v1.ListOptions{
				LabelSelector: podLabelSelector,
			})
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error listing CoreDNS pods", "error", err)
				return microerror.Mask(err)
			}
			if len(pods.Items) == 0 {
				message := fmt.Sprintf("CoreDNS pods not found using label selector %#q", podLabelSelector)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}

			for _, pod := range pods.Items {
				for _, container := range pod.Status.ContainerStatuses {
					if !container.Ready {
						message := fmt.Sprintf("CoreDNS pod container %#q not ready", container.Name)
						r.logger.LogCtx(ctx, "message", message)
						return microerror.Mask(notReadyError)
					}
				}
			}

			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "message", "CoreDNS is ready")
	}

	// We wait for external-dns for Azure and AWS. This prevents failures where
	// the CNCF suite is started before external-dns is functional.
	// This is not necessary on KVM as there is no external-dns app running.
	if r.flag.Provider != "kvm" {
		r.logger.LogCtx(ctx, "message", "waiting for external-dns to be ready")

		targetLabels := map[string]string{
			"giantswarm.io/service-type": "managed",
			"app.kubernetes.io/name":     "external-dns",
		}

		o := func() error {
			serviceLabelSelector := labelsToSelector(targetLabels)
			services, err := k8sClient.CoreV1().Services("kube-system").List(ctx, v1.ListOptions{
				LabelSelector: serviceLabelSelector,
			})
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error listing services", "error", err)
				return microerror.Mask(err)
			}
			if len(services.Items) == 0 {
				message := fmt.Sprintf("external-dns service not found using label selector %#q", serviceLabelSelector)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}

			service := services.Items[0]
			podLabelSelector := labelsToSelector(service.Spec.Selector)
			pods, err := k8sClient.CoreV1().Pods("kube-system").List(ctx, v1.ListOptions{
				LabelSelector: podLabelSelector,
			})
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error listing external-dns pods", "error", err)
				return microerror.Mask(err)
			}
			if len(pods.Items) == 0 {
				message := fmt.Sprintf("external-dns pods not found using label selector %#q", podLabelSelector)
				r.logger.LogCtx(ctx, "message", message)
				return microerror.Mask(notReadyError)
			}

			for _, pod := range pods.Items {
				for _, container := range pod.Status.ContainerStatuses {
					if !container.Ready {
						message := fmt.Sprintf("external-dns pod container %#q not ready", container.Name)
						r.logger.LogCtx(ctx, "message", message)
						return microerror.Mask(notReadyError)
					}
				}
			}

			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "message", "external-dns is ready")
	}

	return nil
}
