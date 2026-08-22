package virtualtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

type TimerID string

type WorkID string

type TimerStatus string

const (
	TimerPending   TimerStatus = "pending"
	TimerReady     TimerStatus = "ready"
	TimerFired     TimerStatus = "fired"
	TimerCancelled TimerStatus = "cancelled"
)

type Timer struct {
	ID       TimerID     `json:"id"`
	Deadline int64       `json:"deadline"`
	Status   TimerStatus `json:"status"`
}

type State struct {
	now      int64
	runnable []WorkID
	timers   map[TimerID]Timer
}

type StateSnapshot struct {
	Now      int64    `json:"now"`
	Runnable []WorkID `json:"runnable"`
	Timers   []Timer  `json:"timers"`
}

func NewState(now int64) State {
	return State{now: now, timers: make(map[TimerID]Timer)}
}

func Restore(snapshot StateSnapshot) (State, error) {
	state := NewState(snapshot.Now)
	state.runnable = append([]WorkID(nil), snapshot.Runnable...)
	if !slices.IsSorted(state.runnable) {
		return State{}, fmt.Errorf("virtual time runnable identities are not sorted")
	}
	for index, timer := range snapshot.Timers {
		if index > 0 && snapshot.Timers[index-1].ID >= timer.ID {
			return State{}, fmt.Errorf("virtual time timer identities are not strictly sorted")
		}
		if err := validateTimer(state.now, timer); err != nil {
			return State{}, err
		}
		state.timers[timer.ID] = timer
	}
	if err := state.validate(); err != nil {
		return State{}, err
	}
	return state, nil
}

func (state State) Now() int64 {
	return state.now
}

func (state State) Runnable() []WorkID {
	return append([]WorkID(nil), state.runnable...)
}

func (state State) Timer(id TimerID) Timer {
	return state.timers[id]
}

func (state State) TimerOK(id TimerID) (Timer, bool) {
	timer, found := state.timers[id]
	return timer, found
}

func (state State) Snapshot() StateSnapshot {
	timers := make([]Timer, 0, len(state.timers))
	for _, timer := range state.timers {
		timers = append(timers, timer)
	}
	slices.SortFunc(timers, func(left, right Timer) int {
		return cmpString(string(left.ID), string(right.ID))
	})
	return StateSnapshot{
		Now:      state.now,
		Runnable: append([]WorkID(nil), state.runnable...),
		Timers:   timers,
	}
}

func (state State) Identity() string {
	encoded, err := json.Marshal(state.Snapshot())
	if err != nil {
		panic(fmt.Sprintf("marshal virtual time state: %v", err))
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv4.virtual-time-state/v1\x00"))
	_, _ = hasher.Write(encoded)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

type ActionKind string

const (
	ActionScheduleTimer ActionKind = "schedule_timer"
	ActionCancelTimer   ActionKind = "cancel_timer"
	ActionSetRunnable   ActionKind = "set_runnable"
	ActionAdvanceTime   ActionKind = "advance_time"
	ActionFireTimer     ActionKind = "fire_timer"
)

type Action struct {
	Kind     ActionKind `json:"kind"`
	TimerID  TimerID    `json:"timer_id,omitempty"`
	Deadline int64      `json:"deadline,omitempty"`
	WorkID   WorkID     `json:"work_id,omitempty"`
	Runnable bool       `json:"runnable,omitempty"`
}

func ScheduleTimer(id TimerID, deadline int64) Action {
	return Action{Kind: ActionScheduleTimer, TimerID: id, Deadline: deadline}
}

func CancelTimer(id TimerID) Action {
	return Action{Kind: ActionCancelTimer, TimerID: id}
}

func SetRunnable(id WorkID, runnable bool) Action {
	return Action{Kind: ActionSetRunnable, WorkID: id, Runnable: runnable}
}

func AdvanceTime() Action {
	return Action{Kind: ActionAdvanceTime}
}

func FireTimer(id TimerID) Action {
	return Action{Kind: ActionFireTimer, TimerID: id}
}

type ObservableDelta struct {
	Kind         ActionKind  `json:"kind"`
	TimerID      TimerID     `json:"timer_id,omitempty"`
	TimerBefore  TimerStatus `json:"timer_before,omitempty"`
	TimerAfter   TimerStatus `json:"timer_after,omitempty"`
	WorkID       WorkID      `json:"work_id,omitempty"`
	Runnable     bool        `json:"runnable,omitempty"`
	PreviousTime int64       `json:"previous_time,omitempty"`
	CurrentTime  int64       `json:"current_time,omitempty"`
	ReadyTimers  []TimerID   `json:"ready_timers,omitempty"`
}

type Transition struct {
	Action            Action          `json:"action"`
	PreStateIdentity  string          `json:"pre_state_identity"`
	PostStateIdentity string          `json:"post_state_identity"`
	Delta             ObservableDelta `json:"observable_delta"`
	PostState         State           `json:"-"`
}

type RejectionCode string

const (
	RejectionInvalidAction     RejectionCode = "invalid_action"
	RejectionTimerExists       RejectionCode = "timer_exists"
	RejectionUnknownTimer      RejectionCode = "unknown_timer"
	RejectionDeadlineBeforeNow RejectionCode = "deadline_before_now"
	RejectionTimerTerminal     RejectionCode = "timer_terminal"
	RejectionTimerNotReady     RejectionCode = "timer_not_ready"
	RejectionRunnableUnchanged RejectionCode = "runnable_unchanged"
	RejectionRunnableWork      RejectionCode = "runnable_work"
	RejectionReadyTimer        RejectionCode = "ready_timer"
	RejectionNoPendingTimer    RejectionCode = "no_pending_timer"
)

type Rejection struct {
	Code   RejectionCode
	Action ActionKind
	Detail string
}

func (rejection *Rejection) Error() string {
	if rejection.Detail == "" {
		return fmt.Sprintf("virtual time action %s rejected: %s", rejection.Action, rejection.Code)
	}
	return fmt.Sprintf("virtual time action %s rejected: %s: %s", rejection.Action, rejection.Code, rejection.Detail)
}

func Step(state State, action Action) (Transition, error) {
	if err := state.validate(); err != nil {
		return Transition{}, fmt.Errorf("invalid virtual time state: %w", err)
	}
	preIdentity := state.Identity()
	next := state.clone()
	delta := ObservableDelta{Kind: action.Kind}

	var err error
	switch action.Kind {
	case ActionScheduleTimer:
		err = stepSchedule(&next, action, &delta)
	case ActionCancelTimer:
		err = stepCancel(&next, action, &delta)
	case ActionSetRunnable:
		err = stepRunnable(&next, action, &delta)
	case ActionAdvanceTime:
		err = stepAdvance(&next, action, &delta)
	case ActionFireTimer:
		err = stepFire(&next, action, &delta)
	default:
		err = reject(action, RejectionInvalidAction, "unknown action kind")
	}
	if err != nil {
		return Transition{}, err
	}
	if err := next.validate(); err != nil {
		return Transition{}, fmt.Errorf("virtual time transition produced invalid state: %w", err)
	}
	return Transition{
		Action:            action,
		PreStateIdentity:  preIdentity,
		PostStateIdentity: next.Identity(),
		Delta:             delta,
		PostState:         next,
	}, nil
}

func stepSchedule(state *State, action Action, delta *ObservableDelta) error {
	if action.TimerID == "" {
		return reject(action, RejectionInvalidAction, "timer identity is empty")
	}
	if _, found := state.timers[action.TimerID]; found {
		return reject(action, RejectionTimerExists, string(action.TimerID))
	}
	if action.Deadline < state.now {
		return reject(action, RejectionDeadlineBeforeNow, string(action.TimerID))
	}
	status := TimerPending
	if action.Deadline == state.now {
		status = TimerReady
	}
	state.timers[action.TimerID] = Timer{ID: action.TimerID, Deadline: action.Deadline, Status: status}
	delta.TimerID = action.TimerID
	delta.TimerAfter = status
	return nil
}

func stepCancel(state *State, action Action, delta *ObservableDelta) error {
	timer, found := state.timers[action.TimerID]
	if !found {
		return reject(action, RejectionUnknownTimer, string(action.TimerID))
	}
	if timer.Status == TimerFired || timer.Status == TimerCancelled {
		return reject(action, RejectionTimerTerminal, string(action.TimerID))
	}
	delta.TimerID = action.TimerID
	delta.TimerBefore = timer.Status
	delta.TimerAfter = TimerCancelled
	timer.Status = TimerCancelled
	state.timers[action.TimerID] = timer
	return nil
}

func stepRunnable(state *State, action Action, delta *ObservableDelta) error {
	if action.WorkID == "" {
		return reject(action, RejectionInvalidAction, "work identity is empty")
	}
	index, found := slices.BinarySearch(state.runnable, action.WorkID)
	if found == action.Runnable {
		return reject(action, RejectionRunnableUnchanged, string(action.WorkID))
	}
	if action.Runnable {
		state.runnable = append(state.runnable, "")
		copy(state.runnable[index+1:], state.runnable[index:])
		state.runnable[index] = action.WorkID
	} else {
		state.runnable = slices.Delete(state.runnable, index, index+1)
	}
	delta.WorkID = action.WorkID
	delta.Runnable = action.Runnable
	return nil
}

func stepAdvance(state *State, action Action, delta *ObservableDelta) error {
	if len(state.runnable) > 0 {
		return reject(action, RejectionRunnableWork, "runnable work remains")
	}
	var earliest int64
	found := false
	for _, timer := range state.timers {
		if timer.Status == TimerReady {
			return reject(action, RejectionReadyTimer, string(timer.ID))
		}
		if timer.Status == TimerPending && (!found || timer.Deadline < earliest) {
			earliest = timer.Deadline
			found = true
		}
	}
	if !found {
		return reject(action, RejectionNoPendingTimer, "no pending timer remains")
	}
	delta.PreviousTime = state.now
	delta.CurrentTime = earliest
	state.now = earliest
	for id, timer := range state.timers {
		if timer.Status == TimerPending && timer.Deadline == earliest {
			timer.Status = TimerReady
			state.timers[id] = timer
			delta.ReadyTimers = append(delta.ReadyTimers, id)
		}
	}
	slices.Sort(delta.ReadyTimers)
	return nil
}

func stepFire(state *State, action Action, delta *ObservableDelta) error {
	timer, found := state.timers[action.TimerID]
	if !found {
		return reject(action, RejectionUnknownTimer, string(action.TimerID))
	}
	if timer.Status != TimerReady {
		return reject(action, RejectionTimerNotReady, string(action.TimerID))
	}
	delta.TimerID = action.TimerID
	delta.TimerBefore = TimerReady
	delta.TimerAfter = TimerFired
	timer.Status = TimerFired
	state.timers[action.TimerID] = timer
	return nil
}

func (state State) clone() State {
	cloned := State{
		now:      state.now,
		runnable: append([]WorkID(nil), state.runnable...),
		timers:   make(map[TimerID]Timer, len(state.timers)),
	}
	for id, timer := range state.timers {
		cloned.timers[id] = timer
	}
	return cloned
}

func (state State) validate() error {
	if !slices.IsSorted(state.runnable) {
		return fmt.Errorf("runnable identities are not sorted")
	}
	for index, id := range state.runnable {
		if id == "" {
			return fmt.Errorf("runnable identity is empty")
		}
		if index > 0 && state.runnable[index-1] == id {
			return fmt.Errorf("runnable identity %q is duplicated", id)
		}
	}
	for id, timer := range state.timers {
		if timer.ID != id {
			return fmt.Errorf("timer key %q does not match identity %q", id, timer.ID)
		}
		if err := validateTimer(state.now, timer); err != nil {
			return err
		}
	}
	return nil
}

func validateTimer(now int64, timer Timer) error {
	if timer.ID == "" {
		return fmt.Errorf("timer identity is empty")
	}
	switch timer.Status {
	case TimerPending:
		if timer.Deadline <= now {
			return fmt.Errorf("pending timer %q deadline %d is not after now %d", timer.ID, timer.Deadline, now)
		}
	case TimerReady:
		if timer.Deadline != now {
			return fmt.Errorf("ready timer %q deadline %d does not equal now %d", timer.ID, timer.Deadline, now)
		}
	case TimerFired, TimerCancelled:
	default:
		return fmt.Errorf("timer %q has invalid status %q", timer.ID, timer.Status)
	}
	return nil
}

func reject(action Action, code RejectionCode, detail string) error {
	return &Rejection{Code: code, Action: action.Kind, Detail: detail}
}

func cmpString(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
