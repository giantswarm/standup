package utils

// map of features (we're depending on standup) by provider
var providerFeatureSet = map[string][]string{
	"kvm":       {},
	"openstack": {},
	"azure":     {"external-dns"},
	"gcp":       {"external-dns"},
	"aws":       {"external-dns"},
}

func ProviderHasFeature(provider, feature string) bool {
	for _, n := range providerFeatureSet[provider] {
		if feature == n {
			return true
		}
	}
	return false
}
