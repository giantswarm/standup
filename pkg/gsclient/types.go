package gsclient

type ClusterEntry struct {
	ID             string `json:"id"`
	ReleaseVersion string `json:"release_version"`
}

type CreationResponse struct {
	ClusterID string `json:"id"`
	Result    string `json:"result"`
}

type DeletionResponse struct {
	ClusterID string `json:"id"`
	Result    string `json:"result"`
	Error     Error  `json:"error,omitempty"`
}

type Error struct {
	Kind string `json:"kind"`
}
