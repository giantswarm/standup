package key

import "fmt"

func KubeconfigPath(base, provider string) (path string) {
	return fmt.Sprintf("%s/%s", base, provider)
}
