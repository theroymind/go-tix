package yamlstore

import (
	"slices"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
)

func ResolveReference(reference, prefix string, ids []string) (string, error) {
	if reference == "" {
		return "", backend.NoMatch(reference)
	}
	if slices.Contains(ids, reference) {
		return reference, nil
	}
	qualifiedPrefix := prefix + ticket.IDSeparator
	qualified := reference
	if !strings.HasPrefix(reference, qualifiedPrefix) {
		qualified = qualifiedPrefix + reference
	}
	if slices.Contains(ids, qualified) {
		return qualified, nil
	}
	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(id, qualified) {
			matches = append(matches, id)
		}
	}
	slices.Sort(matches)
	switch len(matches) {
	case 0:
		return "", backend.NoMatch(reference)
	case 1:
		return matches[0], nil
	default:
		return "", backend.Ambiguous(reference, matches)
	}
}
