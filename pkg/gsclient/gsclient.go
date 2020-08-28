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
	Username string
	Password string
	Token    string
}

type Client struct {
	endpoint string
	username string
	password string
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

type ClusterEntry struct {
	ID             string `json:"id"`
	ReleaseVersion string `json:"release_version"`
}

const (
	CreationResultError            = "error"
	CreationResultCreatedWithError = "created-with-errors"
	DeletionResultScheduled        = "deletion scheduled"
)

func IsValidRelease(candidate string) bool {
	_, err := semver.NewVersion(candidate)
	return err == nil
}

func New(config Config) (*Client, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Token == "" && (config.Username == "" || config.Password == "") {
		return nil, microerror.Maskf(invalidConfigError, "%T.Token must not be empty if a %T.Username and %T.Password are not specified", config, config, config)
	}

	if config.Endpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Endpoint must not be empty", config)
	}

	client := Client{
		endpoint: config.Endpoint,
		username: config.Username,
		password: config.Password,
		token:    config.Token,
	}

	return &client, nil
}

func (c *Client) runWithGsctl(ctx context.Context, args ...string) (bytes.Buffer, error) {
	args = append(args, "--endpoint", c.endpoint)
	if c.token != "" {
		args = append(args, "--auth-token", c.token)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	gsctlCmd := exec.CommandContext(ctx, "gsctl", args...)
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

func (c *Client) Authenticate(ctx context.Context) error {
	_, err := c.runWithGsctl(ctx, "gsctl", "login", "--username", c.username, "--password", c.password)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (c *Client) CreateCluster(ctx context.Context, releaseVersion string) (string, error) {

	// TODO: extract and structure all these hardcoded values
	output, err := c.runWithGsctl(ctx, "--output=json", "create", "cluster", "--owner", "conformance-testing", "--name", releaseVersion, "--release", releaseVersion)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response CreationResponse
	err = json.Unmarshal(ignoreNonJSON(output.Bytes()), &response)
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
	output, err := c.runWithGsctl(ctx, "--output=json", "delete", "cluster", clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	var response DeletionResponse
	err = json.Unmarshal(ignoreNonJSON(output.Bytes()), &response)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Result != DeletionResultScheduled {
		return microerror.Maskf(clusterDeletionError, output.String())
	}

	return nil
}

func (c *Client) CreateKubeconfig(ctx context.Context, clusterID, kubeconfigPath string) error {
	_, err := c.runWithGsctl(ctx, "create", "kubeconfig", "--cluster", clusterID, "--certificate-organizations", "system:masters", "--force", "--self-contained", kubeconfigPath)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Client) ListClusters(ctx context.Context) ([]ClusterEntry, error) {
	output, err := c.runWithGsctl(ctx, "--output=json", "list", "clusters", "--show-deleting")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var response []ClusterEntry
	err = json.Unmarshal(ignoreNonJSON(output.Bytes()), &response)
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

func ignoreNonJSON(input []byte) []byte {
	curlyBracketIndex := strings.Index(string(input), "{")
	squareBracketIndex := strings.Index(string(input), "[")

	if curlyBracketIndex == -1 {
		// Input is not a JSON
		return nil
	}

	if squareBracketIndex == -1 {
		// No arrays, JSON starts with "{"
		return input[curlyBracketIndex : strings.LastIndex(string(input), "}")+1]
	}

	if curlyBracketIndex < squareBracketIndex {
		// JSON starts with "{"
		return input[curlyBracketIndex : strings.LastIndex(string(input), "}")+1]
	}

	// JSON starts with "["
	return input[squareBracketIndex : strings.LastIndex(string(input), "]")+1]
}
