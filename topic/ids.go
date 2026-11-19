package topic

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/canonical/telemetry-namespace/match"
)

const (
	// FleetGroup is the built-in group of every device of a tenant. Its slug
	// is reserved.
	FleetGroup = "fleet"

	// LocalPublisher is the publisher space of side-loaded snaps.
	LocalPublisher = "local"
)

var (
	// ErrInvalidID is returned for an invalid tenant ID, device ID or group
	// slug.
	ErrInvalidID = errors.New("invalid identifier")

	// ErrReservedID is returned when creating a group with a reserved slug.
	ErrReservedID = errors.New("reserved identifier")

	// ErrInvalidPublisher is returned for an invalid publisher segment.
	ErrInvalidPublisher = errors.New("invalid publisher")

	// ErrInvalidLocalTopic is returned for an invalid snap local topic or
	// filter.
	ErrInvalidLocalTopic = errors.New("invalid local topic")
)

// ValidateID checks a tenant ID, device ID or group slug (TM102 §2). It must
// be non-empty valid UTF-8, must not start with '_' or '$', and must not
// contain '/', '+', '#', whitespace or control characters. Dots are allowed.
//
// The built-in group "fleet" is a valid ID; use ValidateGroupSlug when
// creating a group.
func ValidateID(id string) error {
	if err := checkSegment(id); err != nil {
		return fmt.Errorf("%w %q: %w", ErrInvalidID, id, err)
	}
	return nil
}

// ValidateGroupSlug checks the slug of a group being created: it must be a
// valid ID and must not be the reserved "fleet".
func ValidateGroupSlug(slug string) error {
	if err := ValidateID(slug); err != nil {
		return err
	}
	if slug == FleetGroup {
		return fmt.Errorf("%w %q: the built-in group", ErrReservedID, slug)
	}
	return nil
}

// ValidatePublisher checks a publisher segment: the Snap Store publisher of a
// snap, or LocalPublisher. It follows the ValidateID rules; in particular a
// leading '_' is reserved for the platform.
func ValidatePublisher(publisher string) error {
	if err := checkSegment(publisher); err != nil {
		return fmt.Errorf("%w %q: %w", ErrInvalidPublisher, publisher, err)
	}
	return nil
}

// ValidateLocalTopic checks a topic name published by a snap (TM102 §2): a
// valid MQTT topic name whose level 0 doesn't start with '$' or '_'.
//
// A leading '[' is rejected: selectors are parsed by package selector, which
// then validates the rest of the topic with this function.
func ValidateLocalTopic(name string) error {
	if err := match.ValidateName(name); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLocalTopic, err)
	}
	return checkLocalLevel0(name)
}

// ValidateLocalFilter checks a topic filter subscribed to by a snap: a valid
// MQTT topic filter, not shared, whose level 0 doesn't start with '$', '_'
// or '['.
func ValidateLocalFilter(filter string) error {
	if err := match.ValidateFilter(filter); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLocalTopic, err)
	}
	return checkLocalLevel0(filter)
}

func checkLocalLevel0(topic string) error {
	switch topic[0] {
	case '$', '_':
		return fmt.Errorf("%w %q: level 0 must not start with %q", ErrInvalidLocalTopic, topic, topic[0])
	case '[':
		return fmt.Errorf("%w %q: not a valid selector", ErrInvalidLocalTopic, topic)
	}
	return nil
}

func checkSegment(s string) error {
	switch {
	case s == "":
		return errors.New("empty")
	case !utf8.ValidString(s):
		return errors.New("not valid UTF-8")
	case s[0] == '_' || s[0] == '$':
		return fmt.Errorf("must not start with %q", s[0])
	case strings.ContainsAny(s, match.Separator+match.SingleLevel+match.MultiLevel):
		return errors.New("must not contain '/', '+' or '#'")
	case strings.IndexFunc(s, isSpaceOrControl) >= 0:
		return errors.New("must not contain whitespace or control characters")
	}
	return nil
}

func isSpaceOrControl(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsControl(r)
}
