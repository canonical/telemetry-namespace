package match_test

import (
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/match"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	testCases := []struct {
		name    string
		topic   string
		wantErr bool
	}{
		{name: "single level", topic: "a"},
		{name: "several levels", topic: "a/b/c"},
		{name: "empty levels", topic: "/a//b/"},
		{name: "dollar prefix", topic: "$SYS/broker"},
		{name: "unicode", topic: "températures/°C"},
		{name: "maximum length", topic: strings.Repeat("a", match.MaxLength)},
		{name: "empty", topic: "", wantErr: true},
		{name: "too long", topic: strings.Repeat("a", match.MaxLength+1), wantErr: true},
		{name: "single-level wildcard", topic: "a/+/c", wantErr: true},
		{name: "multi-level wildcard", topic: "a/#", wantErr: true},
		{name: "wildcard inside a level", topic: "a/b+c", wantErr: true},
		{name: "null character", topic: "a\x00b", wantErr: true},
		{name: "invalid UTF-8", topic: "a/\xff", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := match.ValidateName(tc.topic)
			if tc.wantErr {
				require.ErrorIs(t, err, match.ErrInvalidName)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateFilter(t *testing.T) {
	testCases := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{name: "plain", filter: "a/b"},
		{name: "multi-level alone", filter: "#"},
		{name: "single-level alone", filter: "+"},
		{name: "trailing multi-level", filter: "a/b/#"},
		{name: "single-level in the middle", filter: "a/+/c"},
		{name: "several single-level", filter: "+/+/+"},
		{name: "single-level then multi-level", filter: "+/#"},
		{name: "empty levels", filter: "a//+/"},
		{name: "shared", filter: "$share/group/a/+/#"},
		{name: "shared multi-level", filter: "$share/group/#"},
		{name: "maximum length", filter: strings.Repeat("a", match.MaxLength)},
		{name: "empty", filter: "", wantErr: true},
		{name: "too long", filter: strings.Repeat("a", match.MaxLength+1), wantErr: true},
		{name: "multi-level not last", filter: "a/#/b", wantErr: true},
		{name: "multi-level inside a level", filter: "a/b#", wantErr: true},
		{name: "single-level inside a level", filter: "a/b+/c", wantErr: true},
		{name: "two wildcards in a level", filter: "a/+#", wantErr: true},
		{name: "null character", filter: "a/\x00", wantErr: true},
		{name: "invalid UTF-8", filter: "\xff/#", wantErr: true},
		{name: "shared without filter", filter: "$share/group", wantErr: true},
		{name: "shared with empty filter", filter: "$share/group/", wantErr: true},
		{name: "shared with empty name", filter: "$share//a", wantErr: true},
		{name: "shared name with wildcard", filter: "$share/g+/a", wantErr: true},
		{name: "shared with invalid filter", filter: "$share/group/a/#/b", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := match.ValidateFilter(tc.filter)
			if tc.wantErr {
				require.ErrorIs(t, err, match.ErrInvalidFilter)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSplitShared(t *testing.T) {
	testCases := []struct {
		name      string
		filter    string
		wantShare string
		wantRest  string
		wantOK    bool
	}{
		{name: "shared", filter: "$share/g1/a/b/#", wantShare: "g1", wantRest: "a/b/#", wantOK: true},
		{name: "not shared", filter: "a/b", wantOK: false},
		{name: "prefix without separator", filter: "$sharegroup/a", wantOK: false},
		{name: "no filter", filter: "$share/g1", wantShare: "g1", wantOK: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			share, rest, ok := match.SplitShared(tc.filter)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantShare, share)
			assert.Equal(t, tc.wantRest, rest)
		})
	}
}

func TestIsWildcard(t *testing.T) {
	assert.True(t, match.IsWildcard("a/+"))
	assert.True(t, match.IsWildcard("#"))
	assert.False(t, match.IsWildcard("a/b"))
}

func TestMatches(t *testing.T) {
	testCases := []struct {
		name   string
		filter string
		topic  string
		want   bool
	}{
		{name: "exact", filter: "a/b/c", topic: "a/b/c", want: true},
		{name: "different literal", filter: "a/b/c", topic: "a/b/d", want: false},
		{name: "single-level", filter: "sensors/+/room1", topic: "sensors/temp/room1", want: true},
		{name: "single-level, different count", filter: "sensors/+", topic: "sensors/temp/room1", want: false},
		{name: "single-level matches an empty level", filter: "a/+/c", topic: "a//c", want: true},
		{name: "single-level doesn't match a missing level", filter: "a/+", topic: "a", want: false},
		{name: "multi-level", filter: "sensors/#", topic: "sensors/temp/room1", want: true},
		{name: "multi-level matches the parent level", filter: "sport/tennis/#", topic: "sport/tennis", want: true},
		{name: "multi-level, different prefix", filter: "sport/#", topic: "news/today", want: false},
		{name: "multi-level alone", filter: "#", topic: "a/b/c", want: true},
		{name: "single-level then multi-level", filter: "+/tennis/#", topic: "sport/tennis/player1", want: true},
		{name: "filter longer than topic", filter: "a/b/c", topic: "a/b", want: false},
		{name: "topic longer than filter", filter: "a/b", topic: "a/b/c", want: false},
		{name: "leading empty level", filter: "+/a", topic: "/a", want: true},
		{name: "dollar topic not matched by multi-level", filter: "#", topic: "$SYS/broker", want: false},
		{name: "dollar topic not matched by single-level", filter: "+/broker", topic: "$SYS/broker", want: false},
		{name: "dollar topic matched by literal", filter: "$SYS/#", topic: "$SYS/broker", want: true},
		{name: "shared filter", filter: "$share/g1/a/+", topic: "a/b", want: true},
		{name: "shared filter, no match", filter: "$share/g1/a/+", topic: "b/b", want: false},
		{name: "wire device slot", filter: "T/devices/d1/#", topic: "T/devices/d1/acme/state", want: true},
		{name: "wire group, other device", filter: "T/devices/+/acme/#", topic: "T/groups/g1/acme/cmd", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, match.Matches(tc.filter, tc.topic))
		})
	}
}

func TestContains(t *testing.T) {
	testCases := []struct {
		name  string
		outer string
		inner string
		want  bool
	}{
		{name: "equal literals", outer: "a/b", inner: "a/b", want: true},
		{name: "equal wildcards", outer: "a/+/#", inner: "a/+/#", want: true},
		{name: "multi-level contains literal", outer: "a/#", inner: "a/b/c", want: true},
		{name: "multi-level contains its parent", outer: "a/#", inner: "a", want: true},
		{name: "multi-level contains narrower multi-level", outer: "a/#", inner: "a/b/#", want: true},
		{name: "multi-level alone contains everything", outer: "#", inner: "+/b/#", want: true},
		{name: "single-level contains literal", outer: "a/+/c", inner: "a/b/c", want: true},
		{name: "single-level contains single-level", outer: "a/+", inner: "a/+", want: true},
		{name: "literal doesn't contain single-level", outer: "a/b", inner: "a/+", want: false},
		{name: "single-level doesn't contain multi-level", outer: "a/+", inner: "a/#", want: false},
		{name: "narrower multi-level doesn't contain wider", outer: "a/b/#", inner: "a/#", want: false},
		{name: "single-level doesn't contain the parent", outer: "a/+", inner: "a", want: false},
		{name: "longer doesn't contain shorter", outer: "a/b/c", inner: "a/b", want: false},
		{name: "shorter doesn't contain longer", outer: "a/b", inner: "a/b/c", want: false},
		{name: "different literals", outer: "a/b", inner: "a/c", want: false},
		{name: "wildcard doesn't contain a dollar topic", outer: "#", inner: "$SYS/#", want: false},
		{name: "literal dollar contains dollar", outer: "$SYS/#", inner: "$SYS/broker", want: true},
		{name: "shared filters", outer: "$share/g1/a/#", inner: "$share/g2/a/b", want: true},
		{name: "device wildcard contains one device", outer: "T/devices/+/acme/#", inner: "T/devices/d1/acme/state/#", want: true},
		{name: "one device doesn't contain all devices", outer: "T/devices/d1/#", inner: "T/devices/+/acme/#", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, match.Contains(tc.outer, tc.inner))
		})
	}
}

// A topic name is a filter that matches only itself, so Contains and Matches
// must agree on it.
func FuzzContainsAgreesWithMatches(f *testing.F) {
	for _, seed := range [][2]string{
		{"a/#", "a"},
		{"a/+/c", "a/b/c"},
		{"#", "$SYS/x"},
		{"+/+", "a/b"},
		{"a/b", "a/b/c"},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, filter, name string) {
		if match.ValidateFilter(filter) != nil || match.ValidateName(name) != nil {
			t.Skip()
		}
		if strings.HasPrefix(name, match.SharePrefix) {
			t.Skip()
		}
		assert.Equal(t, match.Matches(filter, name), match.Contains(filter, name), "filter %q, name %q", filter, name)
		assert.True(t, match.Contains(filter, filter), "filter %q must contain itself", filter)
	})
}
