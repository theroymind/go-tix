// Package timestamp provides the RFC 3339 timestamps, with seconds and an explicit offset, that contracts carry, and formats new ones in UTC.
package timestamp

import (
	"regexp"
	"time"

	"go.yaml.in/yaml/v3"
)

type Timestamp string

var expression = regexp.MustCompile(pattern)

func Format(moment time.Time) Timestamp {
	return Timestamp(moment.UTC().Format(Layout))
}

func (timestamp Timestamp) IsValid() bool {
	if !expression.MatchString(string(timestamp)) {
		return false
	}
	_, err := time.Parse(time.RFC3339, string(timestamp))
	return err == nil
}

// Time parses the timestamp, returning time.Parse's error when it is not RFC 3339.
func (timestamp Timestamp) Time() (time.Time, error) {
	return time.Parse(time.RFC3339, string(timestamp))
}

// MarshalYAML tags the value as a timestamp so it is written unquoted, as tix v1 files have it.
func (timestamp Timestamp) MarshalYAML() (any, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: yamlTag, Value: string(timestamp)}, nil
}
