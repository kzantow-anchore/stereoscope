package oci

import (
	"fmt"
	goRuntime "runtime"

	"github.com/anchore/stereoscope/pkg/image"
)

// func Registry(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
//	provider := NewRegistryProvider(userInput, ctx, cfg.Registry, cfg.Platform)
//	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
//}
//
// func Archive(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
//	filePath, exists, isDir, localFs, err := docker.DetectLocalFile(image.OciTarballSource, userInput, cfg)
//	if !exists || isDir {
//		return nil, fmt.Errorf("not an OCI archive file: %w", err)
//	}
//	err = docker.DetectTarEntry(localFs, filePath, "oci-layout")
//	if err != nil {
//		return nil, err
//	}
//	provider := NewProviderFromTarball(filePath, ctx)
//	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
//}

// func Directory(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
//	filePath, exists, isDir, localFs, err := docker.DetectLocalFile(image.OciDirectorySource, userInput, cfg)
//	if !exists || !isDir {
//		return nil, fmt.Errorf("not an OCI directory: %w", err)
//	}
//	//  check for known oci-layout as an indication this is an oci directory
//	if _, err := localFs.Stat(path.Join(filePath, "oci-layout")); err != nil {
//		return nil, err
//	}
//	provider := NewProviderFromPath(userInput, ctx)
//	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
//}

// defaultPlatformIfNil sets the platform to use the host's architecture
// if no platform was specified. The OCI registry NewProvider uses "linux/amd64"
// as a hard-coded default platform, which has surprised customers
// running stereoscope on non-amd64 hosts. If platform is already
// set on the config, or the code can't generate a matching platform,
// do nothing.
func defaultPlatformIfNil(platform *image.Platform) *image.Platform {
	if platform == nil {
		p, err := image.NewPlatform(fmt.Sprintf("linux/%s", goRuntime.GOARCH))
		if err == nil {
			return p
		}
	}
	return platform
}
