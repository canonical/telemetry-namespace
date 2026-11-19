package topic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/canonical/telemetry-namespace/match"
)

// ErrInvalidFilter is returned for an invalid wire topic filter.
var ErrInvalidFilter = errors.New("invalid wire filter")

// Filter is a wire topic filter in one tenant's namespace, such as a rule
// topic, a connector filter or a remote subscription:
// <tenant>/<root>/<levels...>.
//
// The tenant and the root are literal. The other levels may contain
// wildcards; literal device IDs, group slugs and publishers must be valid.
type Filter struct {
	// Tenant is the tenant owning the namespace.
	Tenant string
	// Root is level 1: "devices", "groups" or "_system".
	Root string
	// Levels are the levels after Root. There is at least one.
	Levels []string
}

// ParseFilter parses and validates a wire topic filter. A topic name is also
// a valid filter. Shared subscriptions are rejected.
func ParseFilter(filter string) (Filter, error) {
	levels := strings.Split(filter, match.Separator)
	if len(levels) < 3 {
		return Filter{}, fmt.Errorf("%w %q: needs a tenant, a root and at least one more level", ErrInvalidFilter, filter)
	}
	f := Filter{Tenant: levels[0], Root: levels[1], Levels: levels[2:]}
	if err := f.Validate(); err != nil {
		return Filter{}, err
	}
	return f, nil
}

// Subtree returns the filter of everything addressed to a device or group:
// <tenant>/<kind>/<target>/#. The target must be literal.
func Subtree(tenant string, kind Kind, target string) (Filter, error) {
	if err := ValidateID(target); err != nil {
		return Filter{}, fmt.Errorf("%w: target: %w", ErrInvalidFilter, err)
	}
	return validatedFilter(Filter{Tenant: tenant, Root: string(kind), Levels: []string{target, match.MultiLevel}})
}

// SystemSubtree returns the filter of the platform data of a device or group:
// <tenant>/_system/<kind>/<target>/#. The target must be literal.
func SystemSubtree(tenant string, kind Kind, target string) (Filter, error) {
	if err := ValidateID(target); err != nil {
		return Filter{}, fmt.Errorf("%w: target: %w", ErrInvalidFilter, err)
	}
	return validatedFilter(Filter{Tenant: tenant, Root: SystemSegment, Levels: []string{string(kind), target, match.MultiLevel}})
}

// String returns the wire filter. It doesn't validate f.
func (f Filter) String() string {
	return strings.Join(append([]string{f.Tenant, f.Root}, f.Levels...), match.Separator)
}

// Validate checks f.
func (f Filter) Validate() error {
	if err := match.ValidateFilter(f.String()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFilter, err)
	}
	if err := ValidateID(f.Tenant); err != nil {
		return fmt.Errorf("%w: tenant: %w", ErrInvalidFilter, err)
	}
	if len(f.Levels) == 0 {
		return fmt.Errorf("%w: needs at least one level after the root", ErrInvalidFilter)
	}
	switch f.Root {
	case string(Devices), string(Groups):
		return f.validateLevels(ValidateID, ValidatePublisher)
	case SystemSegment:
		return f.validateLevels(validateKind, ValidateID)
	default:
		return fmt.Errorf("%w: root %q must be %q, %q or %q", ErrInvalidFilter, f.Root, Devices, Groups, SystemSegment)
	}
}

// Target returns the kind and the target the filter is about, when both are
// literal: <tenant>/<kind>/<target>/... or <tenant>/_system/<kind>/<target>/....
func (f Filter) Target() (kind Kind, target string, ok bool) {
	levels := f.Levels
	switch f.Root {
	case string(Devices), string(Groups):
		kind = Kind(f.Root)
	case SystemSegment:
		if len(levels) < 2 || isWildcardLevel(levels[0]) {
			return "", "", false
		}
		kind, levels = Kind(levels[0]), levels[1:]
	default:
		return "", "", false
	}
	if len(levels) == 0 || isWildcardLevel(levels[0]) {
		return "", "", false
	}
	return kind, levels[0], true
}

// validateLevels applies one check per leading level; later levels are free.
func (f Filter) validateLevels(checks ...func(string) error) error {
	for i, check := range checks {
		if i == len(f.Levels) {
			break
		}
		if isWildcardLevel(f.Levels[i]) {
			continue
		}
		if err := check(f.Levels[i]); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidFilter, err)
		}
	}
	return nil
}

func validateKind(level string) error {
	if level != string(Devices) && level != string(Groups) {
		return fmt.Errorf("kind %q must be %q or %q", level, Devices, Groups)
	}
	return nil
}

func isWildcardLevel(level string) bool {
	return level == match.SingleLevel || level == match.MultiLevel
}

func validatedFilter(f Filter) (Filter, error) {
	if err := f.Validate(); err != nil {
		return Filter{}, err
	}
	return f, nil
}
