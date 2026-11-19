package fga_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/canonical/telemetry-namespace/fga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OpenFGA object validation: openfga/openfga pkg/tuple objectRegex and the
// TupleKey object pattern of openfga/api.
var (
	openFGAObject  = regexp.MustCompile(`^[^:#\s]+:[^#:\s]+$`)
	openFGAPattern = regexp.MustCompile(`^[^\s]{2,256}$`)
)

func TestEncodeTopic(t *testing.T) {
	testCases := []struct {
		name   string
		filter string
		want   string
	}{
		{name: "no special character", filter: "T/devices/d1/acme/state", want: "topic:T/devices/d1/acme/state"},
		{name: "multi-level wildcard", filter: "T/devices/#", want: "topic:T/devices/%23"},
		{name: "single-level wildcard unchanged", filter: "T/devices/+/acme/#", want: "topic:T/devices/+/acme/%23"},
		{name: "literal percent", filter: "a/100%", want: "topic:a/100%25"},
		{name: "literal encoded sequence", filter: "a/%23", want: "topic:a/%2523"},
		{name: "colon", filter: "T/devices/+/acme/urn:x/#", want: "topic:T/devices/+/acme/urn%3Ax/%23"},
		{name: "space", filter: "a/my topic", want: "topic:a/my%20topic"},
		{name: "tab and newline", filter: "a\tb\n", want: "topic:a%09b%0A"},
		{name: "non-ASCII", filter: "a/é", want: "topic:a/%C3%A9"},
		{name: "delete character", filter: "a\x7f", want: "topic:a%7F"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fga.EncodeTopic(tc.filter)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Regexp(t, openFGAObject, got)
			assert.Regexp(t, openFGAPattern, got)
		})
	}
}

func TestEncodeTopic_Invalid(t *testing.T) {
	maxFilter := strings.Repeat("a", fga.MaxObjectLength-len("topic:"))
	_, err := fga.EncodeTopic(maxFilter)
	require.NoError(t, err)

	_, err = fga.EncodeTopic(maxFilter + "a")
	require.ErrorIs(t, err, fga.ErrInvalidObject)

	_, err = fga.EncodeTopic(strings.Repeat("#", 84))
	require.ErrorIs(t, err, fga.ErrInvalidObject, "escaping counts towards the limit")

	_, err = fga.EncodeTopic("")
	require.ErrorIs(t, err, fga.ErrInvalidObject)
}

func TestDecodeTopic(t *testing.T) {
	testCases := []struct {
		name   string
		object string
		want   string
	}{
		{name: "multi-level wildcard", object: "topic:T/devices/%23", want: "T/devices/#"},
		{name: "literal percent", object: "topic:a/100%25", want: "a/100%"},
		{name: "literal encoded sequence", object: "topic:a/%2523", want: "a/%23"},
		{name: "colon", object: "topic:urn%3Ax", want: "urn:x"},
		{name: "lowercase hex", object: "topic:urn%3ax", want: "urn:x"},
		{name: "non-ASCII", object: "topic:a/%C3%A9", want: "a/é"},
		{name: "without prefix", object: "T/%23", want: "T/#"},
		{name: "not hex unchanged", object: "topic:a/%G1", want: "a/%G1"},
		{name: "trailing percent", object: "topic:a/%", want: "a/%"},
		{name: "truncated sequence", object: "topic:a/%2", want: "a/%2"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fga.DecodeTopic(tc.object))
		})
	}
}

func FuzzRoundTrip(f *testing.F) {
	for _, seed := range []string{"T/devices/#", "a/%23", "%%23#", "%25#%", "urn:x y", "é\x00\x7f"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, filter string) {
		object, err := fga.EncodeTopic(filter)
		if err != nil {
			return
		}
		require.Regexp(t, openFGAObject, object)
		require.Regexp(t, openFGAPattern, object)
		require.Equal(t, filter, fga.DecodeTopic(object))
	})
}
