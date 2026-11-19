package selector_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/selector"
	"github.com/canonical/telemetry-namespace/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name      string
		topic     string
		want      *selector.Selector
		wantLocal string
	}{
		{name: "no selector", topic: "sensor/temp", want: nil, wantLocal: "sensor/temp"},
		{name: "empty", topic: "", want: nil, wantLocal: ""},
		{
			name:      "one device",
			topic:     "[devices: d1]/state",
			want:      &selector.Selector{Devices: []string{"d1"}},
			wantLocal: "state",
		},
		{
			name:      "device list",
			topic:     "[devices: [d1, d2]]/state/#",
			want:      &selector.Selector{Devices: []string{"d1", "d2"}},
			wantLocal: "state/#",
		},
		{
			name:      "no optional spaces",
			topic:     "[devices:[d1,d2]]/state",
			want:      &selector.Selector{Devices: []string{"d1", "d2"}},
			wantLocal: "state",
		},
		{
			name:      "several spaces",
			topic:     "[devices:   [d1,   d2],   publisher:  acme]/state",
			want:      &selector.Selector{Devices: []string{"d1", "d2"}, Publisher: "acme"},
			wantLocal: "state",
		},
		{
			name:      "every device",
			topic:     "[devices: +]/state",
			want:      &selector.Selector{Devices: []string{selector.AllDevices}},
			wantLocal: "state",
		},
		{
			name:      "group",
			topic:     "[groups: line-3]/cmd/reboot",
			want:      &selector.Selector{Groups: []string{"line-3"}},
			wantLocal: "cmd/reboot",
		},
		{
			name:      "fleet",
			topic:     "[groups: fleet]/cmd",
			want:      &selector.Selector{Groups: []string{topic.FleetGroup}},
			wantLocal: "cmd",
		},
		{
			name:      "liveness",
			topic:     "[devices: d1, publisher: _system]/connection",
			want:      &selector.Selector{Devices: []string{"d1"}, Publisher: selector.SystemPublisher},
			wantLocal: "connection",
		},
		{
			name:      "publisher first",
			topic:     "[publisher: _system, devices: +]/connection",
			want:      &selector.Selector{Devices: []string{selector.AllDevices}, Publisher: selector.SystemPublisher},
			wantLocal: "connection",
		},
		{
			name:      "duplicates removed in order",
			topic:     "[groups: [g2, g1, g2]]/cmd",
			want:      &selector.Selector{Groups: []string{"g2", "g1"}},
			wantLocal: "cmd",
		},
		{
			name:      "local topic is not checked",
			topic:     "[devices: d1]/_x/+",
			want:      &selector.Selector{Devices: []string{"d1"}},
			wantLocal: "_x/+",
		},
		{
			name:      "maximum number of targets",
			topic:     "[devices: " + list(selector.MaxTargets) + "]/t",
			want:      &selector.Selector{Devices: ids(selector.MaxTargets)},
			wantLocal: "t",
		},
		{
			name:      "maximum after deduplication",
			topic:     "[devices: [" + strings.Repeat("d1, ", 40) + "d1]]/t",
			want:      &selector.Selector{Devices: []string{"d1"}},
			wantLocal: "t",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, local, err := selector.Parse(tc.topic)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantLocal, local)
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	testCases := []struct {
		name    string
		topic   string
		wantErr error
	}{
		{name: "lone bracket", topic: "[", wantErr: selector.ErrSyntax},
		{name: "empty selector", topic: "[]/t", wantErr: selector.ErrSyntax},
		{name: "not a selector", topic: "[x]/t", wantErr: selector.ErrSyntax},
		{name: "unknown key", topic: "[tenant: t1]/t", wantErr: selector.ErrSyntax},
		{name: "singular key", topic: "[device: d1]/t", wantErr: selector.ErrSyntax},
		{name: "members key", topic: "[members: g1]/t", wantErr: selector.ErrSyntax},
		{name: "uppercase key", topic: "[Devices: d1]/t", wantErr: selector.ErrSyntax},
		{name: "space before colon", topic: "[devices : d1]/t", wantErr: selector.ErrSyntax},
		{name: "space after opening bracket", topic: "[ devices: d1]/t", wantErr: selector.ErrSyntax},
		{name: "space before closing bracket", topic: "[devices: d1 ]/t", wantErr: selector.ErrSyntax},
		{name: "space before comma", topic: "[devices: [d1 , d2]]/t", wantErr: selector.ErrSyntax},
		{name: "tab after colon", topic: "[devices:\td1]/t", wantErr: selector.ErrSyntax},
		{name: "missing colon", topic: "[devices d1]/t", wantErr: selector.ErrSyntax},
		{name: "missing value", topic: "[devices: ]/t", wantErr: selector.ErrSyntax},
		{name: "empty list", topic: "[devices: []]/t", wantErr: selector.ErrSyntax},
		{name: "trailing comma in list", topic: "[devices: [d1,]]/t", wantErr: selector.ErrSyntax},
		{name: "trailing comma", topic: "[devices: d1,]/t", wantErr: selector.ErrSyntax},
		{name: "unclosed list", topic: "[devices: [d1, d2]/t", wantErr: selector.ErrSyntax},
		{name: "unclosed selector", topic: "[devices: d1/t", wantErr: selector.ErrSyntax},
		{name: "nested list", topic: "[devices: [[d1]]]/t", wantErr: selector.ErrSyntax},
		{name: "no slash", topic: "[devices: d1]t", wantErr: selector.ErrSyntax},
		{name: "no local topic", topic: "[devices: d1]", wantErr: selector.ErrSyntax},
		{name: "empty local topic", topic: "[devices: d1]/", wantErr: selector.ErrSyntax},
		{name: "duplicate key", topic: "[devices: d1, devices: d2]/t", wantErr: selector.ErrSyntax},
		{name: "devices and groups", topic: "[devices: d1, groups: g1]/t", wantErr: selector.ErrSyntax},
		{name: "publisher only", topic: "[publisher: acme]/t", wantErr: selector.ErrSyntax},
		{name: "wildcard in a list", topic: "[devices: [d1, +]]/t", wantErr: selector.ErrSyntax},
		{name: "wildcard inside an ID", topic: "[devices: d+]/t", wantErr: selector.ErrSyntax},
		{name: "multi-level wildcard", topic: "[devices: #]/t", wantErr: selector.ErrSyntax},
		{name: "wildcard group", topic: "[groups: +]/t", wantErr: selector.ErrSyntax},
		{name: "wildcard publisher", topic: "[devices: d1, publisher: +]/t", wantErr: selector.ErrSyntax},
		{name: "publisher list", topic: "[devices: d1, publisher: [a, b]]/t", wantErr: selector.ErrSyntax},
		{name: "system publisher with groups", topic: "[groups: g1, publisher: _system]/connection", wantErr: selector.ErrSyntax},
		{name: "invalid device ID", topic: "[devices: _d1]/t", wantErr: topic.ErrInvalidID},
		{name: "invalid group slug", topic: "[groups: $g]/t", wantErr: topic.ErrInvalidID},
		{name: "non-ASCII ID", topic: "[devices: dé]/t", wantErr: selector.ErrSyntax},
		{name: "too many devices", topic: "[devices: " + list(selector.MaxTargets+1) + "]/t", wantErr: selector.ErrTooManyTargets},
		{name: "too many groups", topic: "[groups: " + list(selector.MaxTargets+1) + "]/t", wantErr: selector.ErrTooManyTargets},
		{name: "too many after duplicates", topic: "[devices: [" + strings.Repeat("d0, ", 100) + strings.Join(ids(selector.MaxTargets+1), ", ") + "]]/t", wantErr: selector.ErrTooManyTargets},
		{name: "maximum-length topic of distinct IDs", topic: "[devices: " + list(9000) + "]/t", wantErr: selector.ErrTooManyTargets},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := selector.Parse(tc.topic)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func ids(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("d%d", i)
	}
	return out
}

func list(n int) string {
	return "[" + strings.Join(ids(n), ", ") + "]"
}

// Every valid device ID or group slug can be addressed by a selector: TM102 §2
// limits IDs to the characters the selector grammar accepts.
func FuzzValidIDIsSelectable(f *testing.F) {
	for _, seed := range []string{"d1", "ABC123.pi.acme", "line-3", "a$b", "a_b", "a[b", "urn:x", "usine-é", "a b"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, id string) {
		if topic.ValidateID(id) != nil {
			t.Skip()
		}
		for _, key := range []string{selector.KeyDevices, selector.KeyGroups} {
			sel, local, err := selector.Parse("[" + key + ": [" + id + ", " + id + "]]/t")
			require.NoError(t, err, "id %q", id)
			require.Equal(t, "t", local)
			require.Equal(t, []string{id}, append(sel.Devices, sel.Groups...))
		}
	})
}
