package stereoscope

import (
	"context"
	"errors"
	"fmt"
	"path"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/sylabs/sif/v2/pkg/sif"

	"github.com/anchore/stereoscope/internal/containerd"
	"github.com/anchore/stereoscope/internal/docker"
	"github.com/anchore/stereoscope/internal/log"
	"github.com/anchore/stereoscope/internal/podman"
	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/image"
	containerdProvider "github.com/anchore/stereoscope/pkg/image/containerd"
	dockerProvider "github.com/anchore/stereoscope/pkg/image/docker"
	ociProvider "github.com/anchore/stereoscope/pkg/image/oci"
	sifProvider "github.com/anchore/stereoscope/pkg/image/sif"
	"github.com/anchore/stereoscope/tagged"
)

func ImageProviders() tagged.Values[image.Provider] {
	return tagged.Values[image.Provider]{
		// file providers
		provide(image.DockerTarballSource, DockerTarballProvider, "file"),
		provide(image.OciTarballSource, OciTarballProvider, "file"),
		provide(image.OciDirectorySource, OciDirectoryProvider, "file"),
		provide(image.SingularitySource, SingularityProvider, "file"),

		// daemon providers
		provide(image.DockerDaemonSource, DockerDaemonProvider, "daemon", "pull"),
		provide(image.PodmanDaemonSource, PodmanDaemonProvider, "daemon", "pull"),
		provide(image.ContainerdDaemonSource, ContainerdDaemonProvider, "daemon", "pull"),

		// registry providers
		provide(image.OciRegistrySource, OciRegistryProvider, "registry", "pull"),
	}
}

func DockerDaemonProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}
	// verify that the Docker daemon is accessible before assuming we can use it
	c, err := docker.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pong, err := c.Ping(c2)
	if pong.APIVersion == "" {
		return nil, fmt.Errorf("unable to get Docker API response: %w", err)
	}

	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("unable to close docker client: %+v", err)
		}
	}()

	provider, err := dockerProvider.NewProviderFromDaemon(userInput, cfg.TempDirGenerator, c, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func PodmanDaemonProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}

	c, err := podman.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pong, err := c.Ping(c2)
	if pong.APIVersion == "" {
		return nil, fmt.Errorf("unable to get Podman API response: %w", err)
	}

	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("unable to close docker client: %+v", err)
		}
	}()

	provider, err := dockerProvider.NewProviderFromDaemon(userInput, cfg.TempDirGenerator, c, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func ContainerdDaemonProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}

	c, err := containerd.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pong, err := c.Version(c2)
	if pong.Version == "" {
		return nil, fmt.Errorf("unable to get Containerd API response: %w", err)
	}

	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("unable to close docker client: %+v", err)
		}
	}()

	provider, err := containerdProvider.NewProviderFromDaemon(userInput, cfg.TempDirGenerator, c, containerd.Namespace(), cfg.Registry, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func DockerTarballProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.DockerTarballSource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not a Docker archive file: %w", err)
	}
	err = detectTarEntry(localFs, filePath, "manifest.json")
	if err != nil {
		return nil, err
	}
	provider := dockerProvider.NewProviderFromTarball(filePath, cfg.TempDirGenerator)
	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func OciTarballProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.OciTarballSource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not an OCI archive file: %w", err)
	}
	err = detectTarEntry(localFs, filePath, "oci-layout")
	if err != nil {
		return nil, err
	}
	provider := ociProvider.NewProviderFromTarball(filePath, cfg.TempDirGenerator)
	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func OciDirectoryProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.OciDirectorySource, userInput, cfg)
	if !exists || !isDir {
		return nil, fmt.Errorf("not an OCI directory: %w", err)
	}
	//  check for known oci-layout as an indication this is an oci directory
	if _, err := localFs.Stat(path.Join(filePath, "oci-layout")); err != nil {
		return nil, err
	}
	provider := ociProvider.NewProviderFromPath(userInput, cfg.TempDirGenerator)
	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func SingularityProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.SingularitySource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not a Singularity archive: %w", err)
	}

	f, err := localFs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file=%s: %w", userInput, err)
	}
	defer func() { _ = f.Close() }()

	// Check for Singularity container.
	fi, err := sif.LoadContainer(f, sif.OptLoadWithCloseOnUnload(false))
	if err != nil {
		return nil, err
	}
	err = fi.UnloadContainer()
	if err == nil {
		return nil, err
	}
	provider := sifProvider.NewProviderFromPath(filePath, cfg.TempDirGenerator)
	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

func OciRegistryProvider(ctx context.Context, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}
	defaultPlatformIfNil(&cfg)
	provider := ociProvider.NewProviderFromRegistry(userInput, cfg.TempDirGenerator, cfg.Registry, cfg.Platform)
	return provider.Provide(ctx, cfg.AdditionalMetadata...)
}

// ensureRegistryReference takes a string and indicates if it conforms to a container image reference.
func ensureRegistryReference(imageSpec string) error {
	// note: strict validation requires there to be a default registry (e.g. docker.io) which we cannot assume will be provided
	// we only want to validate the bare minimum number of image specification features, not exhaustive.
	_, err := name.ParseReference(imageSpec, name.WeakValidation)
	return err
}

// imageFromProviders takes a user string and determines the image source (e.g. the docker daemon, a tar file, etc.) returning the string subset representing the image (or nothing if it is unknown).
// note: parsing is done relative to the given string and environmental evidence (i.e. the given filesystem) to determine the actual source.
func imageFromProviders(ctx context.Context, userInput string, cfg image.ProviderConfig, providers ...image.Provider) (*image.Image, error) {
	var errs []error
	if len(providers) == 0 {
		providers = ImageProviders().Collect()
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

func detectLocalFile(provider string, userInput string, cfg image.ProviderConfig) (filePath string, exists bool, isDir bool, localFs afero.Fs, err error) {
	if cfg.Platform != nil {
		return "", false, false, nil,
			fmt.Errorf("specified platform=%q however image provider=%q does not support selecting platform", cfg.Platform.String(), provider)
	}

	filePath, err = homedir.Expand(userInput)
	if err != nil {
		return "", false, false, nil, err
	}

	localFs = afero.NewOsFs()

	pathStat, err := localFs.Stat(filePath)
	if err != nil {
		return "", false, false, nil, err
	}

	return filePath, true, pathStat.IsDir(), localFs, nil
}

// detectTarEntry attempts to open the archive and read a file from the provided path, returning any error encountered
func detectTarEntry(fs afero.Fs, archive, path string) error {
	f, err := fs.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = file.ReaderFromTar(f, path)
	return err
}

// defaultPlatformIfNil sets the platform to use the host's architecture
// if no platform was specified. The OCI registry provider uses "linux/amd64"
// as a hard-coded default platform, which has surprised customers
// running stereoscope on non-amd64 hosts. If platform is already
// set on the config, or the code can't generate a matching platform,
// do nothing.
func defaultPlatformIfNil(cfg *image.ProviderConfig) {
	if cfg.Platform == nil {
		p, err := image.NewPlatform(fmt.Sprintf("linux/%s", runtime.GOARCH))
		if err == nil {
			cfg.Platform = p
		}
	}
}
