package data

import "encoding/json"

func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func unmarshalJSON(s string, v any) {
	if s == "" {
		return
	}
	json.Unmarshal([]byte(s), v)
}
