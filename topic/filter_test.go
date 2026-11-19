package topic_test

import (
	"testing"

	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilter(t *testing.T) {
	testCases := []struct {
		name    string
		filter  string
		want    topic.Filter
		wantErr bool
	}{
		{
			name:   "all devices",
			filter: "3e1d/devices/#",
			want:   topic.Filter{Tenant: tenant, Root: "devices", Levels: []string{"#"}},
		},
		{
			name:   "one publisher on every device",
			filter: "3e1d/devices/+/acme/state/#",
			want:   topic.Filter{Tenant: tenant, Root: "devices", Levels: []string{"+", "acme", "state", "#"}},
		},
		{
			name:   "local-topic expansion",
			filter: "3e1d/devices/+/+/stats/#",
			want:   topic.Filter{Tenant: tenant, Root: "devices", Levels: []string{"+", "+", "stats", "#"}},
		},
		{
			name:   "privileged group grant",
			filter: "3e1d/groups/line-3/#",
			want:   topic.Filter{Tenant: tenant, Root: "groups", Levels: []string{"line-3", "#"}},
		},
		{
			name:   "liveness of every device",
			filter: "3e1d/_system/devices/+/connection",
			want:   topic.Filter{Tenant: tenant, Root: "_system", Levels: []string{"devices", "+", "connection"}},
		},
		{
			name:   "whole platform subtree",
			filter: "3e1d/_system/#",
			want:   topic.Filter{Tenant: tenant, Root: "_system", Levels: []string{"#"}},
		},
		{
			name:   "a topic name is a filter",
			filter: "3e1d/devices/d1/acme/state",
			want:   topic.Filter{Tenant: tenant, Root: "devices", Levels: []string{"d1", "acme", "state"}},
		},
		{name: "v1 leading slash", filter: "/3e1d/+/+/x", wantErr: true},
		{name: "wildcard tenant", filter: "+/devices/#", wantErr: true},
		{name: "wildcard root", filter: "3e1d/+/d1/#", wantErr: true},
		{name: "whole tenant", filter: "3e1d/#", wantErr: true},
		{name: "unknown root", filter: "3e1d/reports/#", wantErr: true},
		{name: "nothing after the root", filter: "3e1d/devices", wantErr: true},
		{name: "shared subscription", filter: "$share/c1/3e1d/devices/#", wantErr: true},
		{name: "invalid device ID", filter: "3e1d/devices/_d1/#", wantErr: true},
		{name: "reserved publisher", filter: "3e1d/devices/+/_system/#", wantErr: true},
		{name: "unknown platform kind", filter: "3e1d/_system/tenants/#", wantErr: true},
		{name: "invalid platform target", filter: "3e1d/_system/devices/$d/#", wantErr: true},
		{name: "misplaced wildcard", filter: "3e1d/devices/#/acme", wantErr: true},
		{name: "partial wildcard", filter: "3e1d/devices/d+/acme", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := topic.ParseFilter(tc.filter)
			if tc.wantErr {
				require.ErrorIs(t, err, topic.ErrInvalidFilter)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.filter, got.String())
		})
	}
}

func TestFilterValidate_NoLevels(t *testing.T) {
	f := topic.Filter{Tenant: tenant, Root: "devices"}
	require.ErrorIs(t, f.Validate(), topic.ErrInvalidFilter)
}

func TestFilterTarget(t *testing.T) {
	testCases := []struct {
		name       string
		filter     string
		wantKind   topic.Kind
		wantTarget string
		wantOK     bool
	}{
		{name: "device", filter: "3e1d/devices/d1/acme/x", wantKind: topic.Devices, wantTarget: "d1", wantOK: true},
		{name: "device subtree", filter: "3e1d/devices/d1/#", wantKind: topic.Devices, wantTarget: "d1", wantOK: true},
		{name: "group", filter: "3e1d/groups/g1/acme/#", wantKind: topic.Groups, wantTarget: "g1", wantOK: true},
		{name: "platform device", filter: "3e1d/_system/devices/d1/connection", wantKind: topic.Devices, wantTarget: "d1", wantOK: true},
		{name: "platform group", filter: "3e1d/_system/groups/g1/#", wantKind: topic.Groups, wantTarget: "g1", wantOK: true},
		{name: "wildcard device", filter: "3e1d/devices/+/acme/x", wantOK: false},
		{name: "multi-level device", filter: "3e1d/devices/#", wantOK: false},
		{name: "wildcard platform kind", filter: "3e1d/_system/+/d1/x", wantOK: false},
		{name: "platform kind only", filter: "3e1d/_system/devices/#", wantOK: false},
		{name: "whole platform subtree", filter: "3e1d/_system/#", wantOK: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := topic.ParseFilter(tc.filter)
			require.NoError(t, err)
			kind, target, ok := f.Target()
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantKind, kind)
			assert.Equal(t, tc.wantTarget, target)
		})
	}
}

func TestFilterTarget_UnknownRoot(t *testing.T) {
	_, _, ok := topic.Filter{Tenant: tenant, Root: "reports", Levels: []string{"g1"}}.Target()
	assert.False(t, ok)
}

func TestSubtrees(t *testing.T) {
	f, err := topic.Subtree(tenant, topic.Devices, device)
	require.NoError(t, err)
	assert.Equal(t, "3e1d/devices/ABC123.pi.acme/#", f.String())

	f, err = topic.Subtree(tenant, topic.Groups, topic.FleetGroup)
	require.NoError(t, err)
	assert.Equal(t, "3e1d/groups/fleet/#", f.String())

	f, err = topic.SystemSubtree(tenant, topic.Devices, device)
	require.NoError(t, err)
	assert.Equal(t, "3e1d/_system/devices/ABC123.pi.acme/#", f.String())

	_, err = topic.Subtree(tenant, topic.Groups, "+")
	require.ErrorIs(t, err, topic.ErrInvalidFilter)

	_, err = topic.SystemSubtree(tenant, "tenants", "t1")
	require.ErrorIs(t, err, topic.ErrInvalidFilter)
}
