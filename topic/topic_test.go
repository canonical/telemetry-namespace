package topic_test

import (
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenant = "3e1d"
	device = "ABC123.pi.acme"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name    string
		topic   string
		want    topic.Topic
		wantErr bool
	}{
		{
			name:  "device topic",
			topic: "3e1d/devices/ABC123.pi.acme/acme/sensor/temp",
			want:  topic.Topic{Tenant: tenant, Kind: topic.Devices, Target: device, Publisher: "acme", Local: "sensor/temp"},
		},
		{
			name:  "group topic",
			topic: "3e1d/groups/line-3/acme/cmd/reboot",
			want:  topic.Topic{Tenant: tenant, Kind: topic.Groups, Target: "line-3", Publisher: "acme", Local: "cmd/reboot"},
		},
		{
			name:  "fleet topic",
			topic: "3e1d/groups/fleet/acme/config",
			want:  topic.Topic{Tenant: tenant, Kind: topic.Groups, Target: topic.FleetGroup, Publisher: "acme", Local: "config"},
		},
		{
			name:  "device liveness",
			topic: "3e1d/_system/devices/ABC123.pi.acme/connection",
			want:  topic.Topic{Tenant: tenant, System: true, Kind: topic.Devices, Target: device, Local: topic.ItemConnection},
		},
		{
			name:  "group platform item",
			topic: "3e1d/_system/groups/line-3/members",
			want:  topic.Topic{Tenant: tenant, System: true, Kind: topic.Groups, Target: "line-3", Local: "members"},
		},
		{
			name:  "HTTP response",
			topic: "3e1d/devices/ABC123.pi.acme/acme/_http-response/42",
			want:  topic.Topic{Tenant: tenant, Kind: topic.Devices, Target: device, Publisher: "acme", Local: "_http-response/42"},
		},
		{
			name:  "empty local level",
			topic: "3e1d/devices/d1/acme//x",
			want:  topic.Topic{Tenant: tenant, Kind: topic.Devices, Target: "d1", Publisher: "acme", Local: "/x"},
		},
		{name: "v1 leading slash", topic: "/3e1d/ABC123.pi.acme/acme/state", wantErr: true},
		{name: "too few levels", topic: "3e1d/devices/d1/acme", wantErr: true},
		{name: "empty local topic", topic: "3e1d/devices/d1/acme/", wantErr: true},
		{name: "unknown kind", topic: "3e1d/reports/g1/acme/x", wantErr: true},
		{name: "unknown platform kind", topic: "3e1d/_system/tenants/t1/x", wantErr: true},
		{name: "invalid tenant", topic: "_t/devices/d1/acme/x", wantErr: true},
		{name: "empty target", topic: "3e1d/devices//acme/x", wantErr: true},
		{name: "reserved publisher", topic: "3e1d/devices/d1/_system/state/connection", wantErr: true},
		{name: "empty platform item", topic: "3e1d/_system/devices/d1/", wantErr: true},
		{name: "wildcard", topic: "3e1d/devices/+/acme/x", wantErr: true},
		{name: "empty", topic: "", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := topic.Parse(tc.topic)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidTopic)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.topic, got.String())
		})
	}
}

func TestConstructors(t *testing.T) {
	testCases := []struct {
		name string
		got  func() (topic.Topic, error)
		want string
	}{
		{
			name: "device",
			got:  func() (topic.Topic, error) { return topic.NewDevice(tenant, device, "acme", "sensor/temp") },
			want: "3e1d/devices/ABC123.pi.acme/acme/sensor/temp",
		},
		{
			name: "group",
			got:  func() (topic.Topic, error) { return topic.NewGroup(tenant, "line-3", "acme", "cmd") },
			want: "3e1d/groups/line-3/acme/cmd",
		},
		{
			name: "system",
			got:  func() (topic.Topic, error) { return topic.NewSystem(tenant, topic.Groups, "line-3", "x") },
			want: "3e1d/_system/groups/line-3/x",
		},
		{
			name: "device connection",
			got:  func() (topic.Topic, error) { return topic.DeviceConnection(tenant, device) },
			want: "3e1d/_system/devices/ABC123.pi.acme/connection",
		},
		{
			name: "device membership",
			got:  func() (topic.Topic, error) { return topic.DeviceMembership(tenant, device) },
			want: "3e1d/_system/devices/ABC123.pi.acme/groups",
		},
		{
			name: "HTTP response",
			got:  func() (topic.Topic, error) { return topic.HTTPResponse(tenant, device, "acme", "c0ffee") },
			want: "3e1d/devices/ABC123.pi.acme/acme/_http-response/c0ffee",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.got()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())
		})
	}
}

func TestConstructors_Invalid(t *testing.T) {
	testCases := []struct {
		name string
		got  func() (topic.Topic, error)
	}{
		{name: "device with invalid tenant", got: func() (topic.Topic, error) { return topic.NewDevice("", device, "acme", "x") }},
		{name: "device with invalid ID", got: func() (topic.Topic, error) { return topic.NewDevice(tenant, "a/b", "acme", "x") }},
		{name: "device with reserved publisher", got: func() (topic.Topic, error) { return topic.NewDevice(tenant, device, "_system", "x") }},
		{name: "device with empty local topic", got: func() (topic.Topic, error) { return topic.NewDevice(tenant, device, "acme", "") }},
		{name: "device with wildcard local topic", got: func() (topic.Topic, error) { return topic.NewDevice(tenant, device, "acme", "a/#") }},
		{name: "group with invalid slug", got: func() (topic.Topic, error) { return topic.NewGroup(tenant, "+", "acme", "x") }},
		{name: "system with unknown kind", got: func() (topic.Topic, error) { return topic.NewSystem(tenant, "tenants", "t1", "x") }},
		{name: "system with empty item", got: func() (topic.Topic, error) { return topic.NewSystem(tenant, topic.Devices, device, "") }},
		{name: "HTTP response without correlation", got: func() (topic.Topic, error) { return topic.HTTPResponse(tenant, device, "acme", "") }},
		{name: "HTTP response with multi-level correlation", got: func() (topic.Topic, error) { return topic.HTTPResponse(tenant, device, "acme", "a/b") }},
		{name: "too long", got: func() (topic.Topic, error) {
			return topic.NewDevice(tenant, device, "acme", strings.Repeat("a", 65535))
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.got()
			require.ErrorIs(t, err, topic.ErrInvalidTopic)
		})
	}
}

func TestValidate_PlatformTopicWithPublisher(t *testing.T) {
	tp := topic.Topic{Tenant: tenant, System: true, Kind: topic.Devices, Target: device, Publisher: "acme", Local: "x"}
	require.ErrorIs(t, tp.Validate(), topic.ErrInvalidTopic)
}

func TestHTTPResponseCorrelation(t *testing.T) {
	testCases := []struct {
		name   string
		topic  string
		want   string
		wantOK bool
	}{
		{name: "response", topic: "3e1d/devices/d1/acme/_http-response/42", want: "42", wantOK: true},
		{name: "plain device topic", topic: "3e1d/devices/d1/acme/state", wantOK: false},
		{name: "deeper than one level", topic: "3e1d/devices/d1/acme/_http-response/42/x", wantOK: false},
		{name: "group topic", topic: "3e1d/groups/g1/acme/_http-response/42", wantOK: false},
		{name: "platform topic", topic: "3e1d/_system/devices/d1/_http-response/42", wantOK: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tp, err := topic.Parse(tc.topic)
			require.NoError(t, err)
			got, ok := tp.HTTPResponseCorrelation()
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"3e1d/devices/d1/acme/state",
		"3e1d/groups/fleet/acme/cmd/#",
		"3e1d/_system/devices/d1/connection",
		"/3e1d/d1/acme/x",
		"t/devices/d1/_x/y",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		tp, err := topic.Parse(s)
		if err != nil {
			return
		}
		require.Equal(t, s, tp.String())
		again, err := topic.Parse(tp.String())
		require.NoError(t, err)
		require.Equal(t, tp, again)
	})
}
