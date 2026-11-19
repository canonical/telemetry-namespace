package props_test

import (
	"testing"

	"github.com/canonical/telemetry-namespace/props"
	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsReserved(t *testing.T) {
	assert.True(t, props.IsReserved("cc-source-device"))
	assert.True(t, props.IsReserved("CC-Dest-Type"))
	assert.True(t, props.IsReserved("cc-"))
	assert.False(t, props.IsReserved("cc"))
	assert.False(t, props.IsReserved("x-cc-source"))
	assert.False(t, props.IsReserved("content-type"))
}

func TestStrip(t *testing.T) {
	in := []props.UserProperty{
		{Key: "trace", Value: "1"},
		{Key: "cc-source-device", Value: "forged"},
		{Key: "CC-DEST-GROUP", Value: "forged"},
		{Key: "trace", Value: "2"},
		{Key: "cc-unknown", Value: "x"},
	}
	want := []props.UserProperty{{Key: "trace", Value: "1"}, {Key: "trace", Value: "2"}}
	assert.Equal(t, want, props.Strip(in))
	assert.Empty(t, props.Strip(nil))
}

func TestDest(t *testing.T) {
	testCases := []struct {
		name  string
		topic string
		want  []props.UserProperty
	}{
		{
			name:  "device topic",
			topic: "T/devices/d1/acme/state",
			want: []props.UserProperty{
				{Key: props.DestType, Value: props.DestTypeDevice},
				{Key: props.DestTenant, Value: "T"},
				{Key: props.DestDevice, Value: "d1"},
				{Key: props.DestPublisher, Value: "acme"},
			},
		},
		{
			name:  "group topic",
			topic: "T/groups/line-3/acme/cmd",
			want: []props.UserProperty{
				{Key: props.DestType, Value: props.DestTypeGroup},
				{Key: props.DestTenant, Value: "T"},
				{Key: props.DestGroup, Value: "line-3"},
				{Key: props.DestPublisher, Value: "acme"},
			},
		},
		{
			name:  "liveness",
			topic: "T/_system/devices/d1/connection",
			want: []props.UserProperty{
				{Key: props.DestType, Value: props.DestTypeDevice},
				{Key: props.DestTenant, Value: "T"},
				{Key: props.DestDevice, Value: "d1"},
				{Key: props.DestPublisher, Value: props.SystemPublisher},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tp, err := topic.Parse(tc.topic)
			require.NoError(t, err)
			assert.Equal(t, tc.want, props.Dest(tp))
		})
	}
}

func TestSetDest_OverwritesForgedDestination(t *testing.T) {
	tp, err := topic.Parse("T/groups/g1/acme/cmd")
	require.NoError(t, err)
	in := []props.UserProperty{
		{Key: "cc-dest-type", Value: "device"},
		{Key: "CC-DEST-DEVICE", Value: "forged"},
		{Key: props.SourceDevice, Value: "sender"},
		{Key: "trace", Value: "1"},
	}
	got := props.SetDest(in, tp)
	want := []props.UserProperty{
		{Key: props.SourceDevice, Value: "sender"},
		{Key: "trace", Value: "1"},
		{Key: props.DestType, Value: props.DestTypeGroup},
		{Key: props.DestTenant, Value: "T"},
		{Key: props.DestGroup, Value: "g1"},
		{Key: props.DestPublisher, Value: "acme"},
	}
	assert.Equal(t, want, got)
}

func TestSources(t *testing.T) {
	assert.Equal(t, []props.UserProperty{
		{Key: props.SourceType, Value: props.SourceTypeDevice},
		{Key: props.SourceTenant, Value: "T"},
		{Key: props.SourceDevice, Value: "d1"},
		{Key: props.SourcePublisher, Value: "acme"},
		{Key: props.SourceSnap, Value: "acme-sensor"},
	}, props.DeviceSource("T", "d1", "acme", "acme-sensor"))

	assert.Equal(t, []props.UserProperty{
		{Key: props.SourceType, Value: props.SourceTypeConnector},
		{Key: props.SourceTenant, Value: "T"},
		{Key: props.SourceConnector, Value: "c-uuid"},
	}, props.ConnectorSource("T", "c-uuid"))
}

func TestGet(t *testing.T) {
	in := []props.UserProperty{{Key: "CC-Dest-Group", Value: "g1"}, {Key: props.DestGroup, Value: "g2"}}
	v, ok := props.Get(in, props.DestGroup)
	assert.True(t, ok)
	assert.Equal(t, "g1", v)

	_, ok = props.Get(in, props.DestDevice)
	assert.False(t, ok)
}
