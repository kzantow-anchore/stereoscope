package stereoscope

import (
	"context"
	"strings"

	"github.com/anchore/stereoscope/pkg/image"
	"github.com/anchore/stereoscope/tagged"
)

// GetImageWithScheme replicates the previous behavior of GetImage with scheme parsing, i.e.
// this function attempts to parse any scheme: prefixes to treat them as an explicit provider name
//
// Deprecated: since it is now possible to select which providers to use, using schemes
// in the user input text is not necessary and should be avoided due to some ambiguity this introduces
func GetImageWithScheme(ctx context.Context, userInput string, opts ...Option) (*image.Image, error) {
	scheme, userInput := ExtractProviderScheme(ImageProviders(), userInput)
	if scheme != "" {
		return GetImageFromSource(ctx, userInput, scheme, opts...)
	}
	return GetImage(ctx, userInput, opts...)
}

// ExtractProviderScheme parses a string with any colon-delimited prefix and validates it against the set
// of known provider tags, returning a valid source name and input string to use for GetImageFromSource
//
// Deprecated: since it is now possible to select which providers to use, using schemes
// in the user input text is not necessary and should be avoided due to some ambiguity this introduces
func ExtractProviderScheme(providers tagged.Values[image.Provider], userInput string) (scheme, newInput string) {
	const SchemeSeparator = ":"

	parts := strings.SplitN(userInput, SchemeSeparator, 2)
	if len(parts) < 2 {
		return "", userInput
	}
	// the user may have provided a source hint (or this is a split from a path or docker image reference, we aren't certain yet)
	sourceHint := parts[0]
	sourceHint = strings.TrimSpace(strings.ToLower(sourceHint))
	// validate the hint against the possible tags
	if !providers.HasTag(sourceHint) {
		// did not have any matching tags, scheme is not a valid provider scheme
		return "", userInput
	}

	return sourceHint, parts[1]
}
