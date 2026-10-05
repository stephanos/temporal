// Package lint reports findings about the quality of a Model from its IR: what a machine declares
// that nothing reaches, takes, asks, evidences or realizes, and the specification holes of
// .plans/MODALITIES.md. It reads what the reader computes (tables, receipts, decisions) and evaluates
// nothing of its own; each kind's population is printed as a coverage count beside its findings.
//
// What lint reads of lowering, a Query's standing and the descriptors of the evidence a realization
// reads, is given to it by its command (Lowering), since the reader may not import lowering.
package lint

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Kind is what a finding says. Kinds are stable: an acceptance names one.
type Kind string

const (
	// UnreachableValue is an enum case or field value of a state field that no reachable state holds.
	UnreachableValue Kind = "unreachable-value"
	// NeverEnabled is an action class with no enabled row in a reachable state.
	NeverEnabled Kind = "never-enabled"
	// Unproduced is an outcome or a fact no reachable row produces.
	Unproduced Kind = "unproduced"
	// UntakenChoice is a named choice no row of a reachable state takes.
	UntakenChoice Kind = "untaken-choice"
	// UnaskedProperty is a Property no Query names.
	UnaskedProperty Kind = "unasked-property"
	// UnfiredVerify is a verify Query whose Property was read on no step it explored.
	UnfiredVerify Kind = "unfired-verify"
	// UnevidencedFact is a fact of a realized machine that no evidence kind of its realization records.
	UnevidencedFact Kind = "unevidenced-fact"
	// UnperformedAction is an action of a realized machine whose party is not `system` that no
	// performance binds and no activity script starts with.
	UnperformedAction Kind = "unperformed-action"
	// UnrealizedFind is a find Query whose machine no realization runs.
	UnrealizedFind Kind = "unrealized-find"
	// UnreadRefinement is a refinement no Query reads through.
	UnreadRefinement Kind = "unread-refinement"
	// UnreadObservation is an observation of a realization that nothing fills or names.
	UnreadObservation Kind = "unread-observation"
	// ExplicitWait is a poll that writes its own interval in a realization that declares the API
	// behavior its reads' waits are derived from (.plans/API_BEHAVIOR_HINTS.md): a wait no hint covers,
	// kept explicit only where an acceptance records why (fn-118 R4).
	ExplicitWait Kind = "explicit-wait"
	// UnmodeledAPIValue is an enum value or oneof member of a field a realization's poll condition or
	// Run Event guard tests, which no test of an evidence kind that records a fact maps.
	UnmodeledAPIValue Kind = "unmodeled-api-value"
	// DisabledByDefault (H1) is a disabled pair whose empty result a default arm decided: a wildcard
	// `match` case, or an `if` whose condition names no field of the state.
	DisabledByDefault Kind = "disabled-by-default"
	// SilentRejection (H2) is a disabled pair of a party action in a reachable state that is no end.
	SilentRejection Kind = "silent-rejection"
	// UnconstrainedResult (H3) is a class with enabled pairs whose results no claim constrains.
	UnconstrainedResult Kind = "unconstrained-result"
	// WitnessOnly (H4) is a same-step Property only find Queries over pinned Scenarios ask.
	WitnessOnly Kind = "witness-only"
	// MustNotPinned (H5) is a disabled pair of a system action no transition claim pins. It is
	// reported only when Options.MustNotPinned asks for it.
	MustNotPinned Kind = "must-not-pinned"
	// WaivedLaw is a law a declaration's capabilities bring that it waives, with `except` or
	// `overriding`, for the reason the law sidecar records; the model gate's update forwards each such
	// reason into the accepted findings, keyed by `<machine>.<law>` (Forward).
	WaivedLaw Kind = "waived-law"
	// LawWaivedWithoutReason is a waiver the law sidecar records with no reason.
	LawWaivedWithoutReason Kind = "law-waived-without-reason"
	// ReasonNamesNoLaw is a waiver, with its reason, of a law the sidecar's catalog does not bring.
	ReasonNamesNoLaw Kind = "reason-names-no-law"
	// ParameterWithoutCitation is a binding of a parameter its law has each instance back with server
	// code, which cites none.
	ParameterWithoutCitation Kind = "parameter-without-citation"
	// LawWithOneInstance is a law of the catalog that fewer than two machines with their own state
	// types instantiate, across every law sidecar a run reads.
	LawWithOneInstance Kind = "law-with-one-instance"
)

// Finding is one thing lint says of a Model: its kind, the machine or composition it is about, what
// in that owner it is about, a message for its reader and the Scala position of what it names. Kind,
// Owner and Subject identify it, so an acceptance does not go stale when a position or a message
// moves.
type Finding struct {
	Kind     Kind
	Owner    string
	Subject  string
	Message  string
	Position string
}

// Tally is one kind's reading of one owner: how many things it looked at, and the findings among
// them. The rest of its population satisfies the kind, which is what a coverage count prints.
type Tally struct {
	Kind       Kind
	Owner      string
	Population int
	Findings   []Finding
}

// Lowering is what lint reads of lowering, which its command supplies from tools/umpire/lower.
type Lowering struct {
	// Unrealized is whether a Query is a find Query whose Scenario's machine no realization runs: the
	// manifest's `no-realization` standing.
	Unrealized func(q *umpirespb.Query, scenario *umpirespb.Scenario, realizations []*umpirespb.Realization) (bool, error)
	// Element is the message one piece of a kind of evidence of a realization is read as.
	Element func(r *umpirespb.Realization, e *umpirespb.Evidence) (protoreflect.MessageDescriptor, error)
	// Field is the field a realization's path reaches in a message.
	Field func(at *umpirespb.Position, md protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error)
}

// Options are the kinds a run reports beyond the default ones, and what a run reads beyond one file.
type Options struct {
	MustNotPinned bool
	// Instances is each law's instantiating entities across every law sidecar the run reads, which
	// LawWithOneInstance counts; nil counts the file's own sidecar alone.
	Instances Instances
}

// Model is one admitted Model as lint reads it: its IR, the reader's receipts of its verify Queries,
// its machines' tables, and the law sidecar beside it.
type Model struct {
	File string
	IR   *umpirespb.Model
	// Laws is the law sidecar beside the IR file, or nil where its Models declare no capabilities.
	Laws *model.LawSidecar
	// Verified is the receipts of a check of the Model with its verify Queries alone: lint reads
	// whether each Property fired, and a find Query's search answers nothing it asks.
	Verified *model.Report
	Machines map[string]*model.Machine
	In       *model.Interpreter
	realizer *model.Realizer
	actions  map[string]*umpirespb.Action
	lowering Lowering
	options  Options
}

// Read loads, checks and interprets the IR file at path, with the law sidecar beside it. A malformed
// file or sidecar is the reader's error, returned as the reader reports it, and gives no findings.
func Read(path string, lowering Lowering, options Options) (*Model, error) {
	ir, err := model.Load(path)
	if err != nil {
		return nil, err
	}
	laws, err := model.ReadLawSidecar(path)
	if err != nil {
		return nil, err
	}
	m, err := Of(path, ir, lowering, options)
	if err != nil {
		return nil, err
	}
	m.Laws = laws
	return m, nil
}

// Of reads an admitted Model as Read does, under the name file.
func Of(file string, ir *umpirespb.Model, lowering Lowering, options Options) (*Model, error) {
	realizer, err := model.NewRealizer(ir, model.DefaultScope)
	if err != nil {
		return nil, err
	}
	machines := map[string]*model.Machine{}
	for _, decl := range ir.GetMachines() {
		if machines[decl.GetName()] = realizer.Machine(decl.GetName()); machines[decl.GetName()] == nil {
			// Build says why a machine has no table, as the reader reports it.
			_, err := model.Build(ir)
			return nil, cmp.Or(err, fmt.Errorf("%s could not be interpreted", decl.GetName()))
		}
	}
	verify := proto.CloneOf(ir)
	verify.Queries = slices.DeleteFunc(verify.Queries, func(q *umpirespb.Query) bool { return q.GetForm() != umpirespb.Query_FORM_VERIFY })
	verify.Progress = nil
	m := &Model{File: file, IR: ir, Verified: model.Check(verify, model.DefaultScope), Machines: machines, In: model.NewInterpreter(ir),
		realizer: realizer, actions: map[string]*umpirespb.Action{}, lowering: lowering, options: options}
	for _, a := range ir.GetActions() {
		m.actions[a.GetId()] = a
	}
	return m, nil
}

// kind is one kind lint reports: the function that computes its tallies, and the coverage line
// that prints them, or none.
type kind struct {
	kind  Kind
	run   func(m *Model) ([]Tally, error)
	count *count
}

// count is how a coverage line prints a tally: its name, and the words for its population and for
// the part that satisfies the kind.
type count struct {
	name, population, satisfied string
}

// kinds is every kind in the order a coverage block prints their counts.
func kinds() []kind {
	return []kind{
		{kind: UnaskedProperty, run: unaskedProperties, count: &count{"properties", "declared", "named by a Query"}},
		{kind: UnfiredVerify, run: unfiredVerifies, count: &count{"verify Queries", "", "whose Property fired"}},
		{kind: UnperformedAction, run: unperformedActions, count: &count{"non-system actions", "", "performed by a realization"}},
		{kind: UnevidencedFact, run: unevidencedFacts, count: &count{"facts", "", "with evidence"}},
		{kind: UntakenChoice, run: untakenChoices, count: &count{"named choices", "", "taken by a reachable state"}},
		{kind: UnreadRefinement, run: unreadRefinements, count: &count{"refinements", "declared", "read through by a Query"}},
		{kind: UnreadObservation, run: unreadObservations, count: &count{"observations", "", "read"}},
		{kind: ExplicitWait, run: explicitWaits},
		{kind: UnmodeledAPIValue, run: unmodeledAPIValues, count: &count{"API values", "tested", "mapped to a fact"}},
		{kind: UnreachableValue, run: unreachableValues},
		{kind: NeverEnabled, run: neverEnabled},
		{kind: Unproduced, run: unproduced},
		{kind: UnrealizedFind, run: unrealizedFinds},
		{kind: WaivedLaw, run: waivedLaws},
		{kind: LawWaivedWithoutReason, run: lawsWaivedWithoutReason},
		{kind: ReasonNamesNoLaw, run: reasonsNamingNoLaw},
		{kind: ParameterWithoutCitation, run: parametersWithoutCitation},
		{kind: LawWithOneInstance, run: lawsWithOneInstance},
	}
}

// Result is what linting one Model found: every kind's tallies, each machine's per-operation
// modality table with the laws it is held to, and the laws of each composition.
type Result struct {
	File    string
	Tallies []Tally
	Tables  []*Table
	Laws    []*LawTable
}

// Lint runs every kind over the Model.
func (m *Model) Lint() (*Result, error) {
	out := &Result{File: m.File}
	for _, k := range kinds() {
		tallies, err := k.run(m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k.kind, err)
		}
		out.Tallies = append(out.Tallies, tallies...)
	}
	tables, tallies, err := m.holes()
	if err != nil {
		return nil, fmt.Errorf("holes: %w", err)
	}
	out.Tables, out.Laws = tables, m.compositionLaws()
	out.Tallies = append(out.Tallies, tallies...)
	for i := range out.Tallies {
		sortFindings(out.Tallies[i].Findings)
	}
	slices.SortStableFunc(out.Tallies, func(x, y Tally) int {
		return cmp.Or(strings.Compare(x.Owner, y.Owner), cmp.Compare(kindOrder(x.Kind), kindOrder(y.Kind)))
	})
	return out, nil
}

// Findings is every finding of the result, by owner, kind and subject.
func (r *Result) Findings() []Finding {
	var out []Finding
	for _, t := range r.Tallies {
		out = append(out, t.Findings...)
	}
	sortFindings(out)
	return out
}

func sortFindings(fs []Finding) {
	slices.SortStableFunc(fs, func(x, y Finding) int {
		return cmp.Or(strings.Compare(x.Owner, y.Owner), cmp.Compare(kindOrder(x.Kind), kindOrder(y.Kind)),
			strings.Compare(x.Subject, y.Subject))
	})
}

// order is every kind in the order findings are listed.
var order = []Kind{UnreachableValue, NeverEnabled, Unproduced, UntakenChoice, UnaskedProperty, UnfiredVerify, UnevidencedFact,
	UnperformedAction, UnrealizedFind, UnreadRefinement, UnreadObservation, ExplicitWait, UnmodeledAPIValue, DisabledByDefault, SilentRejection,
	UnconstrainedResult, WitnessOnly, MustNotPinned, WaivedLaw, LawWaivedWithoutReason, ReasonNamesNoLaw, ParameterWithoutCitation,
	LawWithOneInstance}

func kindOrder(k Kind) int { return slices.Index(order, k) }

// Kinds is every kind lint knows, in the order findings are listed.
func Kinds() []Kind { return slices.Clone(order) }

// where spells a position as the reader does, `file:line`, or "" where the IR gives none.
func where(p *umpirespb.Position) string {
	if p.GetFile() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", p.GetFile(), p.GetLine())
}
