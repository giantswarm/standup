package config

import (
	"os"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

type ProviderConfig struct {
	Endpoint string `json:"endpoint"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

func LoadProviderConfig(path string, provider string) (*ProviderConfig, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	providerConfigs := map[string]ProviderConfig{}
	err = yaml.UnmarshalStrict(configData, &providerConfigs)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	providerConfig, ok := providerConfigs[provider]
	if !ok {
		return nil, microerror.Maskf(invalidConfigError, "missing config for provider %#q", provider)
	}

	if providerConfig.Endpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "missing endpoint for provider %#q", provider)
	}
	if providerConfig.Token == "" && (providerConfig.Username == "" || providerConfig.Password == "") {
		return nil, microerror.Maskf(invalidConfigError, "missing token or username/password for provider %#q", provider)
	}

	return &providerConfig, nil
}
