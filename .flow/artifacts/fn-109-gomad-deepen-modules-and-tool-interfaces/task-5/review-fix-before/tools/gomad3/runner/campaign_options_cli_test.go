package runner

import "testing"

func TestParseSingleBaseSeedSharesRunnerStrategyRule(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "7", want: "7"},
		{input: "7-7", want: "7"},
		{input: "7,8"},
		{input: "bad"},
	} {
		selection, err := ParseSingleBaseSeed(test.input)
		if test.want == "" {
			if err == nil {
				t.Fatalf("ParseSingleBaseSeed(%q) = %q, want error", test.input, selection.String())
			}
			continue
		}
		if err != nil || selection.String() != test.want {
			t.Fatalf("ParseSingleBaseSeed(%q) = %q, %v; want %q", test.input, selection.String(), err, test.want)
		}
	}
}

func TestCampaignOptionNormalizationAndValidation(t *testing.T) {
	if got := NormalizeStrategy(""); got != StrategySeed {
		t.Fatalf("NormalizeStrategy(empty) = %q", got)
	}
	if got := NormalizeCoverage("", true); got != CoverageSemantic {
		t.Fatalf("NormalizeCoverage(empty, guided) = %q", got)
	}
	if got := NormalizeCoverage("", false); got != CoverageNone {
		t.Fatalf("NormalizeCoverage(empty, plain) = %q", got)
	}
	if got := campaignRequestFromSpec(CampaignSpec{Guide: true}).Coverage; got != "" {
		t.Fatalf("Runner changed absent guided coverage to %q", got)
	}
	for _, test := range []struct {
		mode     CoverageMode
		required []string
		valid    bool
	}{
		{mode: CoverageNone, valid: true},
		{mode: CoverageSemantic, required: []string{"stdlib.os.openfile"}, valid: true},
		{mode: CoverageNone, required: []string{"stdlib.os.openfile"}},
		{mode: CoverageSemantic, required: []string{"unknown.probe"}},
		{mode: "unknown"},
	} {
		if err := ValidateCoverage(test.mode, test.required); (err == nil) != test.valid {
			t.Fatalf("ValidateCoverage(%q, %v) = %v; valid=%t", test.mode, test.required, err, test.valid)
		}
	}
	for _, test := range []struct {
		limit uint64
		valid bool
	}{
		{limit: 0, valid: true},
		{limit: MinimumChoiceTraceBytes, valid: true},
		{limit: MaximumChoiceTraceBytes, valid: true},
		{limit: 1},
		{limit: MaximumChoiceTraceBytes + 1},
	} {
		if err := ValidateChoiceTraceLimit(test.limit); (err == nil) != test.valid {
			t.Fatalf("ValidateChoiceTraceLimit(%d) = %v; valid=%t", test.limit, err, test.valid)
		}
	}
	if err := ValidateChoiceCoverage(CoverageChoice, 0); err == nil {
		t.Fatal("choice coverage without a trace succeeded")
	}
	if err := ValidateChoiceCoverage(CoverageChoice, MinimumChoiceTraceBytes); err != nil {
		t.Fatal(err)
	}
	if err := ValidateChoiceCoverage(CoverageSemantic, 0); err != nil {
		t.Fatal(err)
	}
}
