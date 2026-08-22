package regression

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.temporal.io/server/tests/umpire3/execution"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	"go.temporal.io/server/tests/umpire3/scenario"
)

type TestingT interface {
	Helper()
	Name() string
	Fatalf(string, ...any)
}

type CompilerLimits = scenario.Limits

type Corpus interface {
	Save(context.Context, protocolexperiment.Experiment, execution.Result) (string, error)
}

type Option func(*options)

type options struct {
	context         context.Context
	environment     execution.Factory
	compilerLimits  scenario.Limits
	runtimeLimits   execution.Limits
	corpus          Corpus
	expectedClaim   execution.ClaimKind
	expectedOutcome execution.OutcomeKind
}

func WithEnvironment(factory execution.Factory) Option {
	return func(options *options) {
		options.environment = factory
	}
}

func WithCompilerLimits(limits CompilerLimits) Option {
	return func(options *options) {
		options.compilerLimits = limits
	}
}

func WithRuntimeLimits(limits execution.Limits) Option {
	return func(options *options) {
		options.runtimeLimits = limits
	}
}

func WithCorpus(corpus Corpus) Option {
	return func(options *options) {
		options.corpus = corpus
	}
}

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.context = ctx
	}
}

func ExpectViolation() Option {
	return func(options *options) {
		options.expectedClaim = execution.ClaimViolating
		options.expectedOutcome = execution.OutcomeFlagged
	}
}

func RequireRegression(t TestingT, authored scenario.Scenario, optionValues ...Option) {
	t.Helper()
	configuration := options{
		context:       context.Background(),
		expectedClaim: execution.ClaimConforming,
		compilerLimits: scenario.Limits{
			MaxPaths: 128, MaxActions: 256, MaxStates: 100000,
			MaxMemoryBytes: 64 << 20, MaxTime: 10 * time.Second,
		},
	}
	for _, option := range optionValues {
		option(&configuration)
	}
	suite, err := scenario.Compile(configuration.context, authored, configuration.compilerLimits)
	if err != nil {
		t.Fatalf("Umpire3 scenario compilation failed: %v", err)
		return
	}
	if configuration.environment == nil {
		//nolint:revive // TestingT deliberately exposes the source-locating Fatalf seam only.
		t.Fatalf("Umpire3 regression requires an execution factory")
		return
	}
	for index, experiment := range suite.Experiments {
		result, runErr := execution.Run(configuration.context, execution.Request{
			Experiment: experiment, Environment: configuration.environment, Limits: configuration.runtimeLimits,
		})
		if runErr != nil {
			t.Fatalf("Umpire3 regression execution failed: %v", runErr)
			return
		}
		if result.Claim.Kind == configuration.expectedClaim &&
			(configuration.expectedOutcome == "" || result.Outcome.Kind == configuration.expectedOutcome) {
			continue
		}
		artifactPath := "not retained"
		if configuration.corpus != nil {
			path, saveErr := configuration.corpus.Save(configuration.context, experiment, result)
			if saveErr != nil {
				artifactPath = "retention failed: " + saveErr.Error()
			} else {
				artifactPath = path
			}
		}
		path := []string(nil)
		if index < len(suite.Explain.Paths) {
			path = suite.Explain.Paths[index]
		}
		t.Fatalf("Umpire3 regression failed\nclaim: %s\nreason: %s\npath: %v\n"+
			"grounded bindings: %v\nomissions: %v\ncleanup: %+v\nartifact: %s\nreplay: go test -run '^%s$'",
			result.Claim.Kind, result.Claim.Reason, path, result.Bindings, result.Omissions,
			result.Cleanup, artifactPath, regexp.QuoteMeta(t.Name()))
		return
	}
}

func Explain(authored scenario.Scenario, limits CompilerLimits) (scenario.Explain, error) {
	suite, err := scenario.Compile(context.Background(), authored, limits)
	if err != nil {
		return scenario.Explain{}, fmt.Errorf("compile Umpire3 scenario: %w", err)
	}
	return suite.Explain, nil
}
