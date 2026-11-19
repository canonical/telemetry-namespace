// Package selector parses topic selectors and resolves snap topics to wire
// topics (TM103).
//
// A selector is an optional prefix on a snap's local topic that addresses
// other devices or groups:
//
//	[devices: d1]/state               acme's topic "state" on device d1
//	[devices: [d1, d2]]/state         ... on devices d1 and d2
//	[devices: +]/state                ... on every device (subscribe only)
//	[groups: line-3]/cmd              acme's topic "cmd" for group line-3
//	[devices: d1, publisher: _system]/connection   liveness of d1 (subscribe only)
//
// Grammar (ABNF):
//
//	topic        = [ selector "/" ] local-topic
//	selector     = "[" pair *( sep pair ) "]"
//	pair         = key ":" *SP value
//	key          = "devices" / "groups" / "publisher"
//	value        = id / "+" / "[" id *( sep id ) "]"
//	sep          = "," *SP
//	id           = 1*( %x21-7E except "/" "+" "#" "[" "]" "," ":" )
package selector

import (
	"errors"
	"fmt"

	"github.com/canonical/telemetry-namespace/topic"
)

// Selector keys.
const (
	KeyDevices   = "devices"
	KeyGroups    = "groups"
	KeyPublisher = "publisher"
)

const (
	// AllDevices is the value of the devices key that means every device of
	// the tenant. It is only valid on subscribe.
	AllDevices = "+"

	// SystemPublisher is the publisher value that reads platform data. Only
	// device liveness is exposed: [devices: ..., publisher: _system]/connection.
	SystemPublisher = "_system"

	// MaxTargets is the maximum number of devices or groups in a selector,
	// after deduplication.
	MaxTargets = 16
)

var (
	// ErrSyntax is returned for a topic that starts with '[' but isn't a
	// valid selector (TM103 §1.3–§1.4).
	ErrSyntax = errors.New("invalid selector")

	// ErrTooManyTargets is returned when a selector has more than
	// MaxTargets distinct targets.
	ErrTooManyTargets = errors.New("too many selector targets")

	// ErrNotAllowed is returned for a valid selector that is not allowed in
	// this direction, such as "+" on publish or groups on subscribe.
	ErrNotAllowed = errors.New("selector not allowed")
)

// Selector is a parsed selector prefix. Exactly one of Devices and Groups is
// set.
type Selector struct {
	// Devices are the target device IDs, or [AllDevices].
	Devices []string
	// Groups are the target group slugs.
	Groups []string
	// Publisher is the value of the publisher key, empty if absent.
	Publisher string
}

// Parse splits a snap topic into its selector and the local topic after it.
// sel is nil if s doesn't start with '['. Parse checks the grammar and the
// rules of TM103 §1.3, deduplicates the targets and enforces MaxTargets. It
// doesn't check the local topic, nor the direction-specific rules: see
// Origin.Publish and Origin.Subscribe.
func Parse(s string) (sel *Selector, local string, err error) {
	if s == "" || s[0] != '[' {
		return nil, s, nil
	}
	p := parser{s: s, i: 1}
	sel, err = p.selector()
	if errors.Is(err, ErrTooManyTargets) {
		return nil, "", fmt.Errorf("%w %q: at most %d", ErrTooManyTargets, s, MaxTargets)
	}
	if err != nil {
		return nil, "", fmt.Errorf("%w %q: %w", ErrSyntax, s, err)
	}
	local = s[p.i:]
	if local == "" {
		return nil, "", fmt.Errorf("%w %q: the selector must be followed by a local topic", ErrSyntax, s)
	}
	return sel, local, nil
}

type parser struct {
	s string
	i int
}

func (p *parser) selector() (*Selector, error) {
	sel := &Selector{}
	seen := map[string]bool{}
	for {
		key, err := p.key()
		if err != nil {
			return nil, err
		}
		if seen[key] {
			return nil, fmt.Errorf("key %q appears more than once", key)
		}
		seen[key] = true
		p.spaces()
		values, isList, err := p.value()
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key, err)
		}
		if err := sel.set(key, values, isList); err != nil {
			return nil, err
		}
		if p.consume(',') {
			p.spaces()
			continue
		}
		if !p.consume(']') {
			return nil, fmt.Errorf("expected ',' or ']' at offset %d", p.i)
		}
		break
	}
	if !p.consume('/') {
		return nil, errors.New("the selector must be followed by '/'")
	}
	if err := sel.check(); err != nil {
		return nil, err
	}
	return sel, nil
}

func (p *parser) key() (string, error) {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= 'a' && p.s[p.i] <= 'z' {
		p.i++
	}
	key := p.s[start:p.i]
	if key != KeyDevices && key != KeyGroups && key != KeyPublisher {
		return "", fmt.Errorf("unknown key %q at offset %d", key, start)
	}
	if !p.consume(':') {
		return "", fmt.Errorf("expected ':' after key %q", key)
	}
	return key, nil
}

// value returns the values of a pair, deduplicated in order, and whether they
// were given as a list. It stops at the first value beyond MaxTargets, so a
// long topic costs linear time.
func (p *parser) value() ([]string, bool, error) {
	if p.consume('+') {
		return []string{AllDevices}, false, nil
	}
	if !p.consume('[') {
		id, err := p.id()
		return []string{id}, false, err
	}
	var ids []string
	seen := map[string]bool{}
	for {
		id, err := p.id()
		if err != nil {
			return nil, true, err
		}
		if !seen[id] {
			if len(ids) == MaxTargets {
				return nil, true, ErrTooManyTargets
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if p.consume(',') {
			p.spaces()
			continue
		}
		if !p.consume(']') {
			return nil, true, fmt.Errorf("expected ',' or ']' at offset %d", p.i)
		}
		return ids, true, nil
	}
}

func (p *parser) id() (string, error) {
	start := p.i
	for p.i < len(p.s) && isIDChar(p.s[p.i]) {
		p.i++
	}
	if p.i == start {
		return "", fmt.Errorf("expected an ID at offset %d", start)
	}
	return p.s[start:p.i], nil
}

func (p *parser) spaces() {
	for p.consume(' ') {
	}
}

func (p *parser) consume(c byte) bool {
	if p.i < len(p.s) && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}

func isIDChar(c byte) bool {
	if c < 0x21 || c > 0x7e {
		return false
	}
	switch c {
	case '/', '+', '#', '[', ']', ',', ':':
		return false
	}
	return true
}

func (sel *Selector) set(key string, values []string, isList bool) error {
	switch key {
	case KeyDevices:
		if values[0] != AllDevices {
			if err := validateIDs(values); err != nil {
				return err
			}
		}
		sel.Devices = values
	case KeyGroups:
		if values[0] == AllDevices {
			return fmt.Errorf("%q is only valid for %q", AllDevices, KeyDevices)
		}
		if err := validateIDs(values); err != nil {
			return err
		}
		sel.Groups = values
	case KeyPublisher:
		if isList || values[0] == AllDevices {
			return fmt.Errorf("%q takes a single publisher", KeyPublisher)
		}
		sel.Publisher = values[0]
	}
	return nil
}

func (sel *Selector) check() error {
	switch {
	case sel.Devices == nil && sel.Groups == nil:
		return fmt.Errorf("needs %q or %q", KeyDevices, KeyGroups)
	case sel.Devices != nil && sel.Groups != nil:
		return fmt.Errorf("%q and %q are mutually exclusive", KeyDevices, KeyGroups)
	case sel.Publisher == SystemPublisher && sel.Devices == nil:
		return fmt.Errorf("%q is only valid with %q", SystemPublisher, KeyDevices)
	}
	return nil
}

func validateIDs(ids []string) error {
	for _, id := range ids {
		if err := topic.ValidateID(id); err != nil {
			return err
		}
	}
	return nil
}
