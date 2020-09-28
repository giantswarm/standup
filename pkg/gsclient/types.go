package gsclient

type ClusterEntry struct {
	ID             string `json:"id"`
	ReleaseVersion string `json:"release_version"`
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
	Error     Error  `json:"error,omitempty"`
}

type Error struct {
	Kind string `json:"kind"`
}
