package key

import "fmt"

const (
	ClusterOwnerName = "conformance-testing"
)

func KubeconfigPath(base, provider string) (path string) {
	return fmt.Sprintf("%s/%s", base, provider)
}
