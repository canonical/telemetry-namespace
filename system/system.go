// Package system defines the JSON payloads of the platform subtree
// <tenant>/_system/... (TM102 §1.2). They are retained, QoS 1 messages.
//
// Receivers should ignore unknown JSON fields, so that fields can be added.
package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/canonical/telemetry-namespace/topic"
)

var (
	// ErrInvalidMembership is returned for an invalid GroupMembership.
	ErrInvalidMembership = errors.New("invalid group membership")

	// ErrInvalidConnectionState is returned for an invalid ConnectionState.
	ErrInvalidConnectionState = errors.New("invalid connection state")
)

// GroupMembership is the payload of <tenant>/_system/devices/<device>/groups,
// published by TMS (TM106 §3).
type GroupMembership struct {
	// Version increases with every membership change of the device. A
	// receiver ignores a version lower than or equal to the one it applied.
	Version uint64 `json:"version"`
	// Groups are the slugs of the groups of the device. The built-in group
	// "fleet" is implicit and not listed.
	Groups []string `json:"groups"`
}

// ParseGroupMembership decodes and validates a membership payload.
func ParseGroupMembership(data []byte) (GroupMembership, error) {
	var m GroupMembership
	if err := json.Unmarshal(data, &m); err != nil {
		return GroupMembership{}, fmt.Errorf("%w: %w", ErrInvalidMembership, err)
	}
	if err := m.Validate(); err != nil {
		return GroupMembership{}, err
	}
	return m, nil
}

// Validate checks that every group is a valid slug, listed once, and is not
// the built-in group.
func (m GroupMembership) Validate() error {
	for i, g := range m.Groups {
		if err := topic.ValidateGroupSlug(g); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidMembership, err)
		}
		if slices.Contains(m.Groups[:i], g) {
			return fmt.Errorf("%w: group %q listed twice", ErrInvalidMembership, g)
		}
	}
	return nil
}

// MarshalJSON encodes m, with an empty list rather than null when the device
// is in no group.
func (m GroupMembership) MarshalJSON() ([]byte, error) {
	type plain GroupMembership
	if m.Groups == nil {
		m.Groups = []string{}
	}
	return json.Marshal(plain(m))
}

// State is the connection state of a device.
type State string

// Connection states.
const (
	Online  State = "online"
	Offline State = "offline"
)

// Reason says why the connection state changed.
type Reason string

// Reasons of a connection state change (TM107 §7).
const (
	// Birth is published by the agent after it connects.
	Birth Reason = "birth"
	// Graceful is published by the agent before it disconnects.
	Graceful Reason = "graceful"
	// Lost is the broker will, published when the connection is lost and
	// the device doesn't reconnect within the will delay.
	Lost Reason = "lost"
)

// ConnectionState is the payload of
// <tenant>/_system/devices/<device>/connection, published by the agent and,
// as its will, by the broker (TM107 §7).
type ConnectionState struct {
	// State is Online or Offline.
	State State `json:"state"`
	// Reason is Birth for Online, Graceful or Lost for Offline.
	Reason Reason `json:"reason"`
	// ConnectedAt is when the current or last connection started. The will
	// is fixed at connect time, so for Lost the disconnection time is the
	// receive time.
	ConnectedAt time.Time `json:"connected_at"`
	// AgentVersion is the version of the Connect Agent.
	AgentVersion string `json:"agent_version"`
}

// ParseConnectionState decodes and validates a liveness payload.
func ParseConnectionState(data []byte) (ConnectionState, error) {
	var s ConnectionState
	if err := json.Unmarshal(data, &s); err != nil {
		return ConnectionState{}, fmt.Errorf("%w: %w", ErrInvalidConnectionState, err)
	}
	if err := s.Validate(); err != nil {
		return ConnectionState{}, err
	}
	return s, nil
}

// Validate checks that the state and the reason are consistent.
func (s ConnectionState) Validate() error {
	valid := (s.State == Online && s.Reason == Birth) ||
		(s.State == Offline && (s.Reason == Graceful || s.Reason == Lost))
	if !valid {
		return fmt.Errorf("%w: state %q with reason %q", ErrInvalidConnectionState, s.State, s.Reason)
	}
	return nil
}
