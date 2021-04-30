package formatter

import (
	"encoding/json"
)

type jsonFormatter struct{}

// NewJSONFormatter formats output into json
func NewJSONFormatter() jsonFormatter {
	return jsonFormatter{}
}

// Format returns the Marshalled json output
func (f jsonFormatter) Format(data func() interface{}) (string, error) {
	out, err := json.MarshalIndent(data(), "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}
