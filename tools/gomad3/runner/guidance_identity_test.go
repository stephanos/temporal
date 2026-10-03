package runner

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

func TestGuidanceReopensOnlyWithMatchingEnvironmentAndTickPolicy(t *testing.T) {
	for _, coverage := range []CoverageMode{CoverageSemantic, CoverageChoice} {
		for _, test := range []struct {
			name, firstEnv, secondEnv, firstTick, secondTick string
			match                                            bool
		}{
			{name: "same configuration across seeds", firstEnv: "MODE=first", secondEnv: "MODE=first", match: true},
			{name: "changed environment", firstEnv: "MODE=first", secondEnv: "MODE=second"},
			{name: "strict to forward", secondTick: record.ClockTickForward},
			{name: "forward to strict", firstTick: record.ClockTickForward},
		} {
			t.Run(string(coverage)+"/"+test.name, func(t *testing.T) {
				preparer := newFakePreparer(t)
				limit := choiceTraceLimit(t, 1)
				executor := &fakeExecutor{result: func(uint64) execution.Result {
					result := processResult(0, "", "")
					result.IOTranscript = completeEmptyTranscript()
					if coverage == CoverageChoice {
						result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2}})
					}
					return result
				}}
				corpus := filepath.Join(t.TempDir(), "corpus")
				run := func(seed, env, tick string) error {
					config, configDependencies := testConfig(t, preparer, executor, seed, PolicyAll, 1)
					config.Coverage = coverage
					config.Guide = true
					config.Corpus = corpus
					config.Replayer = &matchingReplayer{}
					config.ClockTick = tick
					if coverage == CoverageChoice {
						config.ChoiceTraceLimit = limit
					}
					if env != "" {
						config.Environment = []string{env}
					}
					_, err := exploreWith(context.Background(), config, configDependencies)
					return err
				}
				if err := run("7", test.firstEnv, test.firstTick); err != nil {
					t.Fatal(err)
				}
				err := run("8", test.secondEnv, test.secondTick)
				if test.match {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "guided corpus identity does not match") {
					t.Fatalf("changed configuration error = %v", err)
				}
			})
		}
	}
}
