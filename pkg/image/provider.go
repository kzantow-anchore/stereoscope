package image

import (
	"errors"
	"fmt"

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

func Detect(ctx runtime.ExecutionContext, userInput string, cfg ProviderConfig, providers []Provider) (*Image, error) {
	ctx.Log().Debugf("detect image: location=%s", userInput)

	var errs []error
	if len(providers) == 0 {
		return nil, fmt.Errorf("no image providers specified, no image will be detected")
	}
	for _, provider := range providers {
		img, err := provider.Provide(ctx, userInput, cfg)
		if err != nil {
			errs = append(errs, err)
		}
		if img != nil {
			err = img.Read()
			if err != nil {
				errs = append(errs, fmt.Errorf("could not read image: %w", err))
			}
			return img, errors.Join(errs...)
		}
	}
	return nil, fmt.Errorf("unable to detect input for '%s', err: %w", userInput, errors.Join(errs...))
}
