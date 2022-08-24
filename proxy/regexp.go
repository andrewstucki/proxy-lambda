package proxy

import (
	"regexp"
	"strings"
)

// Regexp adds unmarshalling from json for regexp.Regexp
type Regexp struct {
	*regexp.Regexp
}

// UnmarshalJSON unmarshals json into a regexp.Regexp
func (r *Regexp) UnmarshalJSON(b []byte) error {
	value := strings.TrimPrefix(strings.TrimSuffix(string(b), "\""), "\"")
	regex, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	r.Regexp = regex

	return nil
}

// MarshalJSON marshals regexp.Regexp as string
func (r *Regexp) MarshalJSON() ([]byte, error) {
	if r.Regexp != nil {
		return []byte(r.Regexp.String()), nil
	}

	return nil, nil
}
