package image

import (
	"context"

	"github.com/anchore/stereoscope/pkg/file"
)

// ProviderConfig is the uber-configuration containing everything needed by stereoscope image providers
type ProviderConfig struct {
	Registry           RegistryOptions
	AdditionalMetadata []AdditionalMetadata
	Platform           *Platform
	TempDirGenerator   *file.TempDirGenerator
}

// Provider is an abstraction for any object that provides image objects (e.g. the docker daemon API, a tar file of
// an OCI image, podman varlink API, etc.).
type Provider interface {
	Provide(ctx context.Context, userInput string, cfg ProviderConfig) (*Image, error)
}

type ProviderFunc func(ctx context.Context, userInput string, cfg ProviderConfig) (*Image, error)
