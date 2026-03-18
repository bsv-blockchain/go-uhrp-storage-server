package wallet

import (
	"encoding/json"
)

// ParseCustomInstructions decodes the JSON custom instructions from an output.
func ParseCustomInstructions(s string) map[string]string {
	if s == "" {
		return nil
	}
	result := make(map[string]string)
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}
