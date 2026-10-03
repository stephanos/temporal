package runner

import (
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestValidateConfigRecordsOnlyTheForwardClockTick(t *testing.T) {
	for _, test := range []struct {
		clockTick string
		want      bool
	}{
		{clockTick: "", want: false},
		{clockTick: record.ClockTickStrict, want: false},
		{clockTick: record.ClockTickForward, want: true},
	} {
		config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		config.ClockTick = test.clockTick
		_, environment, err := validateCampaignRequest(campaignRequestFromSpec(config))
		if err != nil {
			t.Fatalf("validateConfig(ClockTick=%q) = %v", test.clockTick, err)
		}
		recorded := slices.Contains(environment, record.Environment{Name: record.ClockTickEnvironment, Value: record.ClockTickForward})
		if recorded != test.want {
			t.Fatalf("validateConfig(ClockTick=%q) environment = %v", test.clockTick, environment)
		}
		if !slices.IsSortedFunc(environment, func(a, b record.Environment) int { return strings.Compare(a.Name, b.Name) }) {
			t.Fatalf("environment is not sorted: %v", environment)
		}
	}

	config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	config.ClockTick = "sometimes"
	if _, _, err := validateCampaignRequest(campaignRequestFromSpec(config)); err == nil || !strings.Contains(err.Error(), "must be strict or forward") {
		t.Fatalf("validateConfig(ClockTick=sometimes) error = %v", err)
	}

	config, _ = testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	config.Environment = append(config.Environment, record.ClockTickEnvironment+"=forward")
	if _, _, err := validateCampaignRequest(campaignRequestFromSpec(config)); err == nil {
		t.Fatal("validateConfig() accepted the clock tick policy as a target --env entry")
	}
}
