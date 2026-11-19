// Package topic builds, parses and validates Telemetry wire topics (TM102):
//
//	<tenant>/devices/<deviceID>/<publisher>/<local topic>   snap data addressed to a device
//	<tenant>/groups/<groupID>/<publisher>/<local topic>     snap data addressed to a group
//	<tenant>/_system/devices/<deviceID>/<item>              platform data about a device
//	<tenant>/_system/groups/<groupID>/<item>                platform data about a group
//
// There is no leading '/'. Level 1 tells the subtrees apart, so that
// "<tenant>/devices/+/..." never matches group or platform traffic.
package topic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/canonical/telemetry-namespace/match"
)

// Kind is the type of target a topic is about: a device or a group.
type Kind string

const (
	// Devices is the kind of topics about a device.
	Devices Kind = "devices"

	// Groups is the kind of topics about a group.
	Groups Kind = "groups"
)

const (
	// SystemSegment is level 1 of the platform subtree.
	SystemSegment = "_system"

	// ItemConnection is the platform item holding a device's liveness.
	ItemConnection = "connection"

	// ItemGroups is the platform item holding a device's group membership.
	ItemGroups = "groups"

	// HTTPResponseSegment is level 0 of the local topic of HTTP-over-MQTT
	// responses: <tenant>/devices/<deviceID>/<publisher>/_http-response/<correlation>.
	HTTPResponseSegment = "_http-response"
)

// ErrInvalidTopic is returned for an invalid wire topic.
var ErrInvalidTopic = errors.New("invalid wire topic")

// Topic is a wire topic name.
type Topic struct {
	// Tenant is the tenant owning the topic.
	Tenant string
	// System is true for the platform subtree <tenant>/_system/....
	System bool
	// Kind says whether Target is a device ID or a group slug.
	Kind Kind
	// Target is the device ID or group slug.
	Target string
	// Publisher is the publisher space. It is empty for platform topics.
	Publisher string
	// Local is the local topic, or the item for platform topics. It may
	// contain several levels.
	Local string
}

// Parse parses and validates a wire topic name.
func Parse(name string) (Topic, error) {
	if err := match.ValidateName(name); err != nil {
		return Topic{}, fmt.Errorf("%w: %w", ErrInvalidTopic, err)
	}
	levels := strings.SplitN(name, match.Separator, 5)
	if len(levels) < 5 {
		return Topic{}, fmt.Errorf("%w %q: too few levels", ErrInvalidTopic, name)
	}
	var t Topic
	if levels[1] == SystemSegment {
		t = Topic{Tenant: levels[0], System: true, Kind: Kind(levels[2]), Target: levels[3], Local: levels[4]}
	} else {
		t = Topic{Tenant: levels[0], Kind: Kind(levels[1]), Target: levels[2], Publisher: levels[3], Local: levels[4]}
	}
	if err := t.Validate(); err != nil {
		return Topic{}, err
	}
	return t, nil
}

// NewDevice returns the topic <tenant>/devices/<device>/<publisher>/<local>.
func NewDevice(tenant, device, publisher, local string) (Topic, error) {
	return validated(Topic{Tenant: tenant, Kind: Devices, Target: device, Publisher: publisher, Local: local})
}

// NewGroup returns the topic <tenant>/groups/<group>/<publisher>/<local>.
func NewGroup(tenant, group, publisher, local string) (Topic, error) {
	return validated(Topic{Tenant: tenant, Kind: Groups, Target: group, Publisher: publisher, Local: local})
}

// NewSystem returns the platform topic <tenant>/_system/<kind>/<target>/<item>.
func NewSystem(tenant string, kind Kind, target, item string) (Topic, error) {
	return validated(Topic{Tenant: tenant, System: true, Kind: kind, Target: target, Local: item})
}

// DeviceConnection returns the liveness topic of a device:
// <tenant>/_system/devices/<device>/connection.
func DeviceConnection(tenant, device string) (Topic, error) {
	return NewSystem(tenant, Devices, device, ItemConnection)
}

// DeviceMembership returns the group membership topic of a device:
// <tenant>/_system/devices/<device>/groups.
func DeviceMembership(tenant, device string) (Topic, error) {
	return NewSystem(tenant, Devices, device, ItemGroups)
}

// HTTPResponse returns the topic of an HTTP-over-MQTT response:
// <tenant>/devices/<device>/<publisher>/_http-response/<correlation>. The
// correlation must be a single non-empty level.
func HTTPResponse(tenant, device, publisher, correlation string) (Topic, error) {
	if correlation == "" || strings.Contains(correlation, match.Separator) {
		return Topic{}, fmt.Errorf("%w: correlation %q must be a single non-empty level", ErrInvalidTopic, correlation)
	}
	return NewDevice(tenant, device, publisher, HTTPResponseSegment+match.Separator+correlation)
}

// String returns the wire topic. It doesn't validate t.
func (t Topic) String() string {
	if t.System {
		return strings.Join([]string{t.Tenant, SystemSegment, string(t.Kind), t.Target, t.Local}, match.Separator)
	}
	return strings.Join([]string{t.Tenant, string(t.Kind), t.Target, t.Publisher, t.Local}, match.Separator)
}

// Validate checks every field of t and that t.String() is a valid MQTT
// topic name. The local topic is not checked against the snap rules of
// ValidateLocalTopic, because platform components also publish under it
// (e.g. HTTP responses).
func (t Topic) Validate() error {
	if err := ValidateID(t.Tenant); err != nil {
		return fmt.Errorf("%w: tenant: %w", ErrInvalidTopic, err)
	}
	if t.Kind != Devices && t.Kind != Groups {
		return fmt.Errorf("%w: kind %q must be %q or %q", ErrInvalidTopic, t.Kind, Devices, Groups)
	}
	if err := ValidateID(t.Target); err != nil {
		return fmt.Errorf("%w: target: %w", ErrInvalidTopic, err)
	}
	if t.System {
		if t.Publisher != "" {
			return fmt.Errorf("%w: platform topics have no publisher", ErrInvalidTopic)
		}
	} else if err := ValidatePublisher(t.Publisher); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTopic, err)
	}
	if t.Local == "" {
		return fmt.Errorf("%w: empty local topic", ErrInvalidTopic)
	}
	if err := match.ValidateName(t.String()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTopic, err)
	}
	return nil
}

// HTTPResponseCorrelation returns the correlation of an HTTP-over-MQTT
// response topic. ok is false if t is not one.
func (t Topic) HTTPResponseCorrelation() (correlation string, ok bool) {
	if t.System || t.Kind != Devices {
		return "", false
	}
	correlation, ok = strings.CutPrefix(t.Local, HTTPResponseSegment+match.Separator)
	if !ok || correlation == "" || strings.Contains(correlation, match.Separator) {
		return "", false
	}
	return correlation, true
}

func validated(t Topic) (Topic, error) {
	if err := t.Validate(); err != nil {
		return Topic{}, err
	}
	return t, nil
}
