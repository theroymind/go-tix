package ticket

import (
	"net/url"
	"slices"
)

func IsValidExternalURL(text string) bool {
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	isWebScheme := slices.Contains(externalURLSchemes, parsed.Scheme)
	return isWebScheme && parsed.Host != ""
}
