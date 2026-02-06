package tools

import (
	"encoding/json"
	"strings"
)

// parseJSON is a helper function to parse JSON arguments.
func parseJSON(jsonStr string, v interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
