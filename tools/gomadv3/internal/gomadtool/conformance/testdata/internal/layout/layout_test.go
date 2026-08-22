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

func TestParseArguments(t *testing.T) {
	for _, test := range []struct {
		name        string
		arguments   []string
		want        uint64
		wantPresent bool
		wantError   bool
	}{
		{name: "absent", arguments: []string{"mode"}},
		{name: "valid", arguments: []string{"mode", "-gomad-address-padding=1048576"}, want: 1048576, wantPresent: true},
		{name: "invalid", arguments: []string{"-gomad-address-padding=invalid"}, wantPresent: true, wantError: true},
		{name: "duplicate", arguments: []string{"-gomad-address-padding=0", "-gomad-address-padding=1"}, wantPresent: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, present, err := parseArguments(test.arguments)
			if got != test.want || present != test.wantPresent || (err != nil) != test.wantError {
				t.Fatalf("parseArguments() = (%d, %t, %v)", got, present, err)
			}
		})
	}
}
