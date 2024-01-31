package stereoscope

import (
	"context"
	"fmt"
	"strings"

	"github.com/wagoodman/go-partybus"

	"github.com/anchore/go-logger"
	"github.com/anchore/stereoscope/internal/bus"
	"github.com/anchore/stereoscope/internal/log"
	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/image"
	"github.com/anchore/stereoscope/runtime"
)

var rootTempDirGenerator = file.NewTempDirGenerator("stereoscope")

func WithRegistryOptions(options image.RegistryOptions) Option {
	return func(c *image.ProviderConfig) error {
		c.Registry = options
		return nil
	}
}

func WithInsecureSkipTLSVerify() Option {
	return func(c *image.ProviderConfig) error {
		c.Registry.InsecureSkipTLSVerify = true
		return nil
	}
}

func WithInsecureAllowHTTP() Option {
	return func(c *image.ProviderConfig) error {
		c.Registry.InsecureUseHTTP = true
		return nil
	}
}

func WithCredentials(credentials ...image.RegistryCredentials) Option {
	return func(c *image.ProviderConfig) error {
		c.Registry.Credentials = append(c.Registry.Credentials, credentials...)
		return nil
	}
}

func WithAdditionalMetadata(metadata ...image.AdditionalMetadata) Option {
	return func(c *image.ProviderConfig) error {
		c.AdditionalMetadata = append(c.AdditionalMetadata, metadata...)
		return nil
	}
}

func WithPlatform(platform string) Option {
	return func(c *image.ProviderConfig) error {
		p, err := image.NewPlatform(platform)
		if err != nil {
			return err
		}
		c.Platform = p
		return nil
	}
}

// GetImage parses the user provided image string and provides an image object;
// note: the source where the image should be referenced from is automatically inferred.
func GetImage(ctx context.Context, imgStr string, options ...Option) (*image.Image, error) {
	return GetImageFromSource(ctx, imgStr, "", options...)
}

// GetImageFromSource returns an image from the explicitly provided source.
func GetImageFromSource(ctx context.Context, imgStr string, source image.Source, options ...Option) (*image.Image, error) {
	log.Debugf("image: source=%+v location=%+v", source, imgStr)

	// apply config options
	cfg := image.ProviderConfig{}
	if err := applyOptions(&cfg, options...); err != nil {
		return nil, err
	}

	// select image provider
	providers := ImageProviders()
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		// if no source is explicitly specified, look for a known scheme like docker:
		source, imgStr = ExtractProviderScheme(providers, imgStr)
	}
	if source != "" {
		providers = providers.Select(source)
	}
	if len(providers) < 1 {
		return nil, fmt.Errorf("unable to find image providers matching: '%s'", source)
	}

	return image.Detect(DefaultExecutionContext(ctx), imgStr, cfg, providers.Collect())
}

func SetLogger(logger logger.Logger) {
	log.Log = logger
}

func SetBus(b *partybus.Bus) {
	bus.SetPublisher(b)
}

type ExecutionContext = runtime.ExecutionContext

func DefaultExecutionContext(ctx ...context.Context) ExecutionContext {
	c := context.Background()
	switch len(ctx) {
	case 0:
	case 1:
		c = ctx[0]
	default:
		panic(fmt.Sprintf("may only specify one context, got: %v", ctx))
	}
	return runtime.NewExecutionContext(c, rootTempDirGenerator)
}

// Cleanup deletes all directories created by stereoscope calls.
// Deprecated: please use image.Image.Cleanup() over this.
func Cleanup() {
	if err := rootTempDirGenerator.Cleanup(); err != nil {
		log.Errorf("failed to cleanup tempdir root: %w", err)
	}
}
