package conformance

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"go.temporal.io/server/tools/gomad3/choice"
)

func (campaign *runtimeCampaign) requireRuntimeOwned(fixture string) error {
	var evidence []map[string]any
	for _, mode := range []string{"two", "zero", "one", "busy"} {
		var runs []choiceRun
		users := map[[sha256.Size]byte]bool{}
		root := sha256.Sum256([]byte("gomad3-choice-goroutine-root/v1"))
		for seed := 1; seed <= 32; seed++ {
			run, err := campaign.runChoiceMode("runtime-owned-"+mode+"-"+strconv.Itoa(seed), fixture, strconv.Itoa(seed), nil, choice.ModeRecord, 0, mode)
			if err != nil {
				return err
			}
			want := "[64 64]"
			if mode == "zero" {
				want = "[0 0]"
			}
			if mode == "one" {
				want = "[64 0]"
			}
			if mode == "busy" {
				want = "runtime 64\n[64 64]"
			}
			if run.transcript != want {
				return fmt.Errorf("runtime-owned %s seed %d output %q, want %q", mode, seed, run.transcript, want)
			}
			decisions := 0
			for _, record := range run.trace.Records {
				if record.Kind != choice.KindRunnable {
					continue
				}
				decisions++
				for ordinal := uint64(1); ordinal <= 8; ordinal++ {
					var encoded [8]byte
					binary.BigEndian.PutUint64(encoded[:], ordinal)
					identity := sha256.Sum256(append([]byte("gomad3-choice-goroutine-runtime/v1"), encoded[:]...))
					if identity == record.SelectedIdentity {
						return fmt.Errorf("%s seed %d decision %d selected runtime ordinal %d", mode, seed, record.Ordinal, ordinal)
					}
				}
				if record.SelectedIdentity != root {
					users[record.SelectedIdentity] = true
				}
			}
			if (mode == "zero" || mode == "one") && decisions != 0 {
				return fmt.Errorf("%s seed %d recorded %d decisions", mode, seed, decisions)
			}
			if (mode == "two" || mode == "busy") && decisions == 0 {
				return fmt.Errorf("%s seed %d exposed no user branching", mode, seed)
			}
			evidence = append(evidence, map[string]any{"mode": mode, "seed": seed, "decisions": decisions, "trace_sha256": fmt.Sprintf("%x", run.trace.SHA256)})
			runs = append(runs, run)
		}
		if mode == "zero" || mode == "one" {
			for _, run := range runs[1:] {
				if !slices.Equal(run.trace.Bytes, runs[0].trace.Bytes) {
					return fmt.Errorf("%s deterministic picks changed their trace", mode)
				}
			}
			continue
		}
		if len(users) != 2 {
			return fmt.Errorf("%s selected %d non-root goroutine identities, want the two user goroutines", mode, len(users))
		}
		allowed := [][sha256.Size]byte{root}
		for identity := range users {
			allowed = append(allowed, identity)
		}
		sets := map[[sha256.Size]byte]bool{}
		for mask := 1; mask < 1<<len(allowed); mask++ {
			var subset [][sha256.Size]byte
			for index, identity := range allowed {
				if mask&(1<<index) != 0 {
					subset = append(subset, identity)
				}
			}
			if len(subset) < 2 {
				continue
			}
			digest, err := choice.AlternativeSetDigest(subset)
			if err != nil {
				return err
			}
			sets[digest] = true
		}
		for _, run := range runs {
			for _, record := range run.trace.Records {
				if record.Kind == choice.KindRunnable && !sets[record.AlternativeSetDigest] {
					return fmt.Errorf("%s decision %d offered an identity beyond main and the two user goroutines", mode, record.Ordinal)
				}
			}
		}
		identity, err := campaign.choiceIdentity(fixture)
		if err != nil {
			return err
		}
		plan, err := choice.ProjectReplayPlan(runs[0].trace, identity)
		if err != nil {
			return err
		}
		replaySeed := "2"
		if mode == "busy" {
			replaySeed = "1"
		}
		replayed, err := campaign.runChoiceMode("runtime-owned-"+mode+"-replay", fixture, replaySeed, &plan, choice.ModeReplay, 0, mode)
		if err != nil {
			return err
		}
		if replayed.transcript != runs[0].transcript || !slices.Equal(replayed.trace.Bytes, runs[0].trace.Bytes) {
			return fmt.Errorf("%s replay changed its trace", mode)
		}
		prefix, err := choice.BuildRankPrefix(plan, 0, (plan.Decisions[0].Selected+1)%plan.Decisions[0].Alternatives)
		if err != nil {
			return err
		}
		if _, err := campaign.runChoiceMode("runtime-owned-"+mode+"-prefix", fixture, "1", &prefix, choice.ModePrefix, 0, mode); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(campaign.workspace, "runtime-owned.json"), append(data, '\n'), 0o600)
}
