package image

import (
	"github.com/anchore/stereoscope/runtime"
)

// ProviderConfig is the uber-configuration containing everything needed by stereoscope image providers
type ProviderConfig struct {
	Registry           RegistryOptions
	AdditionalMetadata []AdditionalMetadata
	Platform           *Platform
}

// Provider is an abstraction for any object that provides image objects (e.g. the docker daemon API, a tar file of
// an OCI image, podman varlink API, etc.).
type Provider interface {
	Name() string
	Provide(ctx runtime.ExecutionContext, userInput string, cfg ProviderConfig) (*Image, error)
}

type ProviderFunc func(ctx runtime.ExecutionContext, userInput string, cfg ProviderConfig) (*Image, error)
