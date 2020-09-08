package gsclient

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type Config struct {
	Logger micrologger.Logger

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
