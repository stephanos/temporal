package model

import (
	"cmp"
	"flag"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var captureMigrationGoldens = flag.String("capture-goldens", "", "exclusively create a new migration golden capture directory")

type migrationTable struct {
	Table    tableSide
	Evidence [][2]string
	Because  [][]string
	Fields   map[string][]Atom
	Claims   []Claim
	Unknown  []string
}

type migrationSubject struct {
	Name       string
	Error      string
	Table      *migrationTable
	Refinement []RefinementRow
}

type migrationPropertyRow struct {
	Row          string
	Result       int
	About, Holds bool
	Error        string
}

type migrationProperty struct {
	Owner, Name, ID, Error string
	Rows                   []migrationPropertyRow
}

// migrationPropertyMeaning is a Property's answers in the compact form a comparison of two live
// interpretations reads: the actions of its subject's rows it is about, each once, and the answer for
// each row result it is about, in table order. A row is named by its index into the subject's table,
// which the comparison holds on its own, so a changed About set, answer, row order or row count each
// shows.
type migrationPropertyMeaning struct {
	Owner, Name, ID, Error string
	About                  []string
	Answers                []migrationPropertyAnswer
}

type migrationPropertyAnswer struct {
	Row    int
	Result int    `json:",omitempty"`
	Holds  bool   `json:",omitempty"`
	Error  string `json:",omitempty"`
}

// rows is the Property's answer for every row result of its table, as the checked-in goldens and the
// original baseline's digests record it. Table is the one the answers index, nil when they were not
// read.
func (p migrationPropertyMeaning) rows(table *Table) migrationProperty {
	out := migrationProperty{Owner: p.Owner, Name: p.Name, ID: p.ID, Error: p.Error}
	if table == nil {
		return out
	}
	about := map[string]bool{}
	for _, action := range p.About {
		about[action] = true
	}
	answers := p.Answers
	for i, row := range table.Rows {
		for j := range row.Results {
			answer := migrationPropertyRow{Row: row.Key, Result: j, About: about[row.Action]}
			if answer.About {
				if answers[0].Row != i || answers[0].Result != j {
					panic(fmt.Sprintf("property %s answers row %d result %d before row %d result %d", p.Name, answers[0].Row, answers[0].Result, i, j))
				}
				answer.Holds, answer.Error = answers[0].Holds, answers[0].Error
				answers = answers[1:]
			}
			out.Rows = append(out.Rows, answer)
		}
	}
	if len(answers) > 0 {
		panic(fmt.Sprintf("property %s answers %d rows its table does not have", p.Name, len(answers)))
	}
	return out
}

type migrationReceipt struct {
	Receipt
	Cause string
	Also  []migrationReceipt
}

type migrationSemantics struct {
	Subjects   []migrationSubject
	Properties []migrationProperty
	Receipts   []migrationReceipt
}

func migrationError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// migrationRefinement is the rows of a refinement over the machine's own rows: the ones RefineTables
// pairs with the product's steps where the rows refine, even when a reachable hole leaves the
// refinement itself unknown, and none where they do not. Why a refinement does not hold is its
// receipt's.
func migrationRefinement(r *refined) []RefinementRow {
	if r.ref == nil || r.err != nil && r.incomplete == nil {
		return nil
	}
	return r.ref.Rows
}

func migrationTableOf(table *Table) *migrationTable {
	out := &migrationTable{Table: sideOf(table), Evidence: table.Evidence, Claims: table.Claims(), Fields: map[string][]Atom{}}
	for _, state := range table.States {
		out.Fields[state] = table.FieldValues(state)
	}
	for _, row := range table.Rows {
		var reasons []string
		for _, result := range row.Results {
			reasons = append(reasons, result.Because)
		}
		out.Because = append(out.Because, reasons)
	}
	for _, u := range table.Unknown {
		out.Unknown = append(out.Unknown, u.Row+": "+migrationError(u.Cause))
	}
	return out
}

func migrationReceipts(receipts []Receipt) []migrationReceipt {
	var out []migrationReceipt
	for _, r := range receipts {
		out = append(out, migrationReceipt{Receipt: r, Cause: migrationError(r.Cause), Also: migrationReceipts(r.Also)})
	}
	return out
}

// migrationBinding is one interpretation of an admitted Model, as a producer of Cases reads it. The
// helpers that read one Model's meaning, declarations and refined Properties share it within one
// comparison, so the Model is interpreted once for them; each comparison binds its own.
func migrationBinding(t *testing.T, m *umpirespb.Model) *binding {
	t.Helper()
	require.NoError(t, Validate(m))
	b := bind(m, DefaultScope)
	b.realizing = true
	return b
}

func migrationMeaning(t *testing.T, m *umpirespb.Model) migrationSemantics {
	t.Helper()
	return migrationMeaningOf(migrationBinding(t, m))
}

func migrationMeaningOf(b *binding) migrationSemantics {
	var out migrationSemantics
	out.Receipts = migrationMeaningEach(b, func(s migrationSubject) { out.Subjects = append(out.Subjects, s) },
		func(p migrationProperty) { out.Properties = append(out.Properties, p) })
	return out
}

// migrationMeaningEach reads a Model's meaning one subject and one Property at a time, so a caller can
// digest a Model whose meaning is too large to hold, and answers its receipts. Each Property has a row
// for every row result of its table.
func migrationMeaningEach(b *binding, subjectOf func(migrationSubject), propertyOf func(migrationProperty)) []migrationReceipt {
	return migrationMeaningParts(b, subjectOf, func(p migrationPropertyMeaning, table *Table) { propertyOf(p.rows(table)) })
}

// migrationMeaningParts is migrationMeaningEach with each Property in its compact form, with the table
// its answers index.
func migrationMeaningParts(b *binding, subjectOf func(migrationSubject), propertyOf func(migrationPropertyMeaning, *Table)) []migrationReceipt {
	m := b.model
	receipts := migrationReceipts(checkWithBinding(m, b.scope, m, b.checking()).Receipts)
	var subjects []string
	for _, machine := range m.GetMachines() {
		subjects = append(subjects, machine.GetName())
	}
	for _, composition := range m.GetCompositions() {
		subjects = append(subjects, composition.GetName())
	}
	for _, name := range subjects {
		subject := b.subject(name)
		entry := migrationSubject{Name: name, Error: migrationError(subject.err)}
		table := subject.table
		if mm := b.machines[name]; mm != nil {
			if mm.Decl.GetRefines() != nil && subject.err == nil {
				entry.Refinement = migrationRefinement(b.refinement(subject))
			}
			if table == nil {
				table = mm.Table
			}
		}
		if table != nil {
			entry.Table = migrationTableOf(table)
		}
		subjectOf(entry)
	}
	for _, p := range m.GetProperties() {
		subject := b.subject(p.GetMachine())
		entry := migrationPropertyMeaning{Owner: p.GetMachine(), Name: p.GetName(), ID: subject.family + ".property." + p.GetName()}
		var table *Table
		if subject.err != nil {
			entry.Error = migrationError(subject.err)
		} else {
			reading, err := b.propertyReads(subject, p)
			if err != nil {
				entry.Error = migrationError(err)
			} else {
				table = subject.table
				bound := boundProperty(p, reading)
				// About reads only the action, so each action is asked once.
				about := map[string]bool{}
				for i, row := range table.Rows {
					is, asked := about[row.Action]
					if !asked {
						is = bound.About(row.Action)
						about[row.Action] = is
						if is {
							entry.About = append(entry.About, row.Action)
						}
					}
					if !is {
						continue
					}
					for j, result := range row.Results {
						answer := migrationPropertyAnswer{Row: i, Result: j}
						answer.Holds, err = bound.Holds(row.Source, result)
						answer.Error = migrationError(err)
						entry.Answers = append(entry.Answers, answer)
					}
				}
			}
		}
		propertyOf(entry, table)
	}
	return receipts
}

func migrationKey(path string) string { return strings.TrimPrefix(path, "model/scalav2/") }

func migrationFiles(t *testing.T, models map[string]*umpirespb.Model) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for _, path := range slices.Sorted(maps.Keys(models)) {
		key := migrationKey(path)
		encoded, err := golden.Proto(models[path])
		require.NoError(t, err)
		files["inputs/"+key] = encoded
		b := migrationBinding(t, models[path])
		encoded, err = golden.JSON(migrationMeaningOf(b))
		require.NoError(t, err)
		files["semantics/"+key] = encoded
		encoded, err = golden.JSON(migrationDefinitionsOf(t, b))
		require.NoError(t, err)
		files["declarations/"+key] = encoded
		encoded, err = golden.JSON(migrationRefinedPropertiesOf(b))
		require.NoError(t, err)
		files["refined-properties/"+key] = encoded
	}
	return files
}

func migrationInputs(t *testing.T) (golden.Config, map[string]*umpirespb.Model) {
	t.Helper()
	cfg, err := golden.Configuration()
	require.NoError(t, err)
	root, err := golden.Root()
	require.NoError(t, err)
	models, err := cfg.Inputs(root)
	require.NoError(t, err)
	return cfg, models
}

func TestCaptureMigrationGoldens(t *testing.T) {
	if *captureMigrationGoldens == "" {
		t.Skip("explicit -capture-goldens=<new directory> required")
	}
	_, models := migrationInputs(t)
	require.NoError(t, golden.Capture(*captureMigrationGoldens, migrationFiles(t, models)))
}

func TestMigrationGoldens(t *testing.T) {
	if *captureMigrationGoldens != "" {
		t.Skip("capture is separate from verification")
	}
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	cfg, models := migrationInputs(t)
	// A reduced or retired fixture is no input, so none of its goldens is compared.
	expected = cfg.Unreduced(expected)
	require.NoError(t, cfg.RootsApply(migrationOriginals(t, expected, models), models))
	for path, current := range models {
		original := new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(expected["inputs/"+migrationKey(path)], original), path)
		_, err := cfg.MatchAt(golden.OriginalKey(path), original, current)
		require.NoError(t, err, path)
		// Evaluate the immutable source spelling so located errors remain byte-comparable after a move.
		models[path] = original
	}
	require.NoError(t, cfg.FunctionsRenamed(models))
	require.NoError(t, cfg.TypesRenamed(models))
	require.NoError(t, golden.Compare(expected, migrationFiles(t, models)))
}

// TestMigrationGoldensAdmitOnlyTheLawReplacements compares the standalone activity, whose laws
// generate claims, with its frozen input: at its original-baseline key the listed replacements are
// admitted, and nothing beside them.
func TestMigrationGoldensAdmitOnlyTheLawReplacements(t *testing.T) {
	const path = "model/scalav2/ir/activity.json"
	frozen, err := golden.Read(filepath.Join("testdata", "migration", "inputs"))
	require.NoError(t, err)
	original := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(frozen[migrationKey(path)], original))
	cfg, models := migrationInputs(t)
	// Every listed rename is to a Function the current IR declares.
	require.NoError(t, cfg.FunctionsCurrent(models))
	current, key := models[path], golden.OriginalKey(path)
	_, err = cfg.MatchAt(key, original, current)
	require.NoError(t, err)
	_, err = cfg.Match(original, current)
	require.Error(t, err, "without the key no replacement applies")
	withoutRoots := cfg
	withoutRoots.RootRetirements = nil
	_, err = withoutRoots.MatchAt(key, original, current)
	require.Error(t, err, "a retired root the configuration does not list")
	for name, change := range map[string]func(*umpirespb.Model){
		"unlisted generated claim": func(m *umpirespb.Model) {
			extra := proto.CloneOf(m.GetQueries()[slices.IndexFunc(m.GetQueries(), func(q *umpirespb.Query) bool {
				return q.GetName() == "activityProduct.terminalStatesAreFinal"
			})])
			extra.Name = "activityProduct.another"
			m.Queries = append(m.Queries, extra)
		},
		"retired Query kept": func(m *umpirespb.Model) {
			m.Queries = append(m.Queries, &umpirespb.Query{Name: "terminalHolds", Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: "activityProduct", Name: "activityProduct.terminalStatesAreFinal"},
				Scenario: &umpirespb.ClaimRef{Machine: "activityProtocol", Name: "completed"}, Through: true,
				Limits: &umpirespb.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}})
		},
		"generated twin without its Scenario": func(m *umpirespb.Model) {
			m.Scenarios = slices.DeleteFunc(m.Scenarios, func(s *umpirespb.Scenario) bool { return s.GetName() == "activityProtocol.terminateSettles" })
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			_, err := cfg.MatchAt(key, original, changed)
			require.Error(t, err)
		})
	}
}

// TestMigrationGoldensAdmitOnlyTheDerivedWaits compares the Nexus caller, whose realization leaves the
// listed waits to the API behavior, with its frozen input: at its original-baseline key exactly those
// waits are admitted. A listed command that writes its wait again, or an unlisted one whose wait
// changes, fails.
func TestMigrationGoldensAdmitOnlyTheDerivedWaits(t *testing.T) {
	const path = "model/scalav2/ir/nexus-caller.json"
	frozen, err := golden.Read(filepath.Join("testdata", "migration", "inputs"))
	require.NoError(t, err)
	original := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(frozen[migrationKey(path)], original))
	cfg, models := migrationInputs(t)
	// Every listed rename is to a Function the current IR declares.
	require.NoError(t, cfg.FunctionsCurrent(models))
	current, key := models[path], golden.OriginalKey(path)
	_, err = cfg.MatchAt(key, original, current)
	require.NoError(t, err)
	_, err = cfg.Match(original, current)
	require.Error(t, err, "without the key no derived wait applies")
	command := func(m *umpirespb.Model, script, id string) *umpirespb.Command {
		for _, r := range m.GetRealizations() {
			for _, s := range r.GetScripts() {
				for _, item := range s.GetItems() {
					commands := []*umpirespb.Command{item.GetCommand()}
					for _, p := range item.GetPerforms() {
						commands = append(commands, p.GetCommand())
					}
					for _, c := range commands {
						if s.GetId() == script && c.GetId() == id {
							return c
						}
					}
				}
			}
		}
		require.FailNow(t, "no command "+script+"/"+id)
		return nil
	}
	for name, change := range map[string]func(*umpirespb.Model){
		"listed timeout written again":  func(m *umpirespb.Model) { command(m, "handler", "respond-async").TimeoutMs = 5000 },
		"listed interval written again": func(m *umpirespb.Model) { command(m, "controller", "await-scheduled").GetPoll().IntervalMs = 250 },
		"listed command changed":        func(m *umpirespb.Model) { command(m, "controller", "await-scheduled").GetPoll().Evidence = "other" },
		"unlisted timeout":              func(m *umpirespb.Model) { command(m, "controller", "await-close").TimeoutMs = 5000 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			_, err := cfg.MatchAt(key, original, changed)
			require.Error(t, err)
		})
	}
}

// migrationOriginals are the frozen inputs of the models, by path.
func migrationOriginals(t *testing.T, expected map[string][]byte, models map[string]*umpirespb.Model) map[string]*umpirespb.Model {
	t.Helper()
	out := map[string]*umpirespb.Model{}
	for path := range models {
		out[path] = new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(expected["inputs/"+migrationKey(path)], out[path]), path)
	}
	return out
}

func TestMigrationGoldenDetectsSemanticMutations(t *testing.T) {
	m, err := Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	original := migrationMeaning(t, m)
	require.NotEmpty(t, original.Subjects)
	require.NotEmpty(t, original.Properties)
	require.NotEmpty(t, original.Receipts)
	mutations := map[string]func(*migrationSemantics){
		"table result":  func(s *migrationSemantics) { s.Subjects[0].Table.Table.Rows[0].Results[0].State += "-changed" },
		"definition ID": func(s *migrationSemantics) { s.Subjects[0].Table.Table.IDs.Target += "-changed" },
		"fingerprint":   func(s *migrationSemantics) { s.Subjects[0].Table.Table.Fingerprint += "-changed" },
		"refinement": func(s *migrationSemantics) {
			s.Subjects[0].Refinement = append(s.Subjects[0].Refinement, RefinementRow{Key: "changed"})
		},
		"property on every row": func(s *migrationSemantics) { s.Properties[0].Rows[0].Holds = !s.Properties[0].Rows[0].Holds },
		"unsupported standing":  func(s *migrationSemantics) { s.Receipts[0].Kind = Unsupported },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			fresh := migrationMeaning(t, m)
			before, err := golden.JSON(fresh)
			require.NoError(t, err)
			mutate(&fresh)
			after, err := golden.JSON(fresh)
			require.NoError(t, err)
			require.Error(t, golden.Compare(map[string][]byte{"model": before}, map[string][]byte{"model": after}))
		})
	}
}

func TestMigrationProjectionPreservesSemantics(t *testing.T) {
	cfg, models := migrationInputs(t)
	originals, err := golden.Read(filepath.Join("testdata", "migration", "inputs"))
	require.NoError(t, err)
	for _, path := range slices.Sorted(maps.Keys(models)) {
		t.Run(migrationKey(path), func(t *testing.T) {
			frozen := new(umpirespb.Model)
			require.NoError(t, protojson.Unmarshal(originals[migrationKey(path)], frozen))
			// The original baseline's delta, the task-queue entity fn-112 attaches and the claims fn-122's
			// laws replace, is read on the frozen input too, and the current Model without the generated
			// claims it lists, as MatchAt reads them.
			key := golden.OriginalKey(path)
			original, err := golden.Attached(key, frozen)
			require.NoError(t, err)
			ungenerated, err := golden.Ungenerated(key, models[path])
			require.NoError(t, err)
			mapped, err := cfg.Migrate(original)
			require.NoError(t, err)
			migrate := locationMigrator(cfg)
			// nexus-close's meaning is hundreds of megabytes of JSON, so each side is digested part by
			// part rather than held.
			project := locationProjection(cfg)
			// The original, the mapped and the current Model are each interpreted once, and apart. Each
			// interpretation is dropped once read: nexus-close's takes most of the memory limit.
			read := readMigration(t, original, migrate, project)
			readMapped := readMigration(t, mapped, func(s string) string { return s })
			requireSameMeaning(t, read.meaning[0], readMapped.meaning[0])
			definitions := read.definitions
			for i := range definitions {
				definitions[i].Error = migrate(definitions[i].Error)
			}
			require.Equal(t, definitions, readMapped.definitions)
			refined := read.refined
			for i := range refined {
				refined[i].Error = migrate(refined[i].Error)
			}
			require.Equal(t, refined, readMapped.refined)
			// The current IR, which Match admits under the projection, reads as the original does. The
			// projection names a path the same in each spelling, so it applies over the migration.
			current := readMigration(t, ungenerated, project)
			requireSameMeaning(t, read.meaning[1], current.meaning[0])
			for i := range definitions {
				definitions[i].Error = project(definitions[i].Error)
			}
			for i := range refined {
				refined[i].Error = project(refined[i].Error)
			}
			want, err := golden.JSON(struct {
				Definitions []migrationDefinition
				Refined     []migrationRefinedProperty
			}{definitions, refined})
			require.NoError(t, err)
			currentDefinitions, currentRefined := current.definitions, current.refined
			for i := range currentDefinitions {
				currentDefinitions[i].Error = project(currentDefinitions[i].Error)
			}
			for i := range currentRefined {
				currentRefined[i].Error = project(currentRefined[i].Error)
			}
			got, err := golden.JSON(struct {
				Definitions []migrationDefinition
				Refined     []migrationRefinedProperty
			}{currentDefinitions, currentRefined})
			require.NoError(t, err)
			require.Equal(t, string(want), string(got))
		})
	}
}

// migrationReading is what the projection test reads of one interpretation: its meaning digested
// under each location mapping, its definitions and its refined Properties.
type migrationReading struct {
	meaning     []*golden.Stream
	definitions []migrationDefinition
	refined     []migrationRefinedProperty
}

// readMigration interprets a Model and reads it in the order the readers always have: its meaning,
// then its definitions, then its refined Properties. The interpretation is not kept.
func readMigration(t *testing.T, m *umpirespb.Model, locates ...func(string) string) migrationReading {
	t.Helper()
	b := migrationBinding(t, m)
	return migrationReading{meaningDigests(t, b, locates...), migrationDefinitionsOf(t, b), migrationRefinedPropertiesOf(b)}
}

// locationProjection maps every source path a located string names, frozen or current, to its
// current spelling before PathMoves, and drops the line and column after it, as Match compares
// positions by file.
func locationProjection(cfg golden.Config) func(string) string {
	current := map[string]string{}
	for _, path := range cfg.Paths {
		current[path.Old], current[path.New] = path.New, path.New
	}
	for _, rename := range cfg.Renames {
		for from, to := range current {
			if to == rename.Old {
				current[from] = rename.New
			}
		}
		current[rename.New] = rename.New
	}
	// A file split out of another names, as Match compares it, the file its declarations came from.
	for _, split := range cfg.Splits {
		current[split.Old] = current[split.New]
	}
	// A path under a merged directory, listed or not, names the directory, as Match compares it.
	merged := func(path string) (string, bool) {
		for _, merge := range cfg.Merges {
			if strings.HasPrefix(path, merge.Old) {
				return merge.New, true
			}
		}
		return path, false
	}
	for from, to := range current {
		current[from], _ = merged(to)
	}
	files := slices.SortedFunc(maps.Keys(current), func(x, y string) int { return cmp.Or(len(y)-len(x), strings.Compare(x, y)) })
	for i := range files {
		files[i] = regexp.QuoteMeta(files[i])
	}
	// A merged directory names every file under it; a merged file names itself, listed or not.
	for _, merge := range cfg.Merges {
		if strings.HasSuffix(merge.Old, "/") {
			files = append(files, regexp.QuoteMeta(merge.Old)+`[\w./-]*\w`)
		} else {
			files = append(files, regexp.QuoteMeta(merge.Old))
		}
	}
	located := regexp.MustCompile(`(` + strings.Join(files, "|") + `)(?::[0-9]+)*`)
	return func(s string) string {
		// A path under a moved directory is first spelled under the old one, as Match reads it.
		return located.ReplaceAllStringFunc(cfg.UnmovedText(s), func(at string) string {
			path := located.FindStringSubmatch(at)[1]
			if to, ok := current[path]; ok {
				return to
			}
			to, _ := merged(path)
			return to
		})
	}
}

// projectedMeaning is every reader snapshot of one interpretation, with its located strings projected.
// It reads the binding's refined Properties, which a binding gives once.
func projectedMeaning(t *testing.T, b *binding, project func(string) string) []byte {
	t.Helper()
	return projectedSnapshots(t, migrationMeaningOf(b), migrationDefinitionsOf(t, b), migrationRefinedPropertiesOf(b), project)
}

func projectedSnapshots(t *testing.T, semantics migrationSemantics, definitions []migrationDefinition, refined []migrationRefinedProperty, project func(string) string) []byte {
	t.Helper()
	migrateSemanticLocations(&semantics, project)
	for i := range definitions {
		definitions[i].Error = project(definitions[i].Error)
	}
	for i := range refined {
		refined[i].Error = project(refined[i].Error)
	}
	encoded, err := golden.JSON(struct {
		Semantics   migrationSemantics
		Definitions []migrationDefinition
		Refined     []migrationRefinedProperty
	}{semantics, definitions, refined})
	require.NoError(t, err)
	return encoded
}

// migrationRewrite edits a Model's ProtoJSON text, for changes that touch every reference at once.
func migrationRewrite(t *testing.T, m *umpirespb.Model, rewrite func(string) string) *umpirespb.Model {
	t.Helper()
	encoded, err := protojson.Marshal(m)
	require.NoError(t, err)
	out := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal([]byte(rewrite(string(encoded))), out))
	return out
}

func firstIf(m protoreflect.Message) *umpirespb.If {
	var found *umpirespb.If
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if f.Message() == nil || f.IsMap() {
			return true
		}
		visit := func(child protoreflect.Message) {
			if found != nil {
				return
			}
			if x, ok := child.Interface().(*umpirespb.If); ok {
				found = x
				return
			}
			found = firstIf(child)
		}
		if f.IsList() {
			for i := range v.List().Len() {
				visit(v.List().Get(i).Message())
			}
		} else {
			visit(v.Message())
		}
		return found == nil
	})
	return found
}

// TestMigrationGoldensAdmitOnlyTheProjection changes the current Nexus caller IR the way the
// IR-changing tasks do, and in the ways the projection must not admit.
func TestMigrationGoldensAdmitOnlyTheProjection(t *testing.T) {
	const path = "model/scalav2/ir/nexus-caller.json"
	const kernel = "temporal.nexuscaller.kernel.Protocol$.completeStep"
	frozen, err := golden.Read(filepath.Join("testdata", "migration", "inputs"))
	require.NoError(t, err)
	original := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(frozen[migrationKey(path)], original))
	cfg, models := migrationInputs(t)
	// Every listed rename is to a Function the current IR declares.
	require.NoError(t, cfg.FunctionsCurrent(models))
	current, key := models[path], golden.OriginalKey(path)
	require.NotNil(t, functionNamed(original, kernel))
	moved := "temporal.features.nexuscaller.Protocol$.effects$.completeStep"
	if i := slices.IndexFunc(cfg.Projection.Functions, func(s golden.Substitution) bool { return s.Old == kernel }); i >= 0 {
		moved = cfg.Projection.Functions[i].New
	} else {
		cfg.Projection.Functions = append(cfg.Projection.Functions, golden.Substitution{Old: kernel, New: moved})
	}
	// The current IR names the Function in the package it moved to since (fn-114.9).
	moved = cfg.MovedPackage(moved)
	// Each substitution renames a Function of the frozen IR that declares it, the Nexus caller's or
	// another's.
	originals, err := golden.FrozenModels(frozen, "ir/")
	require.NoError(t, err)
	require.NoError(t, cfg.FunctionsRenamed(originals))
	require.NoError(t, cfg.TypesRenamed(map[string]*umpirespb.Model{path: original}))
	shifted := regexp.MustCompile(`"line":\s*([0-9]+)`)
	shift := func(s string) string {
		return shifted.ReplaceAllStringFunc(s, func(at string) string {
			line, err := strconv.Atoi(shifted.FindStringSubmatch(at)[1])
			require.NoError(t, err)
			return `"line":` + strconv.Itoa(line+3)
		})
	}
	rename := func(old, name string) func(string) string {
		return func(s string) string { return strings.ReplaceAll(s, strconv.Quote(old), strconv.Quote(name)) }
	}
	admitted := migrationRewrite(t, current, func(s string) string {
		return rename("_$1", "placeholder")(rename(kernel, moved)(shift(s)))
	})
	encoded, err := golden.Proto(admitted)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "_$1")
	_, err = cfg.MatchAt(key, original, admitted)
	require.NoError(t, err)
	project := locationProjection(cfg)
	want := projectedMeaning(t, migrationBinding(t, original), project)
	require.Equal(t, string(want), string(projectedMeaning(t, migrationBinding(t, admitted), project)),
		"what the projection admits reads as the frozen original does")

	// The IR comparison reads no Function, so a flipped guard passes it; the reading refuses it. A
	// renamed step whose guard is negated twice means the same and reads the same.
	t.Run("flipped guard", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		step := functionNamed(changed, changed.GetMachines()[0].GetSteps()[0].GetFunction())
		branch := firstIf(step.ProtoReflect())
		require.NotNil(t, branch)
		branch.Then, branch.Else = branch.Else, branch.Then
		_, err := cfg.MatchAt(key, original, changed)
		require.NoError(t, err)
		require.NotEqual(t, string(want), string(projectedMeaning(t, migrationBinding(t, changed), project)))
	})
	t.Run("renamed step with a guard negated twice", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		step := functionNamed(changed, changed.GetMachines()[0].GetSteps()[0].GetFunction())
		branch := firstIf(step.ProtoReflect())
		require.NotNil(t, branch)
		not := func(e *umpirespb.Expr) *umpirespb.Expr {
			return &umpirespb.Expr{Position: e.GetPosition(), Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: e}}}
		}
		branch.Condition = not(not(branch.GetCondition()))
		changed = migrationRewrite(t, changed, rename(step.GetName(), step.GetName()+"Restructured"))
		_, err := cfg.MatchAt(key, original, changed)
		require.NoError(t, err)
		require.Equal(t, string(want), string(projectedMeaning(t, migrationBinding(t, changed), project)))
	})
	for name, rewrite := range map[string]func(string) string{
		"unlisted function rename":      rename(moved, moved+"Unlisted"),
		"listed function rename unmade": rename(moved, kernel),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cfg.MatchAt(key, original, migrationRewrite(t, admitted, rewrite))
			require.NoError(t, err, "the IR comparison reads every Function reference as one token")
		})
	}
	t.Run("position in an unlisted file", func(t *testing.T) {
		changed := migrationRewrite(t, admitted, rename("model/temporal/features/nexuscaller/NexusCaller.scala", "model/temporal/features/nexuscaller/Caller.scala"))
		_, err := cfg.MatchAt(key, original, changed)
		require.Error(t, err)
	})
	t.Run("Function reference removed", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		i := slices.IndexFunc(changed.GetMachines(), func(m *umpirespb.Machine) bool { return m.GetEvidence() != "" })
		require.GreaterOrEqual(t, i, 0)
		changed.GetMachines()[i].Evidence = ""
		_, err := cfg.MatchAt(key, original, changed)
		require.Error(t, err, "an empty reference stays empty")
	})
	t.Run("parameters swapped over an unchanged body", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		var swapped *umpirespb.Function
		for _, f := range changed.GetFunctions() {
			if len(f.GetParams()) < 2 || swapped != nil {
				continue
			}
			body, err := protojson.Marshal(f.GetBody())
			require.NoError(t, err)
			reads := regexp.MustCompile(`"var":\s*"([^"]*)"`).FindAllStringSubmatch(string(body), -1)
			read := map[string]bool{}
			for _, r := range reads {
				read[r[1]] = true
			}
			if read[f.GetParams()[0].GetName()] && read[f.GetParams()[1].GetName()] && f.GetParams()[0].GetName() != f.GetParams()[1].GetName() {
				swapped = f
			}
		}
		require.NotNil(t, swapped)
		params := swapped.GetParams()
		params[0].Name, params[1].Name = params[1].GetName(), params[0].GetName()
		_, err := cfg.MatchAt(key, original, changed)
		require.NoError(t, err, swapped.GetName())
		require.NotEqual(t, string(want), string(projectedMeaning(t, migrationBinding(t, changed), project)), swapped.GetName())
	})
}

type migrationRefinedProperty struct {
	Machine, Owner, Name, Row, Error string
	Outcome                          Outcome
	About                            bool
	Witness                          *Trace
	Rows                             []string
}

func migrationRefinedProperties(t *testing.T, model *umpirespb.Model) []migrationRefinedProperty {
	t.Helper()
	b := bind(model, DefaultScope)
	b.realizing = true
	return migrationRefinedPropertiesOf(b)
}

func migrationRefinedPropertiesOf(b *binding) []migrationRefinedProperty {
	model := b.model
	var out []migrationRefinedProperty
	for _, machine := range model.GetMachines() {
		if machine.GetRefines() == nil {
			continue
		}
		scenarios := map[string]*ScenarioDecl{}
		for _, property := range model.GetProperties() {
			if property.GetMachine() != machine.GetRefines().GetProduct() {
				continue
			}
			entry := migrationRefinedProperty{Machine: machine.GetName(), Owner: property.GetMachine(), Name: property.GetName()}
			subject := b.subject(machine.GetName())
			if subject.err != nil {
				entry.Error = migrationError(subject.err)
				out = append(out, entry)
				continue
			}
			ref := b.refinement(subject)
			if ref.ref == nil {
				entry.Error = migrationError(ref.err)
				out = append(out, entry)
				continue
			}
			declared, err := b.property(property)
			if err != nil {
				entry.Error = migrationError(err)
				out = append(out, entry)
				continue
			}
			for _, row := range ref.source.Rows {
				result := entry
				result.Row = row.Key
				scenario := scenarios[row.Key]
				if scenario == nil {
					scenario = umpire.KeyScenario(ref.source, "row."+row.Key, row.Source, row.Action)
					scenarios[row.Key] = scenario
				}
				query := umpire.KeyVerifyRefined("row", declared, scenario, ref.ref, Limits{Name: "oneStep", Steps: 1, Actions: 1, Search: 64})
				answer, err := query.Answer()
				result.Outcome, result.About, result.Witness, result.Rows, result.Error = answer.Outcome, answer.Exercised, answer.Witness, answer.Rows, migrationError(err)
				out = append(out, result)
			}
		}
	}
	return out
}

func TestCaptureMigrationRefinedProperties(t *testing.T) {
	if *captureMigrationGoldens == "" {
		t.Skip("explicit -capture-goldens=<new directory> required")
	}
	_, models := migrationInputs(t)
	files := map[string][]byte{}
	for _, path := range slices.Sorted(maps.Keys(models)) {
		encoded, err := golden.JSON(migrationRefinedProperties(t, models[path]))
		require.NoError(t, err)
		files["refined-properties/"+migrationKey(path)] = encoded
	}
	require.NoError(t, golden.Capture(*captureMigrationGoldens, files))
}

func TestMigrationRefinedPropertiesCoverEveryProductPropertyRow(t *testing.T) {
	const key = "ir/activity.json"
	var expected []migrationRefinedProperty
	frozenReaderJSON(t, "refined-properties/"+key, &expected)
	require.NotEmpty(t, expected)
	frozen := frozenReaderModel(t, "activity")
	require.Equal(t, expected, migrationRefinedProperties(t, frozen), "the frozen input derives as it was frozen")
	// The rows of a Property a law replacement renames are derived again from the frozen input with the
	// rename, and compared with the current Model's without the generated claims the delta lists.
	original, err := golden.Attached(key, frozen)
	require.NoError(t, err)
	current, err := golden.Ungenerated(key, activityModel(t))
	require.NoError(t, err)
	require.Equal(t, migrationRefinedProperties(t, original), migrationRefinedProperties(t, current))
}

// locationMigrator maps the source paths a located error names. Its replacer is built once for all the
// strings of a comparison.
func locationMigrator(cfg golden.Config) func(string) string {
	replacements := []string{}
	for _, path := range cfg.Paths {
		replacements = append(replacements, path.Old+":", path.New+":")
	}
	return strings.NewReplacer(replacements...).Replace
}

func migrateSemanticLocations(s *migrationSemantics, replace func(string) string) {
	for i := range s.Subjects {
		migrateSubjectLocations(&s.Subjects[i], replace)
	}
	for i := range s.Properties {
		migratePropertyLocations(&s.Properties[i], replace)
	}
	migrateReceiptLocations(s.Receipts, replace)
}

func migrateSubjectLocations(subject *migrationSubject, replace func(string) string) {
	subject.Error = replace(subject.Error)
	if subject.Table != nil {
		for j := range subject.Table.Unknown {
			subject.Table.Unknown[j] = replace(subject.Table.Unknown[j])
		}
	}
}

func migratePropertyLocations(p *migrationProperty, replace func(string) string) {
	p.Error = replace(p.Error)
	for j := range p.Rows {
		p.Rows[j].Error = replace(p.Rows[j].Error)
	}
}

func migratePropertyMeaningLocations(p *migrationPropertyMeaning, replace func(string) string) {
	p.Error = replace(p.Error)
	for j := range p.Answers {
		p.Answers[j].Error = replace(p.Answers[j].Error)
	}
}

func migrateReceiptLocations(rows []migrationReceipt, replace func(string) string) {
	for i := range rows {
		r := &rows[i]
		r.Position, r.Explanation, r.Cause = replace(r.Position), replace(r.Explanation), replace(r.Cause)
		for j := range r.Holes {
			r.Holes[j].Position = replace(r.Holes[j].Position)
		}
		migrateReceiptLocations(r.Also, replace)
	}
}

// meaningDigests digests a binding's meaning one part at a time: the i-th stream reads each part with
// its located strings mapped by locates[0] through locates[i], in turn.
func meaningDigests(t *testing.T, b *binding, locates ...func(string) string) []*golden.Stream {
	t.Helper()
	return meaningDigestsInspected(t, b, nil, locates...)
}

// meaningDigestsInspected is meaningDigests that hands each Property, with the table its answers
// index, to inspect before it is digested, when inspect is not nil.
func meaningDigestsInspected(t *testing.T, b *binding, inspect func(*migrationPropertyMeaning, *Table), locates ...func(string) string) []*golden.Stream {
	t.Helper()
	streams := make([]*golden.Stream, len(locates))
	for i := range streams {
		streams[i] = &golden.Stream{Keep: true, Verbatim: true}
	}
	add := func(name string, v any, locate func(int)) {
		for i, s := range streams {
			locate(i)
			require.NoError(t, s.Add(name, v))
		}
	}
	receipts := migrationMeaningParts(b,
		func(x migrationSubject) {
			add("subject "+x.Name, &x, func(i int) { migrateSubjectLocations(&x, locates[i]) })
		},
		func(x migrationPropertyMeaning, table *Table) {
			if inspect != nil {
				inspect(&x, table)
			}
			add("property "+x.Owner+"."+x.Name, &x, func(i int) { migratePropertyMeaningLocations(&x, locates[i]) })
		})
	add("receipts", receipts, func(i int) { migrateReceiptLocations(receipts, locates[i]) })
	return streams
}

// requireSameMeaning compares two digested meanings and names the first part where they differ.
func requireSameMeaning(t *testing.T, want, got *golden.Stream) {
	t.Helper()
	if want.Digest() == got.Digest() {
		return
	}
	for i := range min(len(want.Parts), len(got.Parts)) {
		require.Equal(t, want.Parts[i], got.Parts[i], "the first part that differs")
	}
	require.Len(t, got.Parts, len(want.Parts))
	require.Fail(t, "the meanings differ in their framing")
}

// TestMigrationMeaningDetectsCompactMutations changes, one at a time, what the compact Property form
// leaves to its subject's table and what it records itself: each change fails the comparison of two
// interpretations, at the part that carries it.
func TestMigrationMeaningDetectsCompactMutations(t *testing.T) {
	m, err := Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	unmoved := func(s string) string { return s }
	// The Property mutated is the first one whose answers cover more than one action, on two rows or more.
	var property, owner string
	original := meaningDigestsInspected(t, migrationBinding(t, m), func(x *migrationPropertyMeaning, _ *Table) {
		if property == "" && len(x.About) > 1 && len(x.Answers) > 1 && x.Answers[0].Row != x.Answers[len(x.Answers)-1].Row {
			property, owner = x.Name, x.Owner
		}
	}, unmoved)[0]
	require.NotEmpty(t, property)
	same := meaningDigests(t, migrationBinding(t, m), unmoved)[0]
	require.Equal(t, original.Digest(), same.Digest(), "the meaning is deterministic")
	propertyPart, subjectPart := "property "+owner+"."+property, "subject "+owner

	answer := func(change func(*migrationPropertyMeaning, *Table)) func(*migrationPropertyMeaning, *Table) {
		return func(x *migrationPropertyMeaning, table *Table) {
			if x.Name == property && x.Owner == owner {
				change(x, table)
			}
		}
	}
	rows := func(change func([]Row) []Row) func(*binding) {
		return func(b *binding) {
			table := b.subject(owner).table
			table.Rows = change(slices.Clone(table.Rows))
		}
	}
	for _, mutation := range []struct {
		name     string
		table    func(*binding)
		property func(*migrationPropertyMeaning, *Table)
		part     string
	}{
		{name: "About set", part: propertyPart, property: answer(func(x *migrationPropertyMeaning, _ *Table) {
			x.About = x.About[1:]
		})},
		{name: "About set with its answers", part: propertyPart, property: answer(func(x *migrationPropertyMeaning, table *Table) {
			// The Property is no longer about its first action: that action's answers go with it.
			dropped := x.About[0]
			x.About = x.About[1:]
			x.Answers = slices.DeleteFunc(x.Answers, func(a migrationPropertyAnswer) bool { return table.Rows[a.Row].Action == dropped })
		})},
		{name: "Holds", part: propertyPart, property: answer(func(x *migrationPropertyMeaning, _ *Table) {
			x.Answers[0].Holds = !x.Answers[0].Holds
		})},
		{name: "Error", part: propertyPart, property: answer(func(x *migrationPropertyMeaning, _ *Table) {
			x.Answers[len(x.Answers)-1].Error += "changed"
		})},
		{name: "row order", part: subjectPart, table: rows(func(r []Row) []Row {
			r[0], r[len(r)-1] = r[len(r)-1], r[0]
			return r
		})},
		{name: "row count", part: subjectPart, table: rows(func(r []Row) []Row { return r[:len(r)-1] })},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			b := migrationBinding(t, m)
			if mutation.table != nil {
				mutation.table(b)
			}
			changed := meaningDigestsInspected(t, b, mutation.property, unmoved)[0]
			require.NotEqual(t, original.Digest(), changed.Digest())
			require.Len(t, changed.Parts, len(original.Parts))
			first := 0
			for first < len(original.Parts) && original.Parts[first] == changed.Parts[first] {
				first++
			}
			require.Less(t, first, len(original.Parts))
			require.Equal(t, mutation.part, original.Parts[first].Name, "the first part that differs")
		})
	}
}

func TestMigrationGoldenReadsPropertiesWithNoQuery(t *testing.T) {
	model, err := Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	model.Queries = nil
	before := migrationMeaning(t, model)
	require.NotEmpty(t, before.Properties)
	target := model.GetProperties()[0]
	function := functionNamed(model, target.GetHolds())
	require.NotNil(t, function)
	function.Body = expr(boolValue(false))
	after := migrationMeaning(t, model)
	expected, err := golden.JSON(before.Properties)
	require.NoError(t, err)
	actual, err := golden.JSON(after.Properties)
	require.NoError(t, err)
	require.Error(t, golden.Compare(map[string][]byte{"properties": expected}, map[string][]byte{"properties": actual}))
}

func TestMigrationGoldenPreservesEvidenceOrder(t *testing.T) {
	model, err := Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	table := built(t, model)["nexusProtocol"].Table
	before := migrationTableOf(table)
	require.GreaterOrEqual(t, len(table.Evidence), 2)
	original, err := golden.JSON(before)
	require.NoError(t, err)
	table.Evidence = slices.Clone(table.Evidence)
	table.Evidence[0], table.Evidence[1] = table.Evidence[1], table.Evidence[0]
	mutated, err := golden.JSON(migrationTableOf(table))
	require.NoError(t, err)
	require.Error(t, golden.Compare(map[string][]byte{"table": original}, map[string][]byte{"table": mutated}))
}

type migrationDefinition struct {
	Kind, Owner, Name, ID, Canonical, Fingerprint, Error string
}

func migrationDefinitions(t *testing.T, model *umpirespb.Model) []migrationDefinition {
	t.Helper()
	b := bind(model, DefaultScope)
	b.realizing = true
	return migrationDefinitionsOf(t, b)
}

func migrationDefinitionsOf(t *testing.T, b *binding) []migrationDefinition {
	t.Helper()
	model := b.model
	var out []migrationDefinition
	for _, p := range model.GetProperties() {
		subject := b.subject(p.GetMachine())
		entry := migrationDefinition{Kind: "property", Owner: p.GetMachine(), Name: p.GetName(), ID: subject.family + ".property." + p.GetName()}
		declared, err := b.property(p)
		if err == nil {
			var groups []Group
			groups, err = declared.Lower()
			if err == nil {
				entry.Canonical = subject.table.PropertySemantic(declared.PropertyID(subject.table), groups)
			}
		}
		entry.Error = migrationError(err)
		if entry.Canonical != "" {
			entry.Fingerprint = umpire.Fingerprint(entry.Canonical)
		}
		out = append(out, entry)
	}
	for _, s := range model.GetScenarios() {
		subject := b.subject(s.GetMachine())
		entry := migrationDefinition{Kind: "scenario", Owner: s.GetMachine(), Name: s.GetName(), ID: subject.family + ".behavior." + s.GetName()}
		if subject.err != nil {
			entry.Error = migrationError(subject.err)
		} else {
			declared, err := b.scenario(s, subject, subject.table)
			entry.Error = migrationError(err)
			if err == nil {
				entry.Canonical = declared.ScenarioSemantic(subject.table)
				entry.Fingerprint = umpire.Fingerprint(entry.Canonical)
			}
		}
		out = append(out, entry)
	}
	for _, q := range model.GetQueries() {
		subject := b.subject(q.GetScenario().GetMachine())
		entry := migrationDefinition{Kind: "query", Owner: subject.name, Name: q.GetName(), ID: subject.family + ".query." + q.GetName()}
		bound, err := b.query(q)
		if err == nil {
			var groups []Group
			groups, err = bound.q.Property.Lower()
			if err == nil {
				propertyTable, tableErr := bound.q.Property.Machine.Table()
				require.NoError(t, tableErr)
				fingerprint := umpire.Fingerprint(propertyTable.PropertySemantic(bound.q.Property.PropertyID(propertyTable), groups))
				entry.Canonical = bound.q.QueryCanonicalOf(bound.table, fingerprint, bound.table.TargetFingerprint())
				entry.Fingerprint = umpire.Fingerprint(entry.Canonical)
			}
		}
		entry.Error = migrationError(err)
		out = append(out, entry)
	}
	return out
}

func TestCaptureMigrationDeclarations(t *testing.T) {
	if *captureMigrationGoldens == "" {
		t.Skip("explicit -capture-goldens=<new directory> required")
	}
	_, models := migrationInputs(t)
	files := map[string][]byte{}
	for _, path := range slices.Sorted(maps.Keys(models)) {
		encoded, err := golden.JSON(migrationDefinitions(t, models[path]))
		require.NoError(t, err)
		files["declarations/"+migrationKey(path)] = encoded
	}
	require.NoError(t, golden.Capture(*captureMigrationGoldens, files))
}

func TestMigrationDefinitionGoldenIncludesUnrealizedClaims(t *testing.T) {
	model, err := Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	definitions := migrationDefinitions(t, model)
	require.Len(t, definitions, len(model.GetProperties())+len(model.GetScenarios())+len(model.GetQueries()))
	before, err := golden.JSON(definitions)
	require.NoError(t, err)
	model.Queries = nil
	changed := migrationDefinitions(t, model)
	require.Len(t, changed, len(model.GetProperties())+len(model.GetScenarios()))
	after, err := golden.JSON(changed)
	require.NoError(t, err)
	require.Error(t, golden.Compare(map[string][]byte{"declarations": before}, map[string][]byte{"declarations": after}))
	// A composition's claim and a transition claim are searched and verified, never realized, so they
	// have no fingerprint; every other declaration has one.
	neverRealized := map[string]bool{"repliedByPollingWorker": true, "terminalIsFinal": true}
	for _, entry := range changed {
		require.NotEmpty(t, entry.ID)
		if entry.Kind == "property" && neverRealized[entry.Name] {
			require.Empty(t, entry.Fingerprint, entry.Name)
			require.Contains(t, entry.Error, "is searched and verified, never realized", entry.Name)
			continue
		}
		require.NotEmpty(t, entry.Fingerprint, entry.Name)
	}
	for _, mutate := range []func(*migrationDefinition){
		func(d *migrationDefinition) { d.ID += "-changed" },
		func(d *migrationDefinition) { d.Fingerprint += "-changed" },
		func(d *migrationDefinition) { d.Canonical += "-changed" },
	} {
		mutated := slices.Clone(definitions)
		mutate(&mutated[0])
		encoded, err := golden.JSON(mutated)
		require.NoError(t, err)
		require.Error(t, golden.Compare(map[string][]byte{"declarations": before}, map[string][]byte{"declarations": encoded}))
	}
}
