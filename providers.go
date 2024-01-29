package stereoscope

import (
	"context"
	"fmt"
	"path"
	goRuntime "runtime"
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
	"github.com/anchore/stereoscope/runtime"
	"github.com/anchore/stereoscope/tagged"
)

const (
	FileTag     = "file"
	DirTag      = "dir"
	DaemonTag   = "daemon"
	PullTag     = "pull"
	RegistryTag = "registry"
)

func ImageProviders() tagged.Values[image.Provider] {
	return tagged.Values[image.Provider]{
		// file providers
		provide(image.DockerTarballSource, dockerTarballProvider, FileTag),
		provide(image.OciTarballSource, ociTarballProvider, FileTag),
		provide(image.OciDirectorySource, ociDirectoryProvider, FileTag, DirTag),
		provide(image.SingularitySource, singularityProvider, FileTag),

		// daemon providers
		provide(image.DockerDaemonSource, dockerDaemonProvider, DaemonTag, PullTag),
		provide(image.PodmanDaemonSource, podmanDaemonProvider, DaemonTag, PullTag),
		provide(image.ContainerdDaemonSource, containerdDaemonProvider, DaemonTag, PullTag),

		// registry providers
		provide(image.OciRegistrySource, ociRegistryProvider, RegistryTag, PullTag),
	}
}

//nolint:dupl
func dockerDaemonProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}
	// verify that the Docker daemon is accessible before assuming we can use it
	c, err := docker.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
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

	provider, err := dockerProvider.NewProviderFromDaemon(userInput, ctx, c, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

//nolint:dupl
func podmanDaemonProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}

	c, err := podman.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
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

	provider, err := dockerProvider.NewProviderFromDaemon(userInput, ctx, c, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func containerdDaemonProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}

	c, err := containerd.GetClient()
	if err != nil {
		return nil, err
	}

	c2, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
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

	provider, err := containerdProvider.NewProviderFromDaemon(userInput, ctx, c, containerd.Namespace(), cfg.Registry, cfg.Platform)
	if err != nil {
		return nil, err
	}

	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func dockerTarballProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.DockerTarballSource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not a Docker archive file: %w", err)
	}
	err = detectTarEntry(localFs, filePath, "manifest.json")
	if err != nil {
		return nil, err
	}
	provider := dockerProvider.NewProviderFromTarball(filePath, ctx)
	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func ociTarballProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.OciTarballSource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not an OCI archive file: %w", err)
	}
	err = detectTarEntry(localFs, filePath, "oci-layout")
	if err != nil {
		return nil, err
	}
	provider := ociProvider.NewProviderFromTarball(filePath, ctx)
	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func ociDirectoryProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.OciDirectorySource, userInput, cfg)
	if !exists || !isDir {
		return nil, fmt.Errorf("not an OCI directory: %w", err)
	}
	//  check for known oci-layout as an indication this is an oci directory
	if _, err := localFs.Stat(path.Join(filePath, "oci-layout")); err != nil {
		return nil, err
	}
	provider := ociProvider.NewProviderFromPath(userInput, ctx)
	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func singularityProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	filePath, exists, isDir, localFs, err := detectLocalFile(image.SingularitySource, userInput, cfg)
	if !exists || isDir {
		return nil, fmt.Errorf("not a Singularity archive: %w", err)
	}

	f, err := localFs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file=%s: %w", userInput, err)
	}
	ctx.RegisterCleanup(f.Close)

	// Check for Singularity container.
	fi, err := sif.LoadContainer(f, sif.OptLoadWithCloseOnUnload(false))
	if err != nil {
		return nil, err
	}
	err = fi.UnloadContainer()
	if err == nil {
		return nil, err
	}
	provider := sifProvider.NewProviderFromPath(filePath, ctx)
	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

func ociRegistryProvider(ctx runtime.ExecutionContext, userInput string, cfg image.ProviderConfig) (*image.Image, error) {
	if err := ensureRegistryReference(userInput); err != nil {
		return nil, err
	}
	defaultPlatformIfNil(&cfg)
	provider := ociProvider.NewProviderFromRegistry(userInput, ctx, cfg.Registry, cfg.Platform)
	return provider.Provide(ctx.Context(), cfg.AdditionalMetadata...)
}

// ensureRegistryReference takes a string and indicates if it conforms to a container image reference.
func ensureRegistryReference(imageSpec string) error {
	// note: strict validation requires there to be a default registry (e.g. docker.io) which we cannot assume will be provided
	// we only want to validate the bare minimum number of image specification features, not exhaustive.
	_, err := name.ParseReference(imageSpec, name.WeakValidation)
	return err
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
		p, err := image.NewPlatform(fmt.Sprintf("linux/%s", goRuntime.GOARCH))
		if err == nil {
			cfg.Platform = p
		}
	}
}

type stereoscopeProvider struct {
	name    string
	provide image.ProviderFunc
}

func (p stereoscopeProvider) Provide(ctx runtime.ExecutionContext, userInput string, config image.ProviderConfig) (*image.Image, error) {
	return p.provide(ctx, userInput, config)
}

func (p stereoscopeProvider) String() string {
	return p.Name()
}

func (p stereoscopeProvider) Name() string {
	return p.name
}

var _ image.Provider = (*stereoscopeProvider)(nil)

// provide names and tags a provider func to be used in the set of all providers
func provide(name image.Source, providerFunc image.ProviderFunc, tags ...string) tagged.Value[image.Provider] {
	return tagged.New[image.Provider](stereoscopeProvider{name, providerFunc}, append([]string{name}, tags...)...)
}
