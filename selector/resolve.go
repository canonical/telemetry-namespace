package selector

import (
	"fmt"
	"slices"
	"strings"

	"github.com/canonical/telemetry-namespace/match"
	"github.com/canonical/telemetry-namespace/topic"
)

// Origin identifies the snap a topic comes from: its device and its
// publisher. A snap never leaves its publisher space.
type Origin struct {
	// Tenant is the tenant of the device.
	Tenant string
	// Device is the ID of the device the snap runs on.
	Device string
	// Publisher is the publisher of the snap.
	Publisher string
}

// Resolution is a snap topic resolved to wire topics.
type Resolution struct {
	// Selector is the parsed selector, nil if the topic had none.
	Selector *Selector
	// Local is the local topic or filter, without the selector.
	Local string
	// Wire are the wire topics to publish to, or the remote filters to
	// subscribe to. It is empty for a subscription without a selector.
	Wire []string
}

// Validate checks the tenant ID, the device ID and the publisher.
func (o Origin) Validate() error {
	if err := topic.ValidateID(o.Tenant); err != nil {
		return fmt.Errorf("tenant: %w", err)
	}
	if err := topic.ValidateID(o.Device); err != nil {
		return fmt.Errorf("device: %w", err)
	}
	return topic.ValidatePublisher(o.Publisher)
}

// Publish resolves the topic of a snap PUBLISH (TM103 §2). Without a
// selector, the wire topic is the snap's own device slot
// <tenant>/devices/<device>/<publisher>/<topic>. With a selector, there is one
// wire topic per target device or group, under the snap's publisher. The "+"
// value and the publisher key are not allowed.
func (o Origin) Publish(name string) (Resolution, error) {
	if err := o.Validate(); err != nil {
		return Resolution{}, err
	}
	sel, local, err := Parse(name)
	if err != nil {
		return Resolution{}, err
	}
	if err := topic.ValidateLocalTopic(local); err != nil {
		return Resolution{}, err
	}
	if sel == nil {
		return o.resolve(nil, local, topic.Devices, []string{o.Device})
	}
	switch {
	case sel.Publisher != "":
		return Resolution{}, fmt.Errorf("%w: %q is not allowed on publish", ErrNotAllowed, KeyPublisher)
	case slices.Contains(sel.Devices, AllDevices):
		return Resolution{}, fmt.Errorf("%w: %q is not allowed on publish", ErrNotAllowed, AllDevices)
	case sel.Groups != nil:
		return o.resolve(sel, local, topic.Groups, sel.Groups)
	default:
		return o.resolve(sel, local, topic.Devices, sel.Devices)
	}
}

// Subscribe resolves the filter of a snap SUBSCRIBE (TM103 §3). Without a
// selector, the subscription is local only and Wire is empty. With a
// selector, there is one remote filter per target device, under the snap's
// publisher, or under the platform subtree for liveness. Groups are reserved
// on subscribe.
func (o Origin) Subscribe(filter string) (Resolution, error) {
	if err := o.Validate(); err != nil {
		return Resolution{}, err
	}
	sel, local, err := Parse(filter)
	if err != nil {
		return Resolution{}, err
	}
	if sel == nil {
		if err := topic.ValidateLocalFilter(local); err != nil {
			return Resolution{}, err
		}
		return Resolution{Local: local}, nil
	}
	if sel.Groups != nil {
		return Resolution{}, fmt.Errorf("%w: %q is reserved on subscribe", ErrNotAllowed, KeyGroups)
	}
	switch sel.Publisher {
	case SystemPublisher:
		if local != topic.ItemConnection {
			return Resolution{}, fmt.Errorf("%w: %q only exposes %q", ErrNotAllowed, SystemPublisher, topic.ItemConnection)
		}
		return o.resolveFilters(sel, local, func(device string) topic.Filter {
			return topic.Filter{Tenant: o.Tenant, Root: topic.SystemSegment, Levels: []string{string(topic.Devices), device, local}}
		})
	case "", o.Publisher:
		if err := topic.ValidateLocalFilter(local); err != nil {
			return Resolution{}, err
		}
		levels := strings.Split(local, match.Separator)
		return o.resolveFilters(sel, local, func(device string) topic.Filter {
			return topic.Filter{Tenant: o.Tenant, Root: string(topic.Devices), Levels: append([]string{device, o.Publisher}, levels...)}
		})
	default:
		return Resolution{}, fmt.Errorf("%w: publisher %q: only %q or the snap's own publisher", ErrNotAllowed, sel.Publisher, SystemPublisher)
	}
}

func (o Origin) resolve(sel *Selector, local string, kind topic.Kind, targets []string) (Resolution, error) {
	wire := make([]string, 0, len(targets))
	for _, target := range targets {
		t := topic.Topic{Tenant: o.Tenant, Kind: kind, Target: target, Publisher: o.Publisher, Local: local}
		if err := t.Validate(); err != nil {
			return Resolution{}, err
		}
		wire = append(wire, t.String())
	}
	return Resolution{Selector: sel, Local: local, Wire: wire}, nil
}

func (o Origin) resolveFilters(sel *Selector, local string, build func(device string) topic.Filter) (Resolution, error) {
	wire := make([]string, 0, len(sel.Devices))
	for _, device := range sel.Devices {
		f := build(device)
		if err := f.Validate(); err != nil {
			return Resolution{}, err
		}
		wire = append(wire, f.String())
	}
	return Resolution{Selector: sel, Local: local, Wire: wire}, nil
}
