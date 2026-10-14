package client

import (
	"slices"
	"strings"

	http "github.com/aarock1234/fphttp"
)

// mergeHeaders returns an independently owned snapshot with overrides applied.
// Empty values remove a field from this snapshot. The browser client retains
// the original suppression marker when constructing requests.
func mergeHeaders(base, overrides http.Header) http.Header {
	merged := base.Clone()
	if merged == nil {
		merged = make(http.Header)
	}
	for name, values := range overrides {
		for existing := range merged {
			if strings.EqualFold(existing, name) {
				delete(merged, existing)
			}
		}
		if len(values) == 0 || len(values) == 1 && values[0] == "" {
			continue
		}

		merged[http.CanonicalHeaderKey(name)] = slices.Clone(values)
	}

	return merged
}
