package conformance

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	name     string
	outcomes []string
	// readiness is what the runtime must record for the shape's select, on
	// its result and on every select-poll decision of every execution.
	readiness choice.SelectReadiness
	// completesLocked marks a select that takes a case in its first locked
	// pass: nothing runs between its last poll decision and its result but
	// the lock, the readiness count, and that pass, so the diagnostic trace
	// must show no allocation and no seeded draw across them.
	completesLocked bool
	// noOp marks a shape the explorer lists as a no-op
	// (choice.NoOpSelectShapes): exploring the shape with its select-poll
	// decisions left unexpanded must reach the outcomes and deadlocks of
	// exploring it in full. A shape with two ready cases skips nothing, so the
	// two explorations agree without proving anything.
	noOp bool
}

type selectShapeEvidence struct {
	Name              string                 `json:"name"`
	ReadyAtPoll       int                    `json:"ready_at_poll"`
	Readiness         choice.SelectReadiness `json:"readiness"`
	SelectPoll        uint64                 `json:"select_poll"`
	FewerThanTwoReady uint64                 `json:"fewer_than_two_ready"`
	Executions        int                    `json:"executions"`
	Outcomes          []string               `json:"outcomes"`
	Deadlocks         []string               `json:"deadlocks"`
	StopReason        string                 `json:"stop_reason"`
	// Reduced is the same exploration with every select-poll decision of a
	// known ready count below two left unexpanded, and Sound whether it
	// reached the outcomes and deadlocks above. The explorer may list a shape
	// only when its check is sound.
	Reduced selectReductionEvidence `json:"reduced"`
	// RecordingAllocations and RecordingDraws are what the first execution's
	// diagnostic trace counted between the select's last poll decision and
	// its result: heap objects allocated and seeded draws taken. A blocking
	// shape parks in between, so its numbers describe the schedule, not the
	// recording.
	RecordingAllocations uint64 `json:"recording_allocations"`
	RecordingDraws       uint64 `json:"recording_draws"`
}

type selectReductionEvidence struct {
	Executions          int      `json:"executions"`
	Outcomes            []string `json:"outcomes"`
	Deadlocks           []string `json:"deadlocks"`
	StopReason          string   `json:"stop_reason"`
	SkippedAlternatives uint64   `json:"skipped_alternatives"`
	Sound               bool     `json:"sound"`
}

// selectFrontier is one exhaustive exploration of a select shape: the
// executions it took, the outcomes and deadlocks it reached, and, when
// reduced, how many alternatives of no-op poll decisions it left unexpanded.
type selectFrontier struct {
	executions          int
	outcomes            []string
	deadlocks           []string
	skippedAlternatives uint64
}

// goroutineHandoff is the two-way Runnable decision immediately before a
// fixture's first marker select: the scheduler handed the P to the goroutine
// whose marker follows, so that goroutine's identity is the decision's selected
// identity and the two goroutines it chose between are its alternative set.
type goroutineHandoff struct {
	Mode           string `json:"mode,omitempty"`
	Seed           string `json:"seed"`
	Transcript     string `json:"transcript"`
	FirstLabel     string `json:"first_label"`
	FirstIdentity  string `json:"first_identity"`
	AlternativeSet string `json:"alternative_set"`
}

func (handoff goroutineHandoff) name() string {
	if handoff.Mode != "" {
		return handoff.Mode + " seed " + handoff.Seed
	}
	return "seed " + handoff.Seed
}

type timerCallbackEvidence struct {
	goroutineHandoff
	CallbackSites map[string]uint64 `json:"callback_sites"`
}

type searchReproduction struct {
	ToolchainBuildKey            string                  `json:"toolchain_build_key"`
	Seed                         string                  `json:"select_exploration_seed"`
	MaximumExecutionsPerShape    int                     `json:"maximum_executions_per_shape"`
	MaximumDecisionsPerExecution int                     `json:"maximum_decisions_per_execution"`
	TimerCallbacks               []timerCallbackEvidence `json:"timer_callbacks"`
	TimerPrefixes                []timerPrefixEvidence   `json:"timer_prefixes"`
	CrossSeedPrefix              timerCrossSeedEvidence  `json:"cross_seed_prefix"`
	TimerCreators                []goroutineHandoff      `json:"timer_creators"`
	TimerResets                  []goroutineHandoff      `json:"timer_resets"`
	SelectShapes                 []selectShapeEvidence   `json:"select_shapes"`
	RunqueueUserChoice           runqueueUserEvidence    `json:"runqueue_user_choice"`
}

func (campaign *runtimeCampaign) requireSearchReproduction(binaries map[string]string) error {
	identity, err := campaign.choiceIdentity(binaries["timer-callback-identity"])
	if err != nil {
		return err
	}
	evidence := searchReproduction{ToolchainBuildKey: identity.ToolchainBuildKey, Seed: "1", MaximumExecutionsPerShape: 2048, MaximumDecisionsPerExecution: 32}
	var timerBase choiceRun
	var handoffs []goroutineHandoff
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
		evidence.TimerCallbacks = append(evidence.TimerCallbacks, association)
		handoffs = append(handoffs, association.goroutineHandoff)
	}
	led, err := requireStableHandoffs(handoffs)
	if err != nil {
		return fmt.Errorf("timer callback identities: %w", err)
	}
	if led != 2 {
		return errors.New("timer callbacks fired in one order only across the recorded seeds")
	}
	prefixes, crossSeed, err := campaign.requireTimerPrefixes(binaries["timer-callback-identity"], timerBase, identity)
	if err != nil {
		return err
	}
	evidence.TimerPrefixes, evidence.CrossSeedPrefix = prefixes, crossSeed
	evidence.TimerCreators, err = campaign.requireTimerCreatorIdentities(binaries["timer-creator-identity"])
	if err != nil {
		return err
	}
	evidence.TimerResets, err = campaign.requireTimerResetIdentities(binaries["timer-reset-identity"])
	if err != nil {
		return err
	}
	if err := requireNoOpShapesListed(); err != nil {
		return err
	}
	for _, shape := range selectReadinessShapes {
		result, err := campaign.exploreSelectShape(binaries["select-readiness"], shape)
		if err != nil {
			return err
		}
		evidence.SelectShapes = append(evidence.SelectShapes, result)
	}
	evidence.RunqueueUserChoice, err = campaign.requireRunqueueUserChoice(binaries["runq-user-choice"])
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(campaign.workspace, "search-reproduction.json"), append(data, '\n'), 0o600)
}

// runqueueUserGoroutines bounds the user goroutines of the runq_user_choice
// modes that start workers: main and the two workers.
const runqueueUserGoroutines = 3

type runqueueUserEvidence struct {
	// TwoUsers and BusyRuntime list, per seed, the step order or summary the
	// mode printed and its Runnable decisions; Identities counts the distinct
	// goroutines those decisions selected across all seeds of the mode.
	TwoUsers              []runqueueUserRun `json:"two_users"`
	TwoUsersIdentities    int               `json:"two_users_identities"`
	BusyRuntime           []runqueueUserRun `json:"busy_runtime"`
	BusyRuntimeIdentities int               `json:"busy_runtime_identities"`
	// NoChoice lists the modes with at most one runnable user goroutine; each
	// recorded no decision and printed the same output under every seed.
	NoChoice     []runqueueUserRun `json:"no_choice"`
	ReplaySeed   string            `json:"replay_seed"`
	ReplayOutput string            `json:"replay_output"`
}

type runqueueUserRun struct {
	Mode      string `json:"mode"`
	Seed      string `json:"seed"`
	Output    string `json:"output"`
	Decisions int    `json:"decisions"`
}

// requireRunqueueUserChoice holds the run-queue choice to its rule: a
// runtime-owned goroutine in the local run queue runs first, in queue order,
// and only a queue of user goroutines records a decision, among those
// goroutines. The fixture's collector churn puts the sweeper, scavenger, mark
// workers, and finalizer goroutine in the queue beside the user goroutines.
// Under the old rule each of these checks fails: main-only recorded 26
// decisions for seed 11 that all had a runtime-owned alternative, and
// two-users recorded decisions with four and five alternatives.
func (campaign *runtimeCampaign) requireRunqueueUserChoice(binary string) (runqueueUserEvidence, error) {
	var evidence runqueueUserEvidence
	for _, mode := range []string{"main-only", "one-user"} {
		var first string
		for _, seed := range []string{"1", "1", "2", "3", "11", "17"} {
			run, _, err := campaign.runChoiceSpec(choiceRunSpec{
				name: fmt.Sprintf("runq-user-%s-seed-%s-%d", mode, seed, len(evidence.NoChoice)), fixture: binary, seed: seed, mode: choice.ModeRecord, args: []string{mode},
			})
			if err != nil {
				return evidence, err
			}
			decisions := runnableDecisions(run.trace)
			evidence.NoChoice = append(evidence.NoChoice, runqueueUserRun{Mode: mode, Seed: seed, Output: run.transcript, Decisions: len(decisions)})
			if len(decisions) != 0 {
				return evidence, fmt.Errorf("%s seed %s recorded %d Runnable decisions with at most one runnable user goroutine", mode, seed, len(decisions))
			}
			if first == "" {
				first = run.transcript
			} else if run.transcript != first {
				return evidence, campaign.repeatabilityMismatch(mode+" printed a schedule-dependent result without a decision", first, run.transcript)
			}
		}
		if first != mode+" done" {
			return evidence, fmt.Errorf("%s output = %q", mode, first)
		}
	}
	var err error
	var recorded choiceRun
	evidence.TwoUsers, evidence.TwoUsersIdentities, recorded, err = campaign.requireUserAlternatives(binary, "two-users", func(output string) error {
		order := strings.TrimPrefix(output, "two-users ")
		if len(order) != 8 || strings.Count(order, "a") != 4 || strings.Count(order, "b") != 4 {
			return fmt.Errorf("two-users output = %q", output)
		}
		return nil
	})
	if err != nil {
		return evidence, err
	}
	orders := map[string]bool{}
	for _, run := range evidence.TwoUsers {
		orders[run.Output] = true
	}
	if len(orders) < 2 {
		return evidence, errors.New("two-users printed one step order under every seed; the user choice no longer branches")
	}
	evidence.BusyRuntime, evidence.BusyRuntimeIdentities, _, err = campaign.requireUserAlternatives(binary, "busy-runtime", func(output string) error {
		if output != "busy-runtime a=32 b=32 a-saw-finalizers=true b-saw-finalizers=true" {
			return fmt.Errorf("busy-runtime starved a class: output = %q", output)
		}
		return nil
	})
	if err != nil {
		return evidence, err
	}
	// A tape of the user-only decisions forces the recorded order under a
	// seed that picks differently on its own.
	identity, err := campaign.choiceIdentity(binary)
	if err != nil {
		return evidence, err
	}
	plan, err := choice.ProjectReplayPlan(recorded.trace, identity)
	if err != nil {
		return evidence, fmt.Errorf("project two-users tape: %w", err)
	}
	for _, run := range evidence.TwoUsers[1:] {
		if run.Output == recorded.transcript {
			continue
		}
		replayed, err := campaign.runChoiceMode("runq-user-two-users-replay-seed-"+run.Seed, binary, run.Seed, &plan, choice.ModeReplay, 0, "two-users")
		if err != nil {
			return evidence, err
		}
		if replayed.transcript != recorded.transcript {
			return evidence, campaign.repeatabilityMismatch("two-users tape did not reproduce its order under seed "+run.Seed, recorded.transcript, replayed.transcript)
		}
		evidence.ReplaySeed, evidence.ReplayOutput = run.Seed, replayed.transcript
		break
	}
	if evidence.ReplaySeed == "" {
		return evidence, errors.New("no two-users seed picked differently from the recorded tape")
	}
	return evidence, nil
}

// requireUserAlternatives runs one mode under several seeds and requires every
// Runnable decision to choose among user goroutines only. The fixture cannot
// name its goroutines' identities, so they are taken from the decisions
// themselves: across all seeds the decisions may select at most
// runqueueUserGoroutines distinct goroutines, and each decision's alternative
// set must be a set of those selected goroutines. A runtime-owned alternative
// would either be selected somewhere, raising the count, or never be selected,
// leaving a set no combination of selected goroutines reproduces. It returns
// the first seed's run for a replay check.
func (campaign *runtimeCampaign) requireUserAlternatives(binary, mode string, check func(string) error) ([]runqueueUserRun, int, choiceRun, error) {
	var runs []runqueueUserRun
	var first choiceRun
	var decisions []choice.Record
	for seed := 1; seed <= 8; seed++ {
		value := strconv.Itoa(seed)
		run, err := campaign.runChoiceMode(fmt.Sprintf("runq-user-%s-seed-%s", mode, value), binary, value, nil, choice.ModeRecord, 0, mode)
		if err != nil {
			return runs, 0, first, err
		}
		if err := check(run.transcript); err != nil {
			return runs, 0, first, fmt.Errorf("seed %s: %w", value, err)
		}
		if seed == 1 {
			first = run
		}
		recorded := runnableDecisions(run.trace)
		runs = append(runs, runqueueUserRun{Mode: mode, Seed: value, Output: run.transcript, Decisions: len(recorded)})
		decisions = append(decisions, recorded...)
	}
	if len(decisions) == 0 {
		return runs, 0, first, fmt.Errorf("%s recorded no Runnable decision", mode)
	}
	var selected [][sha256.Size]byte
	for _, decision := range decisions {
		if !slices.Contains(selected, decision.SelectedIdentity) {
			selected = append(selected, decision.SelectedIdentity)
		}
	}
	if len(selected) > runqueueUserGoroutines {
		return runs, len(selected), first, fmt.Errorf("%s decisions selected %d distinct goroutines, more than its %d user goroutines", mode, len(selected), runqueueUserGoroutines)
	}
	for _, decision := range decisions {
		matched, err := alternativeSetOfSelected(decision, selected)
		if err != nil {
			return runs, len(selected), first, err
		}
		if !matched {
			return runs, len(selected), first, fmt.Errorf("%s decision %d chose among %d goroutines that are not all user goroutines", mode, decision.Ordinal, decision.Alternatives)
		}
	}
	return runs, len(selected), first, nil
}

func runnableDecisions(trace choice.Trace) []choice.Record {
	var decisions []choice.Record
	for _, record := range trace.Records {
		if record.Kind == choice.KindRunnable && record.Flags&choice.FlagDecision != 0 {
			decisions = append(decisions, record)
		}
	}
	return decisions
}

// alternativeSetOfSelected reports whether some subset of selected that holds
// the decision's selected goroutine has the decision's alternative-set digest.
func alternativeSetOfSelected(decision choice.Record, selected [][sha256.Size]byte) (bool, error) {
	for mask := 1; mask < 1<<len(selected); mask++ {
		var members [][sha256.Size]byte
		for index, identity := range selected {
			if mask&(1<<index) != 0 {
				members = append(members, identity)
			}
		}
		if len(members) != int(decision.Alternatives) || !slices.Contains(members, decision.SelectedIdentity) {
			continue
		}
		digest, err := choice.AlternativeSetDigest(members)
		if err != nil {
			return false, err
		}
		if digest == decision.AlternativeSetDigest {
			return true, nil
		}
	}
	return false, nil
}

func timerCallbackAssociation(seed string, run choiceRun) (timerCallbackEvidence, error) {
	labels := strings.Fields(run.transcript)
	if len(labels) != 2 || labels[0] == labels[1] || !slices.Contains(labels, "A") || !slices.Contains(labels, "B") {
		return timerCallbackEvidence{}, fmt.Errorf("timer callback output = %q", run.transcript)
	}
	evidence := timerCallbackEvidence{goroutineHandoff: goroutineHandoff{Seed: seed, Transcript: run.transcript}, CallbackSites: map[string]uint64{}}
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
		if index != 1 {
			continue
		}
		// The virtual clock advanced only after all goroutines blocked, so a
		// two-way decision here is between the callbacks the two due timers just
		// created. A missing or wider decision identifies nothing.
		if handoff, ok := markerHandoff(run.trace.Records, recordIndex); ok {
			evidence.FirstLabel = label
			evidence.FirstIdentity = fmt.Sprintf("%x", handoff.SelectedIdentity)
			evidence.AlternativeSet = fmt.Sprintf("%x", handoff.AlternativeSetDigest)
		}
	}

	if index != 2 || evidence.CallbackSites["A"] == evidence.CallbackSites["B"] {
		return evidence, errors.New("timer callbacks did not expose two distinct select sites")
	}
	return evidence, nil
}

func markerHandoff(records []choice.Record, marker int) (choice.Record, bool) {
	if marker == 0 || records[marker-1].Kind != choice.KindRunnable || records[marker-1].Alternatives != 2 {
		return choice.Record{}, false
	}
	return records[marker-1], true
}

// goroutineHandoffEvidence requires the fixture's first marker select to follow
// a two-way decision and labels the goroutine that ran it with the first word
// of the transcript, which must be a permutation of labels.
func goroutineHandoffEvidence(mode, seed string, run choiceRun, labels []string) (goroutineHandoff, error) {
	evidence := goroutineHandoff{Mode: mode, Seed: seed, Transcript: run.transcript}
	words := strings.Fields(run.transcript)
	sorted, want := slices.Clone(words), slices.Clone(labels)
	slices.Sort(sorted)
	slices.Sort(want)
	if len(words) != 2 || !slices.Equal(sorted, want) {
		return evidence, fmt.Errorf("%s output = %q", evidence.name(), run.transcript)
	}
	evidence.FirstLabel = words[0]
	marker := slices.IndexFunc(run.trace.Records, func(record choice.Record) bool { return record.Kind == choice.KindSelectPoll })
	if marker < 0 {
		return evidence, fmt.Errorf("%s recorded no marker select", evidence.name())
	}
	handoff, ok := markerHandoff(run.trace.Records, marker)
	if !ok {
		return evidence, fmt.Errorf("%s did not hand off through a two-way decision", evidence.name())
	}
	evidence.FirstIdentity = fmt.Sprintf("%x", handoff.SelectedIdentity)
	evidence.AlternativeSet = fmt.Sprintf("%x", handoff.AlternativeSetDigest)
	return evidence, nil
}

// requireStableHandoffs requires every run to decide between the same two
// goroutines and each label to lead with one identity wherever it leads. When
// both labels have led, their identities must differ and make up that
// alternative set, which fixes the identity of the goroutine that ran second
// as well. It reports how many labels led.
func requireStableHandoffs(runs []goroutineHandoff) (int, error) {
	if len(runs) == 0 {
		return 0, errors.New("no hand-offs to compare")
	}
	identities := map[string]string{}
	for _, run := range runs {
		if run.AlternativeSet == "" {
			return 0, fmt.Errorf("%s did not identify its hand-off", run.name())
		}
		if run.AlternativeSet != runs[0].AlternativeSet {
			return 0, fmt.Errorf("%s decided among alternative set %s, %s among %s", run.name(), run.AlternativeSet, runs[0].name(), runs[0].AlternativeSet)
		}
		if previous, led := identities[run.FirstLabel]; led && previous != run.FirstIdentity {
			return 0, fmt.Errorf("%s led %s with identity %s, earlier runs with %s", run.name(), run.FirstLabel, run.FirstIdentity, previous)
		}
		identities[run.FirstLabel] = run.FirstIdentity
	}
	if len(identities) == 2 {
		var members [][sha256.Size]byte
		for _, identity := range identities {
			decoded, err := hex.DecodeString(identity)
			if err != nil || len(decoded) != sha256.Size {
				return 0, fmt.Errorf("malformed goroutine identity %q", identity)
			}
			members = append(members, [sha256.Size]byte(decoded))
		}
		if members[0] == members[1] {
			return 0, errors.New("both labels led with the same identity")
		}
		digest, err := choice.AlternativeSetDigest(members)
		if err != nil {
			return 0, err
		}
		if fmt.Sprintf("%x", digest) != runs[0].AlternativeSet {
			return 0, errors.New("the two leading identities do not make up the alternative set")
		}
	}
	return len(identities), nil
}

// requireTimerCreatorIdentities runs the creator fixture in every mode under
// several seeds: the children started after a stopped timer, after a timer
// that never fires, and after no timer must be the same two goroutines.
func (campaign *runtimeCampaign) requireTimerCreatorIdentities(fixture string) ([]goroutineHandoff, error) {
	var runs []goroutineHandoff
	for _, mode := range []string{"none", "stopped", "pending"} {
		for seed := 1; seed <= 8; seed++ {
			value := strconv.Itoa(seed)
			run, err := campaign.runChoiceMode(fmt.Sprintf("timer-creator-%s-seed-%s", mode, value), fixture, value, nil, choice.ModeRecord, 0, mode)
			if err != nil {
				return runs, err
			}
			handoff, err := goroutineHandoffEvidence(mode, value, run, []string{"A", "B"})
			if err != nil {
				return runs, err
			}
			runs = append(runs, handoff)
		}
	}
	if _, err := requireStableHandoffs(runs); err != nil {
		return runs, fmt.Errorf("timer creator identities: %w", err)
	}
	return runs, nil
}

// requireTimerResetIdentities runs the reset fixture under several seeds. Its
// two callback goroutines of one timer are both runnable when main releases
// them, so a two-way decision between them exists only if their identities
// differ; a shared identity diverges the run instead of completing it.
func (campaign *runtimeCampaign) requireTimerResetIdentities(fixture string) ([]goroutineHandoff, error) {
	var runs []goroutineHandoff
	for seed := 1; seed <= 8; seed++ {
		value := strconv.Itoa(seed)
		run, err := campaign.runChoice("timer-reset-seed-"+value, fixture, value, nil)
		if err != nil {
			return runs, err
		}
		handoff, err := goroutineHandoffEvidence("reset", value, run, []string{"1", "2"})
		if err != nil {
			return runs, err
		}
		runs = append(runs, handoff)
	}
	if _, err := requireStableHandoffs(runs); err != nil {
		return runs, fmt.Errorf("timer reset identities: %w", err)
	}
	return runs, nil
}

// requireNoOpShapesListed requires the shapes this fixture marks as no-ops and
// the shapes the explorer leaves unexpanded to be the same set, so neither can
// gain a shape the other does not know. Every fixture select polls two cases.
func requireNoOpShapesListed() error {
	proven := []choice.NoOpSelectShape{}
	for _, shape := range selectReadinessShapes {
		if shape.noOp {
			proven = append(proven, choice.NoOpSelectShape{PolledCases: 2, Readiness: shape.readiness})
		}
	}
	listed := choice.NoOpSelectShapes()
	for _, shape := range listed {
		if !slices.Contains(proven, shape) {
			return fmt.Errorf("the explorer lists select shape %+v as a no-op, which no fixture shape proves", shape)
		}
	}
	for _, shape := range proven {
		if !slices.Contains(listed, shape) {
			return fmt.Errorf("the fixture proves select shape %+v, which the explorer does not list", shape)
		}
	}
	if len(listed) != len(proven) {
		return fmt.Errorf("the explorer lists %d no-op select shapes for %d proven", len(listed), len(proven))
	}
	return nil
}

// exploreSelectShape explores a shape to exhaustion twice, in full and with
// the no-op poll decisions left unexpanded, and records whether the reduced
// frontier reached the same outcomes and deadlocks. Only the full exploration
// checks the recording cost, from its first execution's diagnostic trace.
func (campaign *runtimeCampaign) exploreSelectShape(fixture string, shape selectShape) (selectShapeEvidence, error) {
	evidence := selectShapeEvidence{Name: shape.name, ReadyAtPoll: int(shape.readiness.Ready), Readiness: shape.readiness, SelectPoll: 1}
	if shape.readiness.Ready < 2 {
		evidence.FewerThanTwoReady = evidence.SelectPoll
	}
	full, err := campaign.exploreSelectFrontier(fixture, shape, false, &evidence)
	evidence.Executions, evidence.Outcomes, evidence.Deadlocks = full.executions, full.outcomes, full.deadlocks
	if err != nil {
		return evidence, err
	}
	if !slices.Equal(evidence.Outcomes, shape.outcomes) {
		return evidence, fmt.Errorf("select shape %s outcomes = %v, want %v", shape.name, evidence.Outcomes, shape.outcomes)
	}
	evidence.StopReason = "frontier_exhausted"
	reduced, err := campaign.exploreSelectFrontier(fixture, shape, true, nil)
	evidence.Reduced = selectReductionEvidence{Executions: reduced.executions, Outcomes: reduced.outcomes, Deadlocks: reduced.deadlocks, SkippedAlternatives: reduced.skippedAlternatives}
	if err != nil {
		return evidence, err
	}
	evidence.Reduced.StopReason = "frontier_exhausted"
	evidence.Reduced.Sound = slices.Equal(reduced.outcomes, full.outcomes) && slices.Equal(reduced.deadlocks, full.deadlocks)
	if shape.noOp && !evidence.Reduced.Sound {
		return evidence, fmt.Errorf("select shape %s is listed as a no-op but its reduced exploration reached %v, not %v", shape.name, reduced.outcomes, full.outcomes)
	}
	if shape.noOp && reduced.skippedAlternatives == 0 {
		return evidence, fmt.Errorf("select shape %s is listed as a no-op but its reduced exploration skipped nothing", shape.name)
	}
	return evidence, nil
}

// exploreSelectFrontier runs the shape under seed 1 and every non-selected
// rank of every decision it records, breadth first, until no unseen prefix is
// left. When reduced, a select-poll decision whose select recorded fewer than
// two ready cases is not expanded. Evidence, when given, receives the first
// execution's recording cost and is checked against the shape.
func (campaign *runtimeCampaign) exploreSelectFrontier(fixture string, shape selectShape, reduced bool, evidence *selectShapeEvidence) (selectFrontier, error) {
	identity, err := campaign.choiceIdentity(fixture)
	if err != nil {
		return selectFrontier{}, err
	}
	result := selectFrontier{outcomes: []string{}, deadlocks: []string{}}
	frontier := []*choice.ReplayPlan{nil}
	seen := map[[sha256.Size]byte]bool{}
	outcomes := map[string]bool{}
	label := shape.name
	if reduced {
		label += "-reduced"
	}
	for len(frontier) != 0 {
		if result.executions == 2048 {
			return result, fmt.Errorf("select shape %s exceeded the 2048-execution bound before exhaustion", label)
		}
		prefix := frontier[0]
		frontier = frontier[1:]
		mode := choice.ModeRecord
		if prefix != nil {
			mode = choice.ModePrefix
		}
		name := fmt.Sprintf("select-%s-execution-%04d", label, result.executions)
		run, _, err := campaign.runChoiceSpec(choiceRunSpec{name: name, fixture: fixture, seed: "1", tape: prefix, mode: mode, diagnostic: evidence != nil && result.executions == 0, args: []string{shape.name}})
		if err != nil {
			return result, err
		}
		result.executions++
		if run.trace.Summary.SelectPoll != 1 || run.trace.Summary.SelectResult != 1 {
			return result, fmt.Errorf("select shape %s recorded poll/result counts %+v, want one each", label, run.trace.Summary)
		}
		if evidence != nil && result.executions == 1 {
			evidence.RecordingAllocations, evidence.RecordingDraws, err = selectRecordingCost(run)
			if err != nil {
				return result, fmt.Errorf("select shape %s: %w", label, err)
			}
			if shape.completesLocked && (evidence.RecordingAllocations != 0 || evidence.RecordingDraws != 0) {
				return result, fmt.Errorf("select shape %s allocated %d objects and drew %d times between its last poll decision and its result", label, evidence.RecordingAllocations, evidence.RecordingDraws)
			}
		}
		if !slices.Contains(shape.outcomes, run.transcript) {
			return result, fmt.Errorf("select shape %s outcome = %q", label, run.transcript)
		}
		outcomes[run.transcript] = true
		plan, err := choice.ProjectReplayPlan(run.trace, identity)
		if err != nil {
			return result, err
		}
		if err := requireSelectReadiness(plan, shape.readiness); err != nil {
			return result, fmt.Errorf("select shape %s execution %d: %w", label, result.executions-1, err)
		}
		if len(plan.Decisions) > 32 {
			return result, fmt.Errorf("select shape %s exceeded the 32-decision bound", label)
		}
		for ordinal, decision := range plan.Decisions {
			if reduced && decision.Kind == choice.KindSelectPoll && plan.Readiness[ordinal].Known && plan.Readiness[ordinal].Ready < 2 {
				result.skippedAlternatives += uint64(decision.Alternatives - 1)
				continue
			}
			for rank := uint32(0); rank < decision.Alternatives; rank++ {
				if rank == decision.Selected {
					continue
				}
				candidate, err := choice.BuildRankPrefix(plan, uint64(ordinal), rank)
				if err != nil {
					return result, err
				}
				if !seen[candidate.SHA256] {
					seen[candidate.SHA256] = true
					frontier = append(frontier, &candidate)
				}
			}
		}
	}
	for outcome := range outcomes {
		result.outcomes = append(result.outcomes, outcome)
	}
	slices.Sort(result.outcomes)
	return result, nil
}

// requireSelectReadiness requires the projected plan to carry want on every
// select-poll decision, at least one of which exists, and nothing on any other.
func requireSelectReadiness(plan choice.ReplayPlan, want choice.SelectReadiness) error {
	if len(plan.Readiness) != len(plan.Decisions) {
		return fmt.Errorf("plan carries readiness for %d of %d decisions", len(plan.Readiness), len(plan.Decisions))
	}
	polls := 0
	for index, decision := range plan.Decisions {
		expected := choice.SelectReadiness{}
		if decision.Kind == choice.KindSelectPoll {
			expected = want
			polls++
		}
		if plan.Readiness[index] != expected {
			return fmt.Errorf("decision %d (%s) readiness = %+v, want %+v", index, choiceKindName(decision.Kind), plan.Readiness[index], expected)
		}
	}
	if polls == 0 {
		return errors.New("plan carries no select-poll decision")
	}
	return nil
}

func choiceKindName(kind choice.Kind) string {
	switch kind {
	case choice.KindRunnable:
		return "runnable"
	case choice.KindSelectPoll:
		return "select-poll"
	case choice.KindSelectResult:
		return "select-result"
	default:
		return "unknown"
	}
}

// selectRecordingCost reads the diagnostic digests at the run's single select
// result and at that select's last poll decision, which the result names
// through its origin and polled-case count, and returns how many objects were
// allocated and how many seeded draws were taken between the two records.
func selectRecordingCost(run choiceRun) (allocations, draws uint64, err error) {
	index := slices.IndexFunc(run.trace.Records, func(record choice.Record) bool { return record.Kind == choice.KindSelectResult })
	if index < 0 {
		return 0, 0, errors.New("trace records no select result")
	}
	result := run.trace.Records[index]
	if result.Data < 2 {
		return 0, 0, fmt.Errorf("select result polled %d cases, want at least two", result.Data)
	}
	last := result.Origin + uint64(result.Data) - 2
	if last >= result.Ordinal || result.Ordinal >= uint64(len(run.diagnostic.Records)) {
		return 0, 0, fmt.Errorf("select result %d names poll decision %d outside %d diagnostic records", result.Ordinal, last, len(run.diagnostic.Records))
	}
	before, after := run.diagnostic.Records[last], run.diagnostic.Records[result.Ordinal]
	draws = after.RunqDraws - before.RunqDraws + after.SchedulerDraws - before.SchedulerDraws + after.SelectDraws - before.SelectDraws +
		after.RuntimeRandDraws - before.RuntimeRandDraws + after.RuntimeCheapRandDraws - before.RuntimeCheapRandDraws + after.TimerDraws - before.TimerDraws + after.ClockTickDraws - before.ClockTickDraws
	return after.Allocations - before.Allocations, draws, nil
}

type timerPrefixEvidence struct {
	DecisionOrdinal uint64                `json:"decision_ordinal"`
	SelectedRank    uint32                `json:"selected_rank"`
	PrefixSHA256    string                `json:"prefix_sha256"`
	Association     timerCallbackEvidence `json:"association"`
}

// timerCrossSeedEvidence records the seed-6 plan executed under seed 16. Replay
// binds the recorded seed, so this is an experiment whose outcome is retained
// as evidence; it completes or diverges, and neither is a requirement.
type timerCrossSeedEvidence struct {
	RecordedSeed    string `json:"recorded_seed"`
	ExecutedSeed    string `json:"executed_seed"`
	PrefixSHA256    string `json:"prefix_sha256"`
	Exit            int    `json:"exit"`
	Transcript      string `json:"transcript"`
	DecisionOrdinal uint64 `json:"decision_ordinal"`
	Reason          string `json:"reason"`
}

func (campaign *runtimeCampaign) requireTimerPrefixes(fixture string, base choiceRun, identity choice.ExecutionIdentity) ([]timerPrefixEvidence, timerCrossSeedEvidence, error) {
	plan, err := choice.ProjectReplayPlan(base.trace, identity)
	if err != nil {
		return nil, timerCrossSeedEvidence{}, err
	}
	prefixes := []timerPrefixEvidence{}
	baseAssociation, err := timerCallbackAssociation("6", base)
	if err != nil {
		return nil, timerCrossSeedEvidence{}, err
	}
	handoffs := []goroutineHandoff{baseAssociation.goroutineHandoff}
	swapped := false
	for ordinal, decision := range plan.Decisions {
		for rank := uint32(0); rank < decision.Alternatives; rank++ {
			if rank == decision.Selected {
				continue
			}
			prefix, err := choice.BuildRankPrefix(plan, uint64(ordinal), rank)
			if err != nil {
				return prefixes, timerCrossSeedEvidence{}, err
			}
			name := fmt.Sprintf("timer-prefix-same-seed-6-decision-%d-rank-%d", ordinal, rank)
			run, err := campaign.runChoiceMode(name, fixture, "6", &prefix, choice.ModePrefix, 0)
			if err != nil {
				return prefixes, timerCrossSeedEvidence{}, err
			}
			association, err := timerCallbackAssociation("6", run)
			if err != nil {
				return prefixes, timerCrossSeedEvidence{}, err
			}
			handoffs = append(handoffs, association.goroutineHandoff)
			swapped = swapped || association.Transcript != baseAssociation.Transcript
			prefixes = append(prefixes, timerPrefixEvidence{DecisionOrdinal: uint64(ordinal), SelectedRank: rank, PrefixSHA256: fmt.Sprintf("%x", prefix.SHA256), Association: association})
		}
	}
	if _, err := requireStableHandoffs(handoffs); err != nil {
		return prefixes, timerCrossSeedEvidence{}, fmt.Errorf("same-seed timer prefixes: %w", err)
	}
	if !swapped {
		return prefixes, timerCrossSeedEvidence{}, errors.New("no same-seed timer prefix swapped the callbacks")
	}
	run, exit, err := campaign.runChoiceAccepting("timer-prefix-cross-seed-6-to-16", fixture, "16", &plan, choice.ModePrefix, 0, []int{125})
	if err != nil {
		return prefixes, timerCrossSeedEvidence{}, err
	}
	crossSeed := timerCrossSeedEvidence{RecordedSeed: "6", ExecutedSeed: "16", PrefixSHA256: fmt.Sprintf("%x", plan.SHA256), Exit: exit, Transcript: run.transcript}
	if exit == 125 {
		crossSeed.DecisionOrdinal = binary.BigEndian.Uint64(run.terminal[72:80])
		crossSeed.Reason = choice.DivergenceReasonName(choice.DivergenceReason(run.terminal[13]))
	}
	return prefixes, crossSeed, nil
}
