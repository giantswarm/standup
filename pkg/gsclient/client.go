package gsclient

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
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
		if _, ok := err.(*exec.ExitError); ok {
			// Command started successfully and failed -> we want to parse the output JSON for more info
			return stdout, nil
		}
		return stdout, microerror.Mask(err)
	}

	return stdout, nil
}
