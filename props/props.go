// Package props defines the cc-* MQTT 5 user properties that carry Telemetry
// metadata (TM102 §4), and helpers to strip and stamp them.
//
// Trust: the cc-dest-* properties are set by the receiving agent from the
// wire topic, so they are guaranteed by the broker ACLs. The cc-source-*
// properties are asserted by the sender and not verified by the platform: a
// snap can't forge them, but a compromised device or connector can.
//
// Receivers must ignore unknown cc-* properties.
package props

import (
	"strings"

	"github.com/canonical/telemetry-namespace/topic"
)

// Prefix starts every Telemetry user property. It is matched
// case-insensitively.
const Prefix = "cc-"

// Destination properties, set by the receiving agent from the wire topic.
const (
	// DestType is DestTypeDevice or DestTypeGroup.
	DestType = "cc-dest-type"
	// DestTenant is the tenant of the wire topic.
	DestTenant = "cc-dest-tenant"
	// DestDevice is the device the message was addressed to, when DestType is
	// DestTypeDevice.
	DestDevice = "cc-dest-device"
	// DestGroup is the group the message was addressed to, when DestType is
	// DestTypeGroup.
	DestGroup = "cc-dest-group"
	// DestPublisher is the publisher segment of the wire topic, or "_system"
	// for liveness.
	DestPublisher = "cc-dest-publisher"
)

// Source properties, asserted by the sender.
const (
	// SourceType is SourceTypeDevice or SourceTypeConnector.
	SourceType = "cc-source-type"
	// SourceTenant is the tenant of the sender.
	SourceTenant = "cc-source-tenant"
	// SourceDevice is the sending device, when SourceType is SourceTypeDevice.
	SourceDevice = "cc-source-device"
	// SourcePublisher is the publisher of the sending snap.
	SourcePublisher = "cc-source-publisher"
	// SourceSnap is the name of the sending snap.
	SourceSnap = "cc-source-snap"
	// SourceConnector is the UUID of the sending connector, when SourceType
	// is SourceTypeConnector.
	SourceConnector = "cc-source-connector"
)

// Values of DestType and SourceType.
const (
	DestTypeDevice      = "device"
	DestTypeGroup       = "group"
	SourceTypeDevice    = "device"
	SourceTypeConnector = "connector"
)

// SystemPublisher is the value of DestPublisher for platform data delivered
// to snaps (device liveness).
const SystemPublisher = "_system"

const destPrefix = "cc-dest-"

// UserProperty is an MQTT 5 user property.
type UserProperty struct {
	Key   string
	Value string
}

// IsReserved reports whether key is a Telemetry property, i.e. starts with
// "cc-", in any case.
func IsReserved(key string) bool {
	return hasPrefixFold(key, Prefix)
}

// Strip returns props without the Telemetry properties. The agent strips
// every cc-* property a snap sets before publishing. The order of the other
// properties, duplicates included, is kept.
func Strip(props []UserProperty) []UserProperty {
	return without(props, Prefix)
}

// Dest returns the cc-dest-* properties of a wire topic. For a platform
// topic, DestPublisher is SystemPublisher.
func Dest(t topic.Topic) []UserProperty {
	publisher := t.Publisher
	if t.System {
		publisher = SystemPublisher
	}
	out := []UserProperty{{Key: DestType, Value: DestTypeDevice}, {Key: DestTenant, Value: t.Tenant}}
	if t.Kind == topic.Groups {
		out[0].Value = DestTypeGroup
		out = append(out, UserProperty{Key: DestGroup, Value: t.Target})
	} else {
		out = append(out, UserProperty{Key: DestDevice, Value: t.Target})
	}
	return append(out, UserProperty{Key: DestPublisher, Value: publisher})
}

// SetDest replaces any cc-dest-* property of props with the cc-dest-*
// properties of the wire topic t. Other properties, including cc-source-*,
// are kept.
func SetDest(props []UserProperty, t topic.Topic) []UserProperty {
	return append(without(props, destPrefix), Dest(t)...)
}

// DeviceSource returns the cc-source-* properties of a message sent by a
// snap on a device.
func DeviceSource(tenant, device, publisher, snap string) []UserProperty {
	return []UserProperty{
		{Key: SourceType, Value: SourceTypeDevice},
		{Key: SourceTenant, Value: tenant},
		{Key: SourceDevice, Value: device},
		{Key: SourcePublisher, Value: publisher},
		{Key: SourceSnap, Value: snap},
	}
}

// ConnectorSource returns the cc-source-* properties of a message sent by a
// connector.
func ConnectorSource(tenant, connector string) []UserProperty {
	return []UserProperty{
		{Key: SourceType, Value: SourceTypeConnector},
		{Key: SourceTenant, Value: tenant},
		{Key: SourceConnector, Value: connector},
	}
}

// Get returns the value of the first property named key, compared
// case-insensitively.
func Get(props []UserProperty, key string) (string, bool) {
	for _, p := range props {
		if strings.EqualFold(p.Key, key) {
			return p.Value, true
		}
	}
	return "", false
}

func without(props []UserProperty, prefix string) []UserProperty {
	out := make([]UserProperty, 0, len(props))
	for _, p := range props {
		if !hasPrefixFold(p.Key, prefix) {
			out = append(out, p)
		}
	}
	return out
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}
