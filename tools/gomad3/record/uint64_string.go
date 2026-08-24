package record

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Uint64String uint64

func (value Uint64String) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatUint(uint64(value), 10))), nil
}

func (value *Uint64String) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("decode decimal string into nil destination")
	}
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("decimal integer must be a JSON string")
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("decode decimal string: %w", err)
	}
	if text == "" || len(text) > 1 && text[0] == '0' {
		return fmt.Errorf("invalid canonical decimal string %q", text)
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return fmt.Errorf("invalid canonical decimal string %q", text)
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return fmt.Errorf("decimal string out of range: %w", err)
	}
	*value = Uint64String(parsed)
	return nil
}
