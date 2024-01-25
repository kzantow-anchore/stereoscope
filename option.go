package stereoscope

import (
	"fmt"

	"github.com/anchore/stereoscope/pkg/image"
)

type Option func(*image.ProviderConfig) error

func applyOptions(cfg *image.ProviderConfig, options ...Option) error {
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(cfg); err != nil {
			return fmt.Errorf("unable to parse option: %w", err)
		}
	}
	return nil
}
