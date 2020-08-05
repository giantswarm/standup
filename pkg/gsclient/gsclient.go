package gsclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"

	"github.com/Masterminds/semver/v3"
	gsclient "github.com/giantswarm/gsclientgen/v2/client"
	"github.com/giantswarm/gsclientgen/v2/client/auth_tokens"
	"github.com/giantswarm/gsclientgen/v2/client/clusters"
	"github.com/giantswarm/gsclientgen/v2/client/key_pairs"
	"github.com/giantswarm/gsclientgen/v2/models"
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

func (c *Client) CreateCluster(ctx context.Context, releaseVersion string, provider string) (string, error) {
	if c.authWriter == nil {
		err := c.authorize()
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	switch provider {
	case "aws":
		params := clusters.NewAddClusterV5ParamsWithContext(ctx)
		params.Body.Name = "test"
		params.Body.ReleaseVersion = releaseVersion
		response, err := c.client.Clusters.AddClusterV5(params, c.authWriter)
		if err != nil {
			return "", microerror.Mask(err)
		}
		return response.Payload.ID, nil
	default:
		return "", errors.New("invalid provider")
	}
}

func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {
	if c.authWriter == nil {
		err := c.authorize()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	params := clusters.NewDeleteClusterParamsWithContext(ctx)
	params.ClusterID = clusterID
	_, err := c.client.Clusters.DeleteCluster(params, c.authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Client) GetKeypair(ctx context.Context, clusterID string) (*models.V4AddKeyPairResponse, error) {
	if c.authWriter == nil {
		err := c.authorize()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var response *key_pairs.AddKeyPairOK
	{
		var err error
		params := key_pairs.NewAddKeyPairParamsWithContext(ctx)
		params.ClusterID = clusterID
		response, err = c.client.KeyPairs.AddKeyPair(params, c.authWriter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return response.Payload, nil
}

func (c *Client) authorize() error {
	params := auth_tokens.NewCreateAuthTokenParams().WithBody(&models.V4CreateAuthTokenRequest{
		Email:          c.email,
		PasswordBase64: base64.StdEncoding.EncodeToString([]byte(c.password)),
	})
	response, err := c.client.AuthTokens.CreateAuthToken(params, nil)
	if err != nil {
		return microerror.Mask(err)
	}

	authHeader := "giantswarm " + response.Payload.AuthToken
	c.authWriter = httptransport.APIKeyAuth("Authorization", "header", authHeader)

	return nil
}
