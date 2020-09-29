package gsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/standup/pkg/key"
)

func (c *Client) CreateCluster(ctx context.Context, releaseVersion string) (string, error) {
	err := c.authenticate(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// TODO: extract and structure all these hardcoded values
	output, err := c.runWithGsctl(ctx, "--output=json", "create", "cluster", "--owner", "conformance-testing", "--name", releaseVersion, "--release", releaseVersion)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var response CreationResponse
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return "", microerror.Maskf(invalidResponseError, output.String())
	}

	if response.Result == key.ResultError {
		return "", microerror.Maskf(clusterCreationError, output.String())
	} else if response.Result == key.CreationResultCreatedWithError {
		return response.ClusterID, microerror.Maskf(clusterCreationError, output.String())
	}

	return response.ClusterID, nil
}

func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {
	err := c.authenticate(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO: extract and structure all these hardcoded values
	output, err := c.runWithGsctl(ctx, "--output=json", "delete", "cluster", clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	var response DeletionResponse
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return microerror.Maskf(invalidResponseError, output.String())
	}

	if response.Result != key.DeletionResultScheduled {
		if response.Error.Kind == "ClusterNotFoundError" {
			return microerror.Maskf(clusterNotFoundError, output.String())
		}
		return microerror.Maskf(clusterDeletionError, output.String())
	}

	return nil
}

func (c *Client) GetClusterReleaseVersion(ctx context.Context, clusterID string) (string, error) {
	err := c.authenticate(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

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

func (c *Client) ListClusters(ctx context.Context) ([]ClusterEntry, error) {
	err := c.authenticate(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	output, err := c.runWithGsctl(ctx, "--output=json", "list", "clusters", "--show-deleting")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var response []ClusterEntry
	err = json.Unmarshal(output.Bytes(), &response)
	if err != nil {
		return nil, microerror.Maskf(invalidResponseError, output.String())
	}

	return response, nil
}
