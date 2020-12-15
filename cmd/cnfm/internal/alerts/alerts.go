package alerts

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/giantswarm/k8sportforward/v2"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/alertmanager/client"
	"github.com/prometheus/client_golang/api"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	alertManagerPort = 9093

	monitoringNamespace       = "monitoring"
	alertmanagerLabelSelector = "app=alertmanager"
)

// Config represents the configuration used to create a new alertmanager service.
type Config struct {
	Logger     micrologger.Logger
	RestConfig *rest.Config

	ClusterID string
}

type Service struct {
	logger     micrologger.Logger
	restConfig *rest.Config

	clusterID string
}

// New creates a new configured alerts service.
func New(config Config) (*Service, error) {
	// Settings.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be nil")
	}
	if config.RestConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.RestConfig must not be nil")
	}

	newService := &Service{
		// Settings.
		logger:     config.Logger,
		restConfig: config.RestConfig,
	}

	return newService, nil
}

// List alerts in alert manager
func (s *Service) ListAlerts(ctx context.Context, receiver string, silenced, inhibited bool) ([]*client.ExtendedAlert, error) {
	podName, err := GetAlertmanagerPodName(ctx, s.restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var fw *k8sportforward.Forwarder
	{
		c := k8sportforward.ForwarderConfig{
			RestConfig: s.restConfig,
		}

		fw, err = k8sportforward.NewForwarder(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	tunnel, err := fw.ForwardPort(monitoringNamespace, podName, alertManagerPort)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	defer tunnel.Close()

	var alertClient client.AlertAPI
	{
		var amClientConfig api.Config
		amClientConfig.Address = fmt.Sprintf("http://%s", tunnel.LocalAddress())
		amClientConfig.RoundTripper = http.DefaultTransport

		amClient, err := api.NewClient(amClientConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		alertClient = client.NewAlertAPI(amClient)
	}

	var filter = fmt.Sprintf("cluster_id=\"%s\"", s.clusterID)

	alerts, err := alertClient.List(ctx, filter, receiver, silenced, inhibited, true, false)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if receiver == "" {
		return alerts, nil
	}

	filtered := make([]*client.ExtendedAlert, 0, len(alerts))
	for _, alert := range alerts {
		receiversMatch := false
		for _, receiver := range alert.Receivers {
			receiversMatch, err = regexp.MatchString(receiver, receiver)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			if receiversMatch {
				filtered = append(filtered, alert)
				break
			}
		}
	}

	return filtered, nil
}

func GetAlertmanagerPodName(ctx context.Context, config *rest.Config) (string, error) {
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", microerror.Mask(err)
	}
	listOpts := v1.ListOptions{
		LabelSelector: alertmanagerLabelSelector,
	}

	pods, err := k8sClient.CoreV1().Pods(monitoringNamespace).List(ctx, listOpts)
	if err != nil {
		return "", microerror.Mask(err)
	}
	// only 1 pod of alertmanager should avaiable
	if len(pods.Items) != 1 {
		return "", microerror.Maskf(executionFailedError, fmt.Sprintf("wrong amount of alertmanager pods: %d", len(pods.Items)))
	}
	return pods.Items[0].Name, nil
}
