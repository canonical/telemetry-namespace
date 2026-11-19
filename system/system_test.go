package system_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/canonical/telemetry-namespace/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGroupMembership(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
		want    system.GroupMembership
		wantErr bool
	}{
		{
			name:    "spec example",
			payload: `{"version": 42, "groups": ["eu-central", "line-3"]}`,
			want:    system.GroupMembership{Version: 42, Groups: []string{"eu-central", "line-3"}},
		},
		{
			name:    "no group",
			payload: `{"version": 3, "groups": []}`,
			want:    system.GroupMembership{Version: 3, Groups: []string{}},
		},
		{
			name:    "unknown field ignored",
			payload: `{"version": 1, "groups": ["g1"], "extra": true}`,
			want:    system.GroupMembership{Version: 1, Groups: []string{"g1"}},
		},
		{name: "fleet listed", payload: `{"version": 1, "groups": ["fleet"]}`, wantErr: true},
		{name: "invalid slug", payload: `{"version": 1, "groups": ["a/b"]}`, wantErr: true},
		{name: "duplicate", payload: `{"version": 1, "groups": ["g1", "g1"]}`, wantErr: true},
		{name: "negative version", payload: `{"version": -1, "groups": []}`, wantErr: true},
		{name: "not JSON", payload: `groups`, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := system.ParseGroupMembership([]byte(tc.payload))
			if tc.wantErr {
				require.ErrorIs(t, err, system.ErrInvalidMembership)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGroupMembership_MarshalJSON(t *testing.T) {
	data, err := json.Marshal(system.GroupMembership{Version: 7})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": 7, "groups": []}`, string(data))

	data, err = json.Marshal(system.GroupMembership{Version: 8, Groups: []string{"g1"}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": 8, "groups": ["g1"]}`, string(data))
}

func TestParseConnectionState(t *testing.T) {
	connectedAt := time.Date(2026, 9, 30, 8, 12, 3, 0, time.UTC)
	testCases := []struct {
		name    string
		payload string
		want    system.ConnectionState
		wantErr bool
	}{
		{
			name:    "birth",
			payload: `{"state": "online", "reason": "birth", "connected_at": "2026-09-30T08:12:03Z", "agent_version": "0.7.0"}`,
			want:    system.ConnectionState{State: system.Online, Reason: system.Birth, ConnectedAt: connectedAt, AgentVersion: "0.7.0"},
		},
		{
			name:    "graceful",
			payload: `{"state": "offline", "reason": "graceful", "connected_at": "2026-09-30T08:12:03Z", "agent_version": "0.7.0"}`,
			want:    system.ConnectionState{State: system.Offline, Reason: system.Graceful, ConnectedAt: connectedAt, AgentVersion: "0.7.0"},
		},
		{
			name:    "lost",
			payload: `{"state": "offline", "reason": "lost", "connected_at": "2026-09-30T08:12:03Z", "agent_version": "0.7.0"}`,
			want:    system.ConnectionState{State: system.Offline, Reason: system.Lost, ConnectedAt: connectedAt, AgentVersion: "0.7.0"},
		},
		{name: "online but lost", payload: `{"state": "online", "reason": "lost"}`, wantErr: true},
		{name: "offline birth", payload: `{"state": "offline", "reason": "birth"}`, wantErr: true},
		{name: "unknown state", payload: `{"state": "sleeping", "reason": "birth"}`, wantErr: true},
		{name: "missing fields", payload: `{}`, wantErr: true},
		{name: "invalid time", payload: `{"state": "online", "reason": "birth", "connected_at": "yesterday"}`, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := system.ParseConnectionState([]byte(tc.payload))
			if tc.wantErr {
				require.ErrorIs(t, err, system.ErrInvalidConnectionState)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConnectionState_RoundTrip(t *testing.T) {
	in := system.ConnectionState{
		State:        system.Online,
		Reason:       system.Birth,
		ConnectedAt:  time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC),
		AgentVersion: "0.7.0",
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"state":"online","reason":"birth","connected_at":"2026-11-19T10:00:00Z","agent_version":"0.7.0"}`, string(data))

	out, err := system.ParseConnectionState(data)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}
