package stereoscope

import (
	"context"
	"errors"
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
	if source != "" {
		providers = providers.Select(source)
	}
	if len(providers) < 1 {
		return nil, fmt.Errorf("unable to find image providers matching: '%s'", source)
	}

	return DetectImage(runtime.NewExecutionContext(ctx, rootTempDirGenerator), imgStr, DetectionConfig{
		imageProviderConfig: cfg,
		providers:           providers.Collect(),
	})
}

type DetectionConfig struct {
	imageProviderConfig image.ProviderConfig
	providers           []image.Provider
}

// Detect returns the first image found by providers
func DefaultDetectImage(userInput string) (*image.Image, error) {
	return DetectImage(DefaultExecutionContext(), userInput, DefaultDetectionConfig())
}

func DetectImage(ctx runtime.ExecutionContext, userInput string, cfg DetectionConfig) (*image.Image, error) {
	log.Debugf("detect image: location=%s", userInput)

	var errs []error
	if len(cfg.providers) == 0 {
		cfg.providers = ImageProviders().Collect()
	}
	for _, provider := range cfg.providers {
		img, err := provider.Provide(ctx, userInput, cfg.imageProviderConfig)
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

// Deprecated:
func SetLogger(logger logger.Logger) {
	log.Log = logger
}

// Deprecated:
func SetBus(b *partybus.Bus) {
	bus.SetPublisher(b)
}

func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{}
}

func DefaultExecutionContext() runtime.ExecutionContext {
	return NewExecutionContext(context.Background())
}

func NewExecutionContext(ctx context.Context) runtime.ExecutionContext {
	return runtime.NewExecutionContext(ctx, rootTempDirGenerator)
}

// Cleanup deletes all directories created by stereoscope calls.
// Deprecated: please use image.Image.Cleanup() over this.
func Cleanup() {
	if err := rootTempDirGenerator.Cleanup(); err != nil {
		log.Errorf("failed to cleanup tempdir root: %w", err)
	}
}
