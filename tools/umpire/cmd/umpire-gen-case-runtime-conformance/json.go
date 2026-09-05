package main

import "encoding/json"

func marshalExpected(expected expectedResult) ([]byte, error) {
	encoded, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
