package gsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type Config struct {
	Logger micrologger.Logger

	// Installations configuration
	Endpoint string
	Token    string
}

type Client struct {
	endpoint string
	token    string
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

type ClusterEntry struct {
	ID             string `json:"id"`
	ReleaseVersion string `json:"release_version"`
}

const (
	CreationResultCreated          = "created"
	CreationResultError            = "error"
	CreationResultCreatedWithError = "created-with-errors"
	DeletionResultScheduled        = "deletion scheduled"
)

var providers = []string{
	"aws",
}

func AllProviders() []string {
	return providers
}

func IsValidRelease(candidate string) bool {
	_, err := semver.NewVersion(candidate)
	return err == nil
}

func New(config Config) (*Client, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Token == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Token must not be empty", config)
	}

	if config.Endpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Endpoint must not be empty", config)
	}

	client := Client{
		endpoint: config.Endpoint,
		token:    config.Token,
	}

	return &client, nil
}

func (c *Client) runWithGsctl(args string) (bytes.Buffer, error) {
	argsArr := strings.Fields(args)

	// Add additional arguments from our client
	clientArgs := fmt.Sprintf("--endpoint=%s --auth-token=%s", c.endpoint, c.token)
	argsArr = append(argsArr, strings.Fields(clientArgs)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	gsctlCmd := exec.Command("gsctl", argsArr...)
	gsctlCmd.Stdout = &stdout
	gsctlCmd.Stderr = &stderr

	err := gsctlCmd.Run()
	if err != nil {
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		return stdout, microerror.Mask(err)
	}

	return stdout, nil
}

func (c *Client) CreateCluster(ctx context.Context, releaseVersion string) (string, error) {

	// TODO: extract and structure all these hardcoded values
	output, err := c.runWithGsctl("--output=json create cluster --owner conformance-testing --name " + releaseVersion + " --release " + releaseVersion)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response CreationResponse
	err = json.Unmarshal(ignoreWarnings(output.Bytes()), &response)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if response.Result == CreationResultError {
		return "", microerror.Maskf(clusterCreationError, output.String())
	} else if response.Result == CreationResultCreatedWithError {
		return response.ClusterID, microerror.Maskf(clusterCreationError, output.String())
	}

	return response.ClusterID, nil
}

func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {

	// TODO: extract and structure all these hardcoded values
	output, err := c.runWithGsctl(fmt.Sprintf("--output=json delete cluster %s", clusterID))
	if err != nil {
		return microerror.Mask(err)
	}

	var response DeletionResponse
	err = json.Unmarshal(ignoreWarnings(output.Bytes()), &response)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Result != DeletionResultScheduled {
		return microerror.Maskf(clusterDeletionError, output.String())
	}

	return nil
}

func (c *Client) GetKubeconfig(ctx context.Context, clusterID string) (string, error) {

	// TODO: extract and structure all these hardcoded values
	args := fmt.Sprintf("--output=json create kubeconfig --cluster=%s --certificate-organizations system:masters", clusterID)
	output, err := c.runWithGsctl(args)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response KubeconfigResponse
	err = json.Unmarshal(ignoreWarnings(output.Bytes()), &response)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// TODO: Do something useful here or remove it
	if response.Result != "ok" {
		fmt.Println("something went wrong creating kubeconfig")
		fmt.Println(output)
	}

	return response.Kubeconfig, nil
}

func (c *Client) ListClusters(ctx context.Context) ([]ClusterEntry, error) {
	// TODO: extract and structure all these hardcoded values
	args := "--output=json list clusters --show-deleting"
	output, err := c.runWithGsctl(args)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var response []ClusterEntry
	err = json.Unmarshal(ignoreWarnings(output.Bytes()), &response)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return response, nil
}

func (c *Client) GetClusterReleaseVersion(ctx context.Context, clusterID string) (string, error) {

	response, err := c.ListClusters(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	for _, cluster := range response {
		if cluster.ID == clusterID {
			// Have to add back the leading v in the release name
			return fmt.Sprintf("v%s", cluster.ReleaseVersion), nil
		}
	}

	return "", microerror.Maskf(clusterNotFoundError, fmt.Sprintf("cluster %s was not found", clusterID))
}

func ignoreWarnings(input []byte) []byte {
	return bytes.ReplaceAll(input, []byte("Warning:*\n"), []byte{})
}
