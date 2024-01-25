package stereoscope

import (
	"context"

	"github.com/anchore/stereoscope/pkg/image"
	"github.com/anchore/stereoscope/tagged"
)

type stereoscopeProvider struct {
	name    string
	provide image.ProviderFunc
}

func (p stereoscopeProvider) Provide(ctx context.Context, userInput string, config image.ProviderConfig) (*image.Image, error) {
	return p.provide(ctx, userInput, config)
}

func (p stereoscopeProvider) String() string {
	return p.name
}

var _ image.Provider = (*stereoscopeProvider)(nil)

// provide names and tags a provider func to be used in the set of all providers
func provide(name image.Source, providerFunc image.ProviderFunc, tags ...string) tagged.Value[image.Provider] {
	return tagged.New[image.Provider](stereoscopeProvider{name, providerFunc}, append([]string{name}, tags...)...)
}
