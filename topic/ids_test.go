package topic_test

import (
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/require"
)

func TestValidateID(t *testing.T) {
	testCases := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "device ID", id: "ABC123.pi.acme"},
		{name: "UUID tenant", id: "3e1d0c9a-6f1b-4a57-9d2b-2f2a6b0c1d4e"},
		{name: "group slug", id: "line-3"},
		{name: "fleet is a valid ID", id: topic.FleetGroup},
		{name: "underscore inside", id: "a_b"},
		{name: "dollar inside", id: "a$b"},
		{name: "non-ASCII", id: "usine-é", wantErr: true},
		{name: "delete character", id: "a\x7f", wantErr: true},
		{name: "empty", id: "", wantErr: true},
		{name: "leading underscore", id: "_system", wantErr: true},
		{name: "leading dollar", id: "$share", wantErr: true},
		{name: "slash", id: "a/b", wantErr: true},
		{name: "single-level wildcard", id: "+", wantErr: true},
		{name: "multi-level wildcard", id: "a#", wantErr: true},
		{name: "space", id: "a b", wantErr: true},
		{name: "tab", id: "a\tb", wantErr: true},
		{name: "non-breaking space", id: "a\u00a0b", wantErr: true},
		{name: "control character", id: "a\x01b", wantErr: true},
		{name: "null character", id: "a\x00", wantErr: true},
		{name: "invalid UTF-8", id: "a\xff", wantErr: true},
		{name: "opening bracket", id: "a[b", wantErr: true},
		{name: "closing bracket", id: "a]", wantErr: true},
		{name: "comma", id: "a,b", wantErr: true},
		{name: "colon", id: "urn:a", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := topic.ValidateID(tc.id)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidID)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateGroupSlug(t *testing.T) {
	require.NoError(t, topic.ValidateGroupSlug("line-3"))
	require.ErrorIs(t, topic.ValidateGroupSlug(topic.FleetGroup), topic.ErrReservedID)
	require.ErrorIs(t, topic.ValidateGroupSlug("_g"), topic.ErrInvalidID)
}

func TestValidatePublisher(t *testing.T) {
	testCases := []struct {
		name      string
		publisher string
		wantErr   bool
	}{
		{name: "store publisher", publisher: "acme"},
		{name: "store account ID", publisher: "Xp2B7rJ8Vq9Tz3Kd"},
		{name: "local", publisher: topic.LocalPublisher},
		{name: "platform reserved", publisher: "_system", wantErr: true},
		{name: "empty", publisher: "", wantErr: true},
		{name: "wildcard", publisher: "+", wantErr: true},
		{name: "slash", publisher: "a/b", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := topic.ValidatePublisher(tc.publisher)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidPublisher)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateLocalTopic(t *testing.T) {
	testCases := []struct {
		name    string
		topic   string
		wantErr bool
	}{
		{name: "single level", topic: "state"},
		{name: "several levels", topic: "sensor/temp"},
		{name: "underscore inside", topic: "a/_b"},
		{name: "dollar inside", topic: "a/$b"},
		{name: "bracket inside", topic: "a/[b]"},
		{name: "empty level 0", topic: "/a"},
		{name: "empty", topic: "", wantErr: true},
		{name: "leading dollar", topic: "$SYS/x", wantErr: true},
		{name: "leading underscore", topic: "_http-response/1", wantErr: true},
		{name: "leading bracket", topic: "[devices: d1]/t", wantErr: true},
		{name: "wildcard", topic: "a/+", wantErr: true},
		{name: "too long", topic: strings.Repeat("a", 65536), wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := topic.ValidateLocalTopic(tc.topic)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidLocalTopic)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateLocalFilter(t *testing.T) {
	testCases := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{name: "literal", filter: "state"},
		{name: "multi-level alone", filter: "#"},
		{name: "single-level first", filter: "+/temp"},
		{name: "trailing multi-level", filter: "sensor/#"},
		{name: "empty", filter: "", wantErr: true},
		{name: "leading dollar", filter: "$SYS/#", wantErr: true},
		{name: "shared", filter: "$share/g/a", wantErr: true},
		{name: "leading underscore", filter: "_system/#", wantErr: true},
		{name: "leading bracket", filter: "[devices: +]/state", wantErr: true},
		{name: "misplaced wildcard", filter: "a/#/b", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := topic.ValidateLocalFilter(tc.filter)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidLocalTopic)
				return
			}
			require.NoError(t, err)
		})
	}
}
