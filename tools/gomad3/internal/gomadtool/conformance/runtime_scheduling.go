package conformance

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func (campaign *runtimeCampaign) requireSchedulingBehavior(binaries map[string]string) error {
	if err := campaign.requireSearchReproduction(binaries); err != nil {
		return err
	}
	for _, padding := range []string{"", "invalid", "4194305"} {
		if err := campaign.expectedExit(
			"layout-invalid-padding-"+padding, []string{binaries["scheduler"], "-gomad-address-padding=" + padding}, campaign.testdata, 10*time.Second, 2,
			func(result hostexec.Result) error {
				if len(result.Stdout.RawBytes) != 0 || !strings.Contains(string(result.Stderr.RawBytes), "-gomad-address-padding must be a decimal byte count up to 4194304") {
					return fmt.Errorf("invalid address padding %q emitted an unexpected diagnostic", padding)
				}
				return nil
			},
			[]string{"GODEBUG", "GOGC", "GOMADSEED", "GOMAXPROCS"},
			"GODEBUG=asyncpreemptoff=1", "GOGC=off", "GOMADSEED=1", "GOMAXPROCS=1",
		); err != nil {
			return err
		}
	}
	for _, packageName := range []string{"./scheduler", "./select", "./maps", "./channels", "./sync", "./runqueue"} {
		if err := campaign.requireAddressPerturbation(packageName); err != nil {
			return err
		}
	}
	if err := campaign.requireHostLoad(binaries); err != nil {
		return err
	}
	if err := campaign.requireMapFamilies(); err != nil {
		return err
	}
	for packageName, marker := range map[string]string{
		"./select": "select-oracle:ok", "./channels": "channels-oracle:ok", "./maps": "maps-oracle:ok", "./sync": "sync-oracle:ok",
	} {
		output, err := campaign.runEnabled("1", packageName, "semantic-oracle", 0)
		if err != nil {
			return err
		}
		if strings.Count(output, marker) != 1 {
			return fmt.Errorf("%s did not report its semantic oracle", packageName)
		}
	}
	for _, seed := range []string{"0", "1", "18446744073709551615"} {
		output, err := campaign.runEnabled(seed, "./activation", "enabled", 0)
		if err != nil {
			return err
		}
		if output != "init GOMAXPROCS=1\nmain GOMAXPROCS=1" {
			return fmt.Errorf("enabled activation seed %s output = %q", seed, output)
		}
	}
	disabled, err := campaign.command(
		"activation-explicit-disabled", []string{binaries["activation"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOMAD3_IO_PROFILE", "GODEBUG", "GOMAXPROCS"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "init GOMAXPROCS=1\nmain GOMAXPROCS=1", "explicitly disabled activation"); err != nil {
		return errors.New("supporting runtime settings activated Gomad without GOMADSEED")
	}
	var disabledScheduler string
	for iteration := 1; iteration <= 32; iteration++ {
		result, err := campaign.command(
			fmt.Sprintf("scheduler-min-explicit-disabled-%d", iteration), []string{binaries["scheduler-min"]}, campaign.testdata, 10*time.Second,
			[]string{"GOMADSEED", "GOMAD3_IO_PROFILE", "GODEBUG", "GOMAXPROCS"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1",
		)
		if err != nil {
			return err
		}
		output := commandOutput(result)
		if iteration == 1 {
			disabledScheduler = output
		} else if output != disabledScheduler {
			return errors.New("supporting runtime settings activated scheduler randomization without GOMADSEED")
		}
	}
	disabledIO, err := campaign.command(
		"activation-io-disabled", []string{binaries["activation-io"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOMAD3_IO_PROFILE"},
	)
	if err != nil {
		return err
	}
	seededIO, err := campaign.command(
		"activation-io-direct", []string{binaries["activation-io"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMAD3_IO_PROFILE", "GOMADSEED"}, "GOMADSEED=1",
	)
	if err != nil {
		return err
	}
	if commandOutput(seededIO) != "gomad-host" || commandOutput(seededIO) == commandOutput(disabledIO) {
		return errors.New("direct GOMADSEED activation did not select the deterministic boundary explicitly")
	}
	if err := campaign.expectedExit(
		"activation-missing-profile-configuration", []string{binaries["activation"]}, campaign.testdata, 10*time.Second, 2,
		func(result hostexec.Result) error {
			if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: missing Gomad bootstrap configuration" {
				return errors.New("profile activation without Runner configuration emitted an unexpected diagnostic")
			}
			return nil
		},
		[]string{"GOMADSEED", "GOMAD3_IO_PROFILE"}, "GOMAD3_IO_PROFILE=deterministic",
	); err != nil {
		return err
	}
	if err := campaign.requireBootstrapConsumer(binaries["activation"]); err != nil {
		return err
	}
	for _, invalid := range []struct{ seed, name string }{
		{seed: "", name: "empty"}, {seed: "+1", name: "signed-plus"}, {seed: "-1", name: "signed-minus"},
		{seed: " 1", name: "leading-whitespace"}, {seed: "1 ", name: "trailing-whitespace"},
		{seed: "invalid", name: "nondecimal"}, {seed: "0x1", name: "hexadecimal"},
		{seed: "18446744073709551616", name: "overflow"},
	} {
		seed := invalid.seed
		if err := campaign.expectedExit(
			"activation-invalid-seed-"+invalid.name, []string{binaries["activation"]}, campaign.testdata, 10*time.Second, 2,
			func(result hostexec.Result) error {
				if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: invalid GOMADSEED" {
					return fmt.Errorf("invalid GOMADSEED=%q reached user initialization or emitted an unexpected diagnostic", seed)
				}
				return nil
			},
			[]string{"GOMADSEED"}, "GOMADSEED="+seed,
		); err != nil {
			return err
		}
	}
	return nil
}

type selectShape struct {
	name        string
	readyAtPoll int
	outcomes    []string
}

type selectShapeEvidence struct {
	Name              string   `json:"name"`
	ReadyAtPoll       int      `json:"ready_at_poll"`
	SelectPoll        uint64   `json:"select_poll"`
	FewerThanTwoReady uint64   `json:"fewer_than_two_ready"`
	Executions        int      `json:"executions"`
	Outcomes          []string `json:"outcomes"`
	Deadlocks         []string `json:"deadlocks"`
	StopReason        string   `json:"stop_reason"`
}

type timerCallbackEvidence struct {
	Seed               string            `json:"seed"`
	Transcript         string            `json:"transcript"`
	CallbackIdentities map[string]string `json:"callback_identities"`
	ParentlessOrdinals map[string]uint64 `json:"parentless_ordinals"`
	CallbackSites      map[string]uint64 `json:"callback_sites"`
}

type searchReproduction struct {
	ToolchainBuildKey            string                  `json:"toolchain_build_key"`
	Seed                         string                  `json:"select_exploration_seed"`
	MaximumExecutionsPerShape    int                     `json:"maximum_executions_per_shape"`
	MaximumDecisionsPerExecution int                     `json:"maximum_decisions_per_execution"`
	Reduced                      bool                    `json:"reduced"`
	TimerCallbacks               []timerCallbackEvidence `json:"timer_callbacks"`
	TimerPrefixes                []timerPrefixEvidence   `json:"timer_prefixes"`
	CrossSeedPrefix              timerDivergenceEvidence `json:"cross_seed_prefix"`
	SelectShapes                 []selectShapeEvidence   `json:"select_shapes"`
}

func (campaign *runtimeCampaign) requireSearchReproduction(binaries map[string]string) error {
	identity, err := campaign.choiceIdentity(binaries["timer-callback-identity"])
	if err != nil {
		return err
	}
	evidence := searchReproduction{ToolchainBuildKey: identity.ToolchainBuildKey, Seed: "1", MaximumExecutionsPerShape: 2048, MaximumDecisionsPerExecution: 32}
	var timerBase choiceRun
	for seed := 1; seed <= 32; seed++ {
		value := strconv.Itoa(seed)
		run, err := campaign.runChoice("timer-callback-seed-"+value, binaries["timer-callback-identity"], value, nil)
		if err != nil {
			return err
		}
		if seed == 6 {
			timerBase = run
		}
		association, err := timerCallbackAssociation(value, run)
		if err != nil {
			return err
		}
		if len(association.CallbackIdentities) != 0 {
			evidence.TimerCallbacks = append(evidence.TimerCallbacks, association)
		}
	}
	if !oppositeTimerCallbackIdentities(evidence.TimerCallbacks) {
		return errors.New("timer callbacks did not change parentless identity under opposite firing orders")
	}
	prefixes, divergence, err := campaign.requireTimerPrefixes(binaries["timer-callback-identity"], timerBase, identity)
	if err != nil {
		return err
	}
	evidence.TimerPrefixes, evidence.CrossSeedPrefix = prefixes, divergence
	for _, shape := range selectReadinessShapes {
		result, err := campaign.exploreSelectShape(binaries["select-readiness"], shape)
		if err != nil {
			return err
		}
		evidence.SelectShapes = append(evidence.SelectShapes, result)
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(campaign.workspace, "search-reproduction.json"), append(data, '\n'), 0o600)
}

func timerCallbackAssociation(seed string, run choiceRun) (timerCallbackEvidence, error) {
	labels := strings.Fields(run.transcript)
	if len(labels) != 2 || labels[0] == labels[1] || !slices.Contains(labels, "A") || !slices.Contains(labels, "B") {
		return timerCallbackEvidence{}, fmt.Errorf("timer callback output = %q", run.transcript)
	}
	evidence := timerCallbackEvidence{Seed: seed, Transcript: run.transcript, CallbackIdentities: map[string]string{}, ParentlessOrdinals: map[string]uint64{}, CallbackSites: map[string]uint64{}}
	ordinals := map[[sha256.Size]byte]uint64{}
	index := 0
	for recordIndex, record := range run.trace.Records {
		if record.Kind != choice.KindSelectPoll {
			continue
		}
		if index >= len(labels) {
			return evidence, errors.New("timer callbacks emitted more than two select polls")
		}
		label := labels[index]
		index++
		evidence.CallbackSites[label] = record.SiteOffset
		if recordIndex == 0 || run.trace.Records[recordIndex-1].Kind != choice.KindRunnable {
			continue
		}
		runnable := run.trace.Records[recordIndex-1]
		if index == 1 && runnable.Alternatives == 2 {
			// The virtual clock advanced only after all goroutines blocked. These
			// two alternatives are the callbacks just created by the two due timers.
			for ordinal := uint64(1); ordinal < 64; ordinal++ {
				first, second := timerParentlessIdentity(ordinal), timerParentlessIdentity(ordinal+1)
				digest, err := choice.AlternativeSetDigest([][sha256.Size]byte{first, second})
				if err != nil {
					return evidence, err
				}
				if digest == runnable.AlternativeSetDigest {
					ordinals[first], ordinals[second] = ordinal, ordinal+1
					break
				}
			}
		}
		if ordinal, ok := ordinals[runnable.SelectedIdentity]; ok {
			evidence.CallbackIdentities[label] = fmt.Sprintf("%x", runnable.SelectedIdentity)
			evidence.ParentlessOrdinals[label] = ordinal
		}
	}

	if index != 2 || evidence.CallbackSites["A"] == evidence.CallbackSites["B"] {
		return evidence, errors.New("timer callbacks did not expose two distinct select sites")
	}
	return evidence, nil
}

func timerParentlessIdentity(ordinal uint64) [sha256.Size]byte {
	input := append([]byte("gomad3-choice-goroutine-runtime/v1"), make([]byte, 8)...)
	binary.BigEndian.PutUint64(input[len(input)-8:], ordinal)
	return sha256.Sum256(input)
}

func oppositeTimerCallbackIdentities(runs []timerCallbackEvidence) bool {
	for _, first := range runs {
		for _, second := range runs {
			if len(first.ParentlessOrdinals) == 2 && len(second.ParentlessOrdinals) == 2 && first.ParentlessOrdinals["A"] != first.ParentlessOrdinals["B"] && first.ParentlessOrdinals["A"] == second.ParentlessOrdinals["B"] && first.ParentlessOrdinals["B"] == second.ParentlessOrdinals["A"] {
				return true
			}
		}
	}
	return false
}

func (campaign *runtimeCampaign) exploreSelectShape(fixture string, shape selectShape) (selectShapeEvidence, error) {
	identity, err := campaign.choiceIdentity(fixture)
	if err != nil {
		return selectShapeEvidence{}, err
	}
	evidence := selectShapeEvidence{Name: shape.name, ReadyAtPoll: shape.readyAtPoll, Outcomes: []string{}, Deadlocks: []string{}}
	frontier := []*choice.ReplayPlan{nil}
	seen := map[[sha256.Size]byte]bool{}
	outcomes := map[string]bool{}
	for len(frontier) != 0 {
		if evidence.Executions == 2048 {
			return evidence, fmt.Errorf("select shape %s exceeded the 2048-execution bound before exhaustion", shape.name)
		}
		prefix := frontier[0]
		frontier = frontier[1:]
		mode := choice.ModeRecord
		if prefix != nil {
			mode = choice.ModePrefix
		}
		name := fmt.Sprintf("select-%s-execution-%04d", shape.name, evidence.Executions)
		run, err := campaign.runChoiceMode(name, fixture, "1", prefix, mode, 0, shape.name)
		if err != nil {
			return evidence, err
		}
		evidence.Executions++
		if run.trace.Summary.SelectPoll != 1 || run.trace.Summary.SelectResult != 1 {
			return evidence, fmt.Errorf("select shape %s recorded poll/result counts %+v, want one each", shape.name, run.trace.Summary)
		}
		if evidence.Executions == 1 {
			evidence.SelectPoll = run.trace.Summary.SelectPoll
			if shape.readyAtPoll < 2 {
				evidence.FewerThanTwoReady = evidence.SelectPoll
			}
		}
		if !slices.Contains(shape.outcomes, run.transcript) {
			return evidence, fmt.Errorf("select shape %s outcome = %q", shape.name, run.transcript)
		}
		outcomes[run.transcript] = true
		plan, err := choice.ProjectReplayPlan(run.trace, identity)
		if err != nil {
			return evidence, err
		}
		if len(plan.Decisions) > 32 {
			return evidence, fmt.Errorf("select shape %s exceeded the 32-decision bound", shape.name)
		}
		for ordinal, decision := range plan.Decisions {
			for rank := uint32(0); rank < decision.Alternatives; rank++ {
				if rank == decision.Selected {
					continue
				}
				candidate, err := choice.BuildRankPrefix(plan, uint64(ordinal), rank)
				if err != nil {
					return evidence, err
				}
				if !seen[candidate.SHA256] {
					seen[candidate.SHA256] = true
					frontier = append(frontier, &candidate)
				}
			}
		}
	}
	for outcome := range outcomes {
		evidence.Outcomes = append(evidence.Outcomes, outcome)
	}
	slices.Sort(evidence.Outcomes)
	if !slices.Equal(evidence.Outcomes, shape.outcomes) {
		return evidence, fmt.Errorf("select shape %s outcomes = %v, want %v", shape.name, evidence.Outcomes, shape.outcomes)
	}
	evidence.StopReason = "frontier_exhausted"
	return evidence, nil
}

type timerPrefixEvidence struct {
	DecisionOrdinal uint64                `json:"decision_ordinal"`
	SelectedRank    uint32                `json:"selected_rank"`
	PrefixSHA256    string                `json:"prefix_sha256"`
	Association     timerCallbackEvidence `json:"association"`
}

type timerDivergenceEvidence struct {
	RecordedSeed    string `json:"recorded_seed"`
	ExecutedSeed    string `json:"executed_seed"`
	PrefixSHA256    string `json:"prefix_sha256"`
	DecisionOrdinal uint64 `json:"decision_ordinal"`
	Reason          string `json:"reason"`
}

func (campaign *runtimeCampaign) requireTimerPrefixes(fixture string, base choiceRun, identity choice.ExecutionIdentity) ([]timerPrefixEvidence, timerDivergenceEvidence, error) {
	plan, err := choice.ProjectReplayPlan(base.trace, identity)
	if err != nil {
		return nil, timerDivergenceEvidence{}, err
	}
	prefixes := []timerPrefixEvidence{}
	baseAssociation, err := timerCallbackAssociation("6", base)
	if err != nil {
		return nil, timerDivergenceEvidence{}, err
	}
	associations := []timerCallbackEvidence{baseAssociation}
	for ordinal, decision := range plan.Decisions {
		for rank := uint32(0); rank < decision.Alternatives; rank++ {
			if rank == decision.Selected {
				continue
			}
			prefix, err := choice.BuildRankPrefix(plan, uint64(ordinal), rank)
			if err != nil {
				return prefixes, timerDivergenceEvidence{}, err
			}
			name := fmt.Sprintf("timer-prefix-same-seed-6-decision-%d-rank-%d", ordinal, rank)
			run, err := campaign.runChoiceMode(name, fixture, "6", &prefix, choice.ModePrefix, 0)
			if err != nil {
				return prefixes, timerDivergenceEvidence{}, err
			}
			association, err := timerCallbackAssociation("6", run)
			if err != nil {
				return prefixes, timerDivergenceEvidence{}, err
			}
			associations = append(associations, association)
			prefixes = append(prefixes, timerPrefixEvidence{DecisionOrdinal: uint64(ordinal), SelectedRank: rank, PrefixSHA256: fmt.Sprintf("%x", prefix.SHA256), Association: association})
		}
	}
	if !oppositeTimerCallbackIdentities(associations) {
		return prefixes, timerDivergenceEvidence{}, errors.New("same-seed timer prefixes did not swap callback identities")
	}
	run, err := campaign.runChoiceMode("timer-prefix-cross-seed-6-to-16", fixture, "16", &plan, choice.ModePrefix, 125)
	if err != nil {
		return prefixes, timerDivergenceEvidence{}, err
	}
	reason := choice.DivergenceReason(run.terminal[13])
	if reason != choice.DivergenceSite {
		return prefixes, timerDivergenceEvidence{}, fmt.Errorf("cross-seed timer prefix reason = %s, want site", choice.DivergenceReasonName(reason))
	}
	divergence := timerDivergenceEvidence{RecordedSeed: "6", ExecutedSeed: "16", PrefixSHA256: fmt.Sprintf("%x", plan.SHA256), DecisionOrdinal: binary.BigEndian.Uint64(run.terminal[72:80]), Reason: choice.DivergenceReasonName(reason)}
	return prefixes, divergence, nil
}
