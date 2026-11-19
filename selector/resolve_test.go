package selector_test

import (
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/selector"
	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var origin = selector.Origin{Tenant: "T", Device: "self", Publisher: "acme"}

func TestPublish(t *testing.T) {
	testCases := []struct {
		name         string
		topic        string
		wantSelector bool
		wantLocal    string
		wantWire     []string
	}{
		{
			name:      "own slot",
			topic:     "sensor/temp",
			wantLocal: "sensor/temp",
			wantWire:  []string{"T/devices/self/acme/sensor/temp"},
		},
		{
			name:         "one device",
			topic:        "[devices: d1]/cmd",
			wantSelector: true,
			wantLocal:    "cmd",
			wantWire:     []string{"T/devices/d1/acme/cmd"},
		},
		{
			name:         "own device through a selector",
			topic:        "[devices: self]/cmd",
			wantSelector: true,
			wantLocal:    "cmd",
			wantWire:     []string{"T/devices/self/acme/cmd"},
		},
		{
			name:         "device list",
			topic:        "[devices: [d1, d2, d1]]/cmd",
			wantSelector: true,
			wantLocal:    "cmd",
			wantWire:     []string{"T/devices/d1/acme/cmd", "T/devices/d2/acme/cmd"},
		},
		{
			name:         "group",
			topic:        "[groups: line-3]/cmd/reboot",
			wantSelector: true,
			wantLocal:    "cmd/reboot",
			wantWire:     []string{"T/groups/line-3/acme/cmd/reboot"},
		},
		{
			name:         "group list",
			topic:        "[groups: [g1, g2]]/cmd",
			wantSelector: true,
			wantLocal:    "cmd",
			wantWire:     []string{"T/groups/g1/acme/cmd", "T/groups/g2/acme/cmd"},
		},
		{
			name:         "fleet",
			topic:        "[groups: fleet]/config",
			wantSelector: true,
			wantLocal:    "config",
			wantWire:     []string{"T/groups/fleet/acme/config"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := origin.Publish(tc.topic)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSelector, got.Selector != nil)
			assert.Equal(t, tc.wantLocal, got.Local)
			assert.Equal(t, tc.wantWire, got.Wire)
		})
	}
}

func TestPublish_Invalid(t *testing.T) {
	testCases := []struct {
		name    string
		origin  selector.Origin
		topic   string
		wantErr error
	}{
		{name: "every device", origin: origin, topic: "[devices: +]/cmd", wantErr: selector.ErrNotAllowed},
		{name: "publisher key", origin: origin, topic: "[devices: d1, publisher: acme]/cmd", wantErr: selector.ErrNotAllowed},
		{name: "system publisher", origin: origin, topic: "[devices: d1, publisher: _system]/connection", wantErr: selector.ErrNotAllowed},
		{name: "malformed selector", origin: origin, topic: "[devices d1]/cmd", wantErr: selector.ErrSyntax},
		{name: "too many targets", origin: origin, topic: "[groups: " + list(17) + "]/cmd", wantErr: selector.ErrTooManyTargets},
		{name: "reserved local topic", origin: origin, topic: "_system/x", wantErr: topic.ErrInvalidLocalTopic},
		{name: "reserved local topic after a selector", origin: origin, topic: "[devices: d1]/$x", wantErr: topic.ErrInvalidLocalTopic},
		{name: "nested selector", origin: origin, topic: "[devices: d1]/[groups: g1]/x", wantErr: topic.ErrInvalidLocalTopic},
		{name: "wildcard", origin: origin, topic: "[groups: g1]/cmd/#", wantErr: topic.ErrInvalidLocalTopic},
		{name: "too long", origin: origin, topic: "[devices: d1]/" + strings.Repeat("a", 65530), wantErr: topic.ErrInvalidTopic},
		{name: "invalid tenant", origin: selector.Origin{Tenant: "", Device: "self", Publisher: "acme"}, topic: "x", wantErr: topic.ErrInvalidID},
		{name: "invalid device", origin: selector.Origin{Tenant: "T", Device: "a/b", Publisher: "acme"}, topic: "x", wantErr: topic.ErrInvalidID},
		{name: "invalid publisher", origin: selector.Origin{Tenant: "T", Device: "self", Publisher: "_system"}, topic: "x", wantErr: topic.ErrInvalidPublisher},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.origin.Publish(tc.topic)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestSubscribe(t *testing.T) {
	testCases := []struct {
		name         string
		filter       string
		wantSelector bool
		wantLocal    string
		wantWire     []string
	}{
		{name: "plain", filter: "cmd/#", wantLocal: "cmd/#", wantWire: nil},
		{
			name:         "one device",
			filter:       "[devices: d1]/state/#",
			wantSelector: true,
			wantLocal:    "state/#",
			wantWire:     []string{"T/devices/d1/acme/state/#"},
		},
		{
			name:         "device list",
			filter:       "[devices: [d1, d2]]/state/+",
			wantSelector: true,
			wantLocal:    "state/+",
			wantWire:     []string{"T/devices/d1/acme/state/+", "T/devices/d2/acme/state/+"},
		},
		{
			name:         "every device",
			filter:       "[devices: +]/state/#",
			wantSelector: true,
			wantLocal:    "state/#",
			wantWire:     []string{"T/devices/+/acme/state/#"},
		},
		{
			name:         "own publisher given explicitly",
			filter:       "[devices: d1, publisher: acme]/state",
			wantSelector: true,
			wantLocal:    "state",
			wantWire:     []string{"T/devices/d1/acme/state"},
		},
		{
			name:         "liveness of one device",
			filter:       "[devices: d1, publisher: _system]/connection",
			wantSelector: true,
			wantLocal:    "connection",
			wantWire:     []string{"T/_system/devices/d1/connection"},
		},
		{
			name:         "liveness of every device",
			filter:       "[devices: +, publisher: _system]/connection",
			wantSelector: true,
			wantLocal:    "connection",
			wantWire:     []string{"T/_system/devices/+/connection"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := origin.Subscribe(tc.filter)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSelector, got.Selector != nil)
			assert.Equal(t, tc.wantLocal, got.Local)
			assert.Equal(t, tc.wantWire, got.Wire)
		})
	}
}

func TestSubscribe_Invalid(t *testing.T) {
	testCases := []struct {
		name    string
		origin  selector.Origin
		filter  string
		wantErr error
	}{
		{name: "groups reserved", origin: origin, filter: "[groups: g1]/cmd", wantErr: selector.ErrNotAllowed},
		{name: "fleet reserved", origin: origin, filter: "[groups: fleet]/cmd", wantErr: selector.ErrNotAllowed},
		{name: "other publisher", origin: origin, filter: "[devices: d1, publisher: other]/state", wantErr: selector.ErrNotAllowed},
		{name: "other system item", origin: origin, filter: "[devices: d1, publisher: _system]/groups", wantErr: selector.ErrNotAllowed},
		{name: "system wildcard", origin: origin, filter: "[devices: d1, publisher: _system]/#", wantErr: selector.ErrNotAllowed},
		{name: "system subtree", origin: origin, filter: "[devices: d1, publisher: _system]/connection/x", wantErr: selector.ErrNotAllowed},
		{name: "malformed selector", origin: origin, filter: "[devices: [d1]/x", wantErr: selector.ErrSyntax},
		{name: "reserved local filter", origin: origin, filter: "$SYS/#", wantErr: topic.ErrInvalidLocalTopic},
		{name: "reserved local filter after a selector", origin: origin, filter: "[devices: d1]/_http-response/#", wantErr: topic.ErrInvalidLocalTopic},
		{name: "invalid local filter", origin: origin, filter: "[devices: d1]/a/#/b", wantErr: topic.ErrInvalidLocalTopic},
		{name: "too long", origin: origin, filter: "[devices: d1]/" + strings.Repeat("a", 65530), wantErr: topic.ErrInvalidFilter},
		{name: "invalid origin", origin: selector.Origin{Tenant: "T", Device: "", Publisher: "acme"}, filter: "x", wantErr: topic.ErrInvalidID},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.origin.Subscribe(tc.filter)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// Whatever a snap sends, the wire topics stay in its tenant and its
// publisher space (TM104 R5). The only exception is reading liveness.
func FuzzPublisherIsolation(f *testing.F) {
	for _, seed := range []string{
		"state",
		"[devices: d1]/x",
		"[devices: [d1, d2]]/x/#",
		"[devices: +]/x",
		"[groups: fleet]/x",
		"[devices: +, publisher: _system]/connection",
		"[devices: d1, publisher: other]/x",
		"[devices:d1,publisher:acme]/x",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if res, err := origin.Publish(s); err == nil {
			require.NotEmpty(t, res.Wire)
			for _, w := range res.Wire {
				tp, err := topic.Parse(w)
				require.NoError(t, err, w)
				require.Equal(t, origin.Tenant, tp.Tenant, w)
				require.False(t, tp.System, w)
				require.Equal(t, origin.Publisher, tp.Publisher, w)
			}
		}
		if res, err := origin.Subscribe(s); err == nil {
			for _, w := range res.Wire {
				flt, err := topic.ParseFilter(w)
				require.NoError(t, err, w)
				require.Equal(t, origin.Tenant, flt.Tenant, w)
				switch flt.Root {
				case topic.SystemSegment:
					require.Equal(t, []string{string(topic.Devices), flt.Levels[1], topic.ItemConnection}, flt.Levels, w)
				case string(topic.Devices):
					require.Equal(t, origin.Publisher, flt.Levels[1], w)
				default:
					t.Fatalf("unexpected root in %q", w)
				}
			}
		}
	})
}
