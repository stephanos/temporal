package sweep

import "testing"

func TestClassifyMethodRequiresSelectedMethodVerdict(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, selection, output string
		class                   Class
		skipped                 bool
	}{
		{"direct-method", "TestSuite/TestOne", "    --- PASS: TestSuite/TestOne (0.01s)\n--- PASS: TestSuite (0.01s)\n", ClassPass, false},
		{"nested-registration", "TestSuite/.*/TestOne", "        --- PASS: TestSuite/variant/TestOne (0.01s)\n", ClassPass, false},
		{"other-method", "TestSuite/TestOne", "    --- PASS: TestSuite/TestOther (0.01s)\n--- PASS: TestSuite (0.01s)\n", ClassInfrastructure, false},
		{"child-only", "TestSuite/TestOne", "        --- PASS: TestSuite/TestOne/case (0.01s)\n--- PASS: TestSuite (0.01s)\n", ClassInfrastructure, false},
		{"skipped-method", "TestSuite/TestOne", "    --- SKIP: TestSuite/TestOne (0.01s)\n--- PASS: TestSuite (0.01s)\n", ClassPass, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			outcome := classifyResult(test.selection, Event{Classification: "success"}, ArtifactFacts{Stdout: []byte(test.output)}, true)
			if outcome.Class != test.class || outcome.Skipped != test.skipped {
				t.Fatalf("outcome %+v, want class %s skipped %t", outcome, test.class, test.skipped)
			}
		})
	}
}
