// Package fga encodes MQTT topic filters as OpenFGA object IDs.
//
// OpenFGA object IDs must not contain '#', ':' or whitespace, and an object
// ("topic:<id>") is at most 256 characters. The filter is therefore
// percent-encoded: '%', '#', ':' and every byte outside printable ASCII
// become "%XX" (uppercase hex). The object type is "topic".
package fga

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// TopicType is the OpenFGA type of topic objects.
	TopicType = "topic"

	// MaxObjectLength is the maximum length of an OpenFGA object, type
	// included.
	MaxObjectLength = 256
)

// ErrInvalidObject is returned when a filter can't be encoded as an OpenFGA
// object: it is empty, or its encoding exceeds MaxObjectLength.
var ErrInvalidObject = errors.New("cannot encode topic filter as an OpenFGA object")

const hexDigits = "0123456789ABCDEF"

// EncodeTopic returns the OpenFGA object of a topic filter, for example
// "topic:T/devices/%23" for "T/devices/#".
func EncodeTopic(filter string) (string, error) {
	if filter == "" {
		return "", fmt.Errorf("%w: empty filter", ErrInvalidObject)
	}
	var b strings.Builder
	b.Grow(len(TopicType) + 1 + len(filter))
	b.WriteString(TopicType + ":")
	for i := 0; i < len(filter); i++ {
		c := filter[i]
		if needsEscape(c) {
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
			continue
		}
		b.WriteByte(c)
	}
	if b.Len() > MaxObjectLength {
		return "", fmt.Errorf("%w: %q encodes to %d characters, at most %d", ErrInvalidObject, filter, b.Len(), MaxObjectLength)
	}
	return b.String(), nil
}

// DecodeTopic returns the topic filter of an OpenFGA object produced by
// EncodeTopic. The "topic:" prefix is optional. A '%' not followed by two hex
// digits is left unchanged.
func DecodeTopic(object string) string {
	s := strings.TrimPrefix(object, TopicType+":")
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, okHi := unhex(s[i+1])
			lo, okLo := unhex(s[i+2])
			if okHi && okLo {
				b.WriteByte(hi<<4 | lo)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func needsEscape(c byte) bool {
	return c <= ' ' || c >= 0x7f || c == '%' || c == '#' || c == ':'
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}
