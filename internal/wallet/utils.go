package wallet

import (
	"encoding/json"
)

// ParseCustomInstructions decodes the JSON custom instructions from an output.
func ParseCustomInstructions(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
