package gsclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver/v3"
	gsclient "github.com/giantswarm/gsclientgen/v2/client"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

type Config struct {
	Logger micrologger.Logger

	// Installations configuration
	Email    string
	Endpoint string
	Password string
}

type Client struct {
	authWriter runtime.ClientAuthInfoWriter
	email      string
	endpoint   string
	password   string

	client *gsclient.Gsclientgen
}

// TODO: Use the gsctl type directly
type CreationResponse struct {
	ClusterID string `json:"id"`
	Result    string `json:"result"`
}

// TODO: Use the gsctl type directly
type DeletionResponse struct {
	ClusterID string `json:"id"`
	Result    string `json:"result"`
}

type KubeconfigResponse struct {
	Kubeconfig string `json:"kubeconfig"`
	Result     string `json:"result"`
}

const (
	CreationResultCreated          = "created"
	CreationResultCreatedWithError = "created-with-errors"
	DeletionResultScheduled        = "deletion scheduled"
)

var providers = []string{
	"aws",
}

func AllProviders() []string {
	return providers
}

func IsValidProvider(candidate string) bool {
	for _, provider := range providers {
		if candidate == provider {
			return true
		}
	}
	return false
}

func IsValidRelease(candidate string) bool {
	_, err := semver.NewVersion(candidate)
	return err == nil
}

func New(config Config) (*Client, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Email == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Email must not be empty", config)
	}

	if config.Password == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Password must not be empty", config)
	}

	if config.Endpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Endpoint must not be empty", config)
	}

	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, microerror.Maskf(invalidConfigError, "API endpoint URL %s could not be parsed", config.Endpoint)
	}

	tlsConfig := &tls.Config{}
	transport := httptransport.New(u.Host, "", []string{u.Scheme})
	transport.Transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	gsClient := gsclient.New(transport, strfmt.Default)

	client := Client{
		endpoint: config.Endpoint,
		email:    config.Email,
		password: config.Password,
		client:   gsClient,
	}

	return &client, nil
}

func runWithGsctl(args string) (bytes.Buffer, bytes.Buffer, error) {
	argsArr := strings.Fields(args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gsctlCmd := &exec.Cmd{
		Path:   "./gsctl", // TODO: don't hardcode this
		Args:   append([]string{"./gsctl"}, argsArr...),
		Stderr: &stderr,
		Stdout: &stdout,
	}

	err := gsctlCmd.Run()
	if err != nil {
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		return stdout, stderr, microerror.Mask(err)
	}

	return stdout, stderr, nil
}

func (c *Client) CreateCluster(ctx context.Context, releaseVersion string, provider string) (string, error) {

	// TODO: extract and structure all these hardcoded values
	output, stderr, err := runWithGsctl("--output=json create cluster --owner conformance-testing")
	// TODO: Handle stderr somehow
	if stderr.Len() > 0 {
		fmt.Println(stderr)
	}
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response CreationResponse
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if response.Result == CreationResultCreatedWithError {
		return response.ClusterID, microerror.Maskf(clusterCreationError, stderr.String())
	}

	return response.ClusterID, nil
}

func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {

	// TODO: extract and structure all these hardcoded values
	output, stderr, err := runWithGsctl(fmt.Sprintf("--output=json delete cluster %s", clusterID))
	// TODO: Handle stderr somehow
	if stderr.Len() > 0 {
		fmt.Println(stderr)
	}
	if err != nil {
		return microerror.Mask(err)
	}

	var response DeletionResponse
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Result != DeletionResultScheduled {
		return microerror.Maskf(clusterDeletionError, stderr.String())
	}

	return nil
}

func (c *Client) GetKubeconfig(ctx context.Context, clusterID string) (string, error) {

	// TODO: extract and structure all these hardcoded values
	args := fmt.Sprintf("--output=json create kubeconfig --cluster=%s --certificate-organizations system:masters", clusterID)
	output, stderr, err := runWithGsctl(args)
	// TODO: Handle stderr somehow
	if stderr.Len() > 0 {
		fmt.Println(output)
		fmt.Println(stderr)
	}
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response KubeconfigResponse
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// TODO: Do something useful here or remove it
	if response.Result != "ok" {
		fmt.Println("something went wrong creating kubeconfig")
		fmt.Println(output)
		fmt.Println(stderr)
	}

	return response.Kubeconfig, nil
}
