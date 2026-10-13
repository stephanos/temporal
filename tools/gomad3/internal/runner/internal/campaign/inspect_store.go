package campaign

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/internal/runner/internal/logstore"
)

// StoreInspection is what one read-only replay of a campaign log shows: the
// replay report, the lifecycle, the committed outcomes and admission
// frontier, the rounds of an exploration campaign and, once published, the
// validated campaign. Everything in it comes from the same log snapshot.
type StoreInspection struct {
	Report      logstore.Report
	Lifecycle   LifecycleStatus
	Committed   CommittedLog
	Exploration *ExplorationInspection
	Published   *Campaign
}

// CommittedLog is the committed outcome state of a campaign log: the
// outcomes it holds and their encoded bytes, the execution limits they are
// bounded by and the durable admission frontier.
type CommittedLog struct {
	Outcomes uint64
	Bytes    uint64
	Limits   ExecutionJournalPlan
	Frontier Frontier
}

// InspectCampaignStore replays a campaign log once without its lock and
// reports its state. It creates, repairs and removes nothing: a torn final
// record is reported in the replay report, not truncated. A replay that fails
// returns the report of its validated prefix with the error.
func InspectCampaignStore(path string) (_ StoreInspection, retErr error) {
	defer func() {
		retErr = classifyIntegrityError(retErr)
	}()
	absolute, err := filepath.Abs(path)
	if err != nil {
		return StoreInspection{}, fmt.Errorf("resolve campaign lifecycle path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return StoreInspection{}, fmt.Errorf("inspect campaign lifecycle directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return StoreInspection{}, errors.New("campaign lifecycle directory metadata is invalid")
	}
	state, report, err := replayCampaign(absolute)
	if err != nil {
		return StoreInspection{Report: report}, err
	}
	lifecycle, published, err := publishedLifecycle(absolute, state)
	if err != nil {
		return StoreInspection{Report: report}, err
	}
	inspected := StoreInspection{
		Report: report, Lifecycle: lifecycle, Published: published,
		Committed: CommittedLog{Outcomes: uint64(len(state.executions)), Bytes: state.totalBytes, Limits: state.journalPlan(), Frontier: state.frontierSnapshot()},
	}
	if state.rounds != nil {
		rounds := state.rounds.inspection()
		inspected.Exploration = &rounds
	}
	return inspected, nil
}
