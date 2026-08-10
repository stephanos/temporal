package gomadlog_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/temporalio/gomad/internal/gomadlog"
)

func TestUnmarshalLog(t *testing.T) {
	testcases := []struct {
		json     string
		expected *gomadlog.Log
	}{
		{
			json: `{
				"msg": "hello",
				"extra": "1",
				"extra2": "2"
			}`,
			expected: &gomadlog.Log{
				Msg: "hello",
				Unknown: []gomadlog.UnknownField{
					{
						Key:   "extra",
						Value: `"1"`,
					},
					{
						Key:   "extra2",
						Value: `"2"`,
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		var got gomadlog.Log
		if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(&got, tc.expected); diff != "" {
			t.Errorf("unexpected diff unmarshaling %q: %s", tc.json, diff)
		}
	}
}
