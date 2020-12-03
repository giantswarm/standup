package gsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	OutputTypeJSON     = "json"
	FlagOutputTypeJSON = "--output=json"
)

type Config struct {
	Logger micrologger.Logger

	Endpoint string
	Password string
	Token    string
	Username string
}

type Client struct {
	endpoint string
	password string
	token    string
	username string
}

type GsctlCreateClusterOptions struct {
	OutputType string
	Owner      string
	Name       string
	Release    string
}

type GsctlDeleteClusterOptions struct {
	OutputType string
	ID         string
}

type GsctlListClustersOptions struct {
	OutputType   string
	ShowDeleting bool
}

func New(config Config) (*Client, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Endpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Endpoint is required", config)
	}

	if config.Username != "" || config.Password != "" {
		if config.Username == "" {
			return nil, microerror.Maskf(invalidConfigError, "%T.Username is required when password is given", config)
		} else if config.Password == "" {
			return nil, microerror.Maskf(invalidConfigError, "%T.Password is required when username is given", config)
		} else if config.Token == "" {
			return nil, microerror.Maskf(invalidConfigError, "%T.Token must not be provided when using username and password", config)
		}
	} else if config.Token == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Token is required if username and password are not provided", config)
	}

	client := Client{
		endpoint: config.Endpoint,
		username: config.Username,
		password: config.Password,
		token:    config.Token,
	}

	return &client, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	if c.token != "" {
		return nil
	}

	_, err := c.runWithGsctl(ctx, "gsctl", "login", "--username", c.username, "--password", c.password)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Client) gsctlCreateCluster(ctx context.Context, result interface{}, options GsctlCreateClusterOptions) ([]byte, error) {

	gsctlArgs := []string{
		"create", "cluster",
		"--owner", options.Owner,
		"--name", options.Name,
		"--release", options.Release,
	}

	var output []byte
	var err error

	if options.OutputType == OutputTypeJSON {
		gsctlArgs = append([]string{FlagOutputTypeJSON}, gsctlArgs...) // prepend the output format

		output, err = c.runWithGsctlJSON(ctx, &result, gsctlArgs...)
		if err != nil {
			return output, microerror.Mask(err)
		}
	} else {
		return nil, microerror.Maskf(invalidConfigError, "Output type %T.OutputType is not supported.", options)
	}

	return output, nil

}

func (c *Client) gsctlDeleteCluster(ctx context.Context, result interface{}, options GsctlDeleteClusterOptions) ([]byte, error) {

	gsctlArgs := []string{
		"delete", "cluster", options.ID,
	}

	var output []byte
	var err error

	if options.OutputType == OutputTypeJSON {
		gsctlArgs = append([]string{FlagOutputTypeJSON}, gsctlArgs...) // prepend the output format

		output, err = c.runWithGsctlJSON(ctx, &result, gsctlArgs...)
		if err != nil {
			return output, microerror.Mask(err)
		}
	} else {
		return nil, microerror.Maskf(invalidConfigError, "Output type %T.OutputType is not supported.", options)
	}

	return output, nil

}

func (c *Client) gsctlListClusters(ctx context.Context, result interface{}, options GsctlListClustersOptions) ([]byte, error) {

	gsctlArgs := []string{
		"list", "clusters",
	}

	var output []byte
	var err error

	if options.ShowDeleting {
		gsctlArgs = append(gsctlArgs, "--show-deleting")
	}

	if options.OutputType == OutputTypeJSON {
		gsctlArgs = append([]string{FlagOutputTypeJSON}, gsctlArgs...) // prepend the output format

		output, err = c.runWithGsctlJSON(ctx, &result, gsctlArgs...)
		if err != nil {
			return output, microerror.Mask(err)
		}
	} else {
		return nil, microerror.Maskf(invalidConfigError, "Output type %T.OutputType is not supported.", options)
	}

	return output, nil

}

func (c *Client) runWithGsctl(ctx context.Context, args ...string) ([]byte, error) {
	args = append(args, "--endpoint", c.endpoint)
	if c.token != "" {
		args = append(args, "--auth-token", c.token)
	}

	var stdout bytes.Buffer

	gsctlCmd := exec.CommandContext(ctx, "gsctl", args...)
	gsctlCmd.Stdout = &stdout

	err := gsctlCmd.Run()
	if err != nil {
		return stdout.Bytes(), microerror.Mask(err)
	}

	return stdout.Bytes(), nil
}

func (c *Client) runWithGsctlJSON(ctx context.Context, result interface{}, args ...string) ([]byte, error) {
	stdout, err := c.runWithGsctl(ctx, args...)
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		// Command started successfully and failed -> we want to parse the output JSON for more info
		// Fall through
	} else if err != nil {
		return stdout, microerror.Mask(err)
	}

	err = json.Unmarshal(stdout, &result)
	if err != nil {
		return stdout, microerror.Maskf(invalidResponseError, string(stdout))
	}

	return stdout, nil
}
