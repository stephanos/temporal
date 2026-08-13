package layout

import "testing"

func TestParse(t *testing.T) {
	for _, value := range []string{"0", "1048576", "4194304"} {
		if _, err := parse(value); err != nil {
			t.Fatalf("parse(%q): %v", value, err)
		}
	}
	for _, value := range []string{"", "+1", "-1", " 1", "1 ", "invalid", "4194305"} {
		if _, err := parse(value); err == nil {
			t.Fatalf("parse(%q) succeeded", value)
		}
	}
}
