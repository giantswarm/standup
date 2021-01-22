package gsclient

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/standup/pkg/key"
)

func (c *Client) CreateCluster(ctx context.Context, organizationName, releaseVersion string) (string, error) {
	err := c.authenticate(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	createOptions := GsctlCreateClusterOptions{
		OutputType: OutputTypeJSON,
		Owner:      organizationName,
		Name:       releaseVersion,
		Release:    releaseVersion,
	}

	var response CreationResponse
	output, err := c.gsctlCreateCluster(ctx, &response, createOptions)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if response.Result == key.ResultError {
		return "", microerror.Maskf(clusterCreationError, string(output))
	} else if response.Result == key.CreationResultCreatedWithError {
		return response.ClusterID, microerror.Maskf(clusterCreationError, string(output))
	}

	return response.ClusterID, nil
}

func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {
	err := c.authenticate(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	deleteOptions := GsctlDeleteClusterOptions{
		OutputType: OutputTypeJSON,
		ID:         clusterID,
	}

	var response DeletionResponse
	output, err := c.gsctlDeleteCluster(ctx, &response, deleteOptions)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Result != key.DeletionResultScheduled {
		if response.Error.Kind == "ClusterNotFoundError" {
			return microerror.Maskf(clusterNotFoundError, string(output))
		}
		return microerror.Maskf(clusterDeletionError, string(output))
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

	listOptions := GsctlListClustersOptions{
		OutputType:   OutputTypeJSON,
		ShowDeleting: true,
	}

	var response []ClusterEntry
	_, err = c.gsctlListClusters(ctx, &response, listOptions)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return response, nil
}
