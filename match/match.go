// Package match validates MQTT topic names and topic filters, and matches
// names against filters, following the MQTT 5 specification (section 4.7).
//
// It knows nothing about the Telemetry namespace: see package topic for that.
package match

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxLength is the maximum length of a topic name or filter, in bytes.
	MaxLength = 65535

	// Separator separates topic levels.
	Separator = "/"

	// SingleLevel is the single-level wildcard.
	SingleLevel = "+"

	// MultiLevel is the multi-level wildcard. It must be the last level.
	MultiLevel = "#"

	// SharePrefix starts a shared subscription filter: $share/<name>/<filter>.
	SharePrefix = "$share/"
)

var (
	// ErrInvalidName is returned for an invalid topic name.
	ErrInvalidName = errors.New("invalid topic name")

	// ErrInvalidFilter is returned for an invalid topic filter.
	ErrInvalidFilter = errors.New("invalid topic filter")
)

// ValidateName reports whether name is a valid topic name, as used in
// PUBLISH: non-empty, at most MaxLength bytes of valid UTF-8, with no U+0000
// and no wildcard.
func ValidateName(name string) error {
	if err := validateString(name); err != nil {
		return fmt.Errorf("%w %q: %w", ErrInvalidName, name, err)
	}
	if IsWildcard(name) {
		return fmt.Errorf("%w %q: wildcards are not allowed", ErrInvalidName, name)
	}
	return nil
}

// ValidateFilter reports whether filter is a valid topic filter, as used in
// SUBSCRIBE. A wildcard must occupy a whole level, and '#' must be the last
// level. A shared subscription ($share/<name>/<filter>) is accepted if its
// share name is non-empty and contains no wildcard.
func ValidateFilter(filter string) error {
	if err := validateString(filter); err != nil {
		return fmt.Errorf("%w %q: %w", ErrInvalidFilter, filter, err)
	}
	f := filter
	if strings.HasPrefix(filter, SharePrefix) {
		share, rest, ok := SplitShared(filter)
		if !ok || share == "" || IsWildcard(share) {
			return fmt.Errorf("%w %q: a shared subscription needs a share name without wildcards", ErrInvalidFilter, filter)
		}
		if rest == "" {
			return fmt.Errorf("%w %q: empty filter after the share name", ErrInvalidFilter, filter)
		}
		f = rest
	}
	levels := strings.Split(f, Separator)
	for i, level := range levels {
		switch {
		case level == MultiLevel && i != len(levels)-1:
			return fmt.Errorf("%w %q: %q must be the last level", ErrInvalidFilter, filter, MultiLevel)
		case level != SingleLevel && level != MultiLevel && IsWildcard(level):
			return fmt.Errorf("%w %q: a wildcard must occupy a whole level", ErrInvalidFilter, filter)
		}
	}
	return nil
}

// SplitShared splits a shared subscription filter $share/<name>/<filter> into
// its share name and filter. ok is false if filter is not a shared
// subscription.
func SplitShared(filter string) (share, rest string, ok bool) {
	after, found := strings.CutPrefix(filter, SharePrefix)
	if !found {
		return "", "", false
	}
	return strings.Cut(after, Separator)
}

// IsWildcard reports whether s contains a wildcard character.
func IsWildcard(s string) bool {
	return strings.ContainsAny(s, SingleLevel+MultiLevel)
}

// Matches reports whether the topic name matches filter. A $share/<name>/
// prefix on filter is ignored. Both arguments are assumed valid.
//
// As in MQTT, '#' also matches the parent level ("a/#" matches "a"), and a
// filter starting with a wildcard doesn't match a name starting with '$'.
func Matches(filter, name string) bool {
	filter = unshare(filter)
	if filter == name {
		return true
	}
	if strings.HasPrefix(name, "$") && startsWithWildcard(filter) {
		return false
	}
	fl := strings.Split(filter, Separator)
	nl := strings.Split(name, Separator)
	for i, level := range fl {
		if level == MultiLevel {
			return true
		}
		if i == len(nl) || (level != SingleLevel && level != nl[i]) {
			return false
		}
	}
	return len(fl) == len(nl)
}

// Contains reports whether every topic name matched by inner is also matched
// by outer. A $share/<name>/ prefix on either filter is ignored. Both
// arguments are assumed valid.
func Contains(outer, inner string) bool {
	outer, inner = unshare(outer), unshare(inner)
	if strings.HasPrefix(inner, "$") && startsWithWildcard(outer) {
		return false
	}
	ol := strings.Split(outer, Separator)
	il := strings.Split(inner, Separator)
	for i, level := range ol {
		if level == MultiLevel {
			return true
		}
		if i == len(il) || il[i] == MultiLevel {
			return false
		}
		if level != SingleLevel && level != il[i] {
			return false
		}
	}
	return len(ol) == len(il)
}

func unshare(filter string) string {
	if _, rest, ok := SplitShared(filter); ok {
		return rest
	}
	return filter
}

func startsWithWildcard(filter string) bool {
	return strings.HasPrefix(filter, SingleLevel) || strings.HasPrefix(filter, MultiLevel)
}

func validateString(s string) error {
	switch {
	case s == "":
		return errors.New("empty")
	case len(s) > MaxLength:
		return fmt.Errorf("longer than %d bytes", MaxLength)
	case !utf8.ValidString(s):
		return errors.New("not valid UTF-8")
	case strings.ContainsRune(s, 0):
		return errors.New("contains U+0000")
	}
	return nil
}
