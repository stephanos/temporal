package model

import (
	"cmp"
	"flag"
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
	Rejected   string
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
	m := b.model
	out := migrationSemantics{Receipts: migrationReceipts(Check(m, DefaultScope).Receipts)}
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
			entry.Refinement, entry.Rejected = mm.Refinement, migrationError(mm.Rejected)
			if table == nil {
				table = mm.Table
			}
		}
		if table != nil {
			entry.Table = migrationTableOf(table)
		}
		out.Subjects = append(out.Subjects, entry)
	}
	for _, p := range m.GetProperties() {
		subject := b.subject(p.GetMachine())
		entry := migrationProperty{Owner: p.GetMachine(), Name: p.GetName(), ID: subject.family + ".property." + p.GetName()}
		if subject.err != nil {
			entry.Error = migrationError(subject.err)
		} else {
			reading, err := b.propertyReads(subject, p)
			if err != nil {
				entry.Error = migrationError(err)
			} else {
				bound := boundProperty(p, reading)
				for _, row := range subject.table.Rows {
					for i, result := range row.Results {
						answer := migrationPropertyRow{Row: row.Key, Result: i, About: bound.About(row.Action)}
						if answer.About {
							answer.Holds, err = bound.Holds(row.Source, result)
							answer.Error = migrationError(err)
						}
						entry.Rows = append(entry.Rows, answer)
					}
				}
			}
		}
		out.Properties = append(out.Properties, entry)
	}
	return out
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
	for path, current := range models {
		original := new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(expected["inputs/"+migrationKey(path)], original), path)
		_, err := cfg.Match(original, current)
		require.NoError(t, err, path)
		// Evaluate the immutable source spelling so located errors remain byte-comparable after a move.
		models[path] = original
	}
	require.NoError(t, cfg.FunctionsRenamed(models))
	require.NoError(t, golden.Compare(expected, migrationFiles(t, models)))
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
			original := new(umpirespb.Model)
			require.NoError(t, protojson.Unmarshal(originals[migrationKey(path)], original))
			mapped, err := cfg.Migrate(original)
			require.NoError(t, err)
			// The original and the mapped Model are each interpreted once, and apart.
			read, readMapped := migrationBinding(t, original), migrationBinding(t, mapped)
			migrate := locationMigrator(cfg)
			before := migrationMeaningOf(read)
			after := migrationMeaningOf(readMapped)
			migrateSemanticLocations(&before, migrate)
			want, err := golden.JSON(before)
			require.NoError(t, err)
			got, err := golden.JSON(after)
			require.NoError(t, err)
			require.NoError(t, golden.Compare(map[string][]byte{path: want}, map[string][]byte{path: got}))
			definitions := migrationDefinitionsOf(t, read)
			for i := range definitions {
				definitions[i].Error = migrate(definitions[i].Error)
			}
			require.Equal(t, definitions, migrationDefinitionsOf(t, readMapped))
			refined := migrationRefinedPropertiesOf(read)
			for i := range refined {
				refined[i].Error = migrate(refined[i].Error)
			}
			require.Equal(t, refined, migrationRefinedPropertiesOf(readMapped))
			// The current IR, which Match admits under the projection, reads as the original does. The
			// projection names a path the same in each spelling, so it applies over the migration.
			project := locationProjection(cfg)
			frozen := projectedSnapshots(t, before, definitions, refined, project)
			require.Equal(t, string(frozen), string(projectedMeaning(t, migrationBinding(t, models[path]), project)))
		})
	}
}

// locationProjection maps every source path a located string names, frozen or current, to its
// current spelling, and drops the line and column after it, as Match compares positions by file.
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
	files := slices.SortedFunc(maps.Keys(current), func(x, y string) int { return cmp.Or(len(y)-len(x), strings.Compare(x, y)) })
	for i := range files {
		files[i] = regexp.QuoteMeta(files[i])
	}
	located := regexp.MustCompile(`(` + strings.Join(files, "|") + `)(?::[0-9]+)*`)
	return func(s string) string {
		return located.ReplaceAllStringFunc(s, func(at string) string { return current[located.FindStringSubmatch(at)[1]] })
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
	current := models[path]
	require.NotNil(t, functionNamed(original, kernel))
	moved := "temporal.nexuscaller.Protocol$.completeStep"
	if i := slices.IndexFunc(cfg.Projection.Functions, func(s golden.Substitution) bool { return s.Old == kernel }); i >= 0 {
		moved = cfg.Projection.Functions[i].New
	} else {
		cfg.Projection.Functions = append(cfg.Projection.Functions, golden.Substitution{Old: kernel, New: moved})
	}
	require.NoError(t, cfg.FunctionsRenamed(map[string]*umpirespb.Model{path: original}))
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
	_, err = cfg.Match(original, admitted)
	require.NoError(t, err)
	project := locationProjection(cfg)
	want := projectedMeaning(t, migrationBinding(t, original), project)
	require.Equal(t, string(want), string(projectedMeaning(t, migrationBinding(t, admitted), project)),
		"what the projection admits reads as the frozen original does")

	t.Run("changed table row", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		step := functionNamed(changed, changed.GetMachines()[0].GetSteps()[0].GetFunction())
		branch := firstIf(step.ProtoReflect())
		require.NotNil(t, branch)
		branch.Then, branch.Else = branch.Else, branch.Then
		_, err := cfg.Match(original, changed)
		require.Error(t, err)
		require.NotEqual(t, string(want), string(projectedMeaning(t, migrationBinding(t, changed), project)))
	})
	for name, rewrite := range map[string]func(string) string{
		"unlisted function rename":      rename(moved, moved+"Unlisted"),
		"listed function rename unmade": rename(moved, kernel),
		"position in an unlisted file":  rename("model/temporal/nexuscaller/Claims.scala", "model/temporal/nexuscaller/Claim.scala"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cfg.Match(original, migrationRewrite(t, admitted, rewrite))
			require.Error(t, err)
		})
	}
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
		_, err := cfg.Match(original, changed)
		require.Error(t, err, swapped.GetName())
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
	var expected []migrationRefinedProperty
	frozenReaderJSON(t, "refined-properties/ir/activity.json", &expected)
	require.NotEmpty(t, expected)
	require.Equal(t, expected, migrationRefinedProperties(t, activityModel(t)))
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
		subject := &s.Subjects[i]
		subject.Error, subject.Rejected = replace(subject.Error), replace(subject.Rejected)
		if subject.Table != nil {
			for j := range subject.Table.Unknown {
				subject.Table.Unknown[j] = replace(subject.Table.Unknown[j])
			}
		}
	}
	for i := range s.Properties {
		p := &s.Properties[i]
		p.Error = replace(p.Error)
		for j := range p.Rows {
			p.Rows[j].Error = replace(p.Rows[j].Error)
		}
	}
	var receipts func([]migrationReceipt)
	receipts = func(rows []migrationReceipt) {
		for i := range rows {
			r := &rows[i]
			r.Position, r.Explanation, r.Cause = replace(r.Position), replace(r.Explanation), replace(r.Cause)
			for j := range r.Holes {
				r.Holes[j].Position = replace(r.Holes[j].Position)
			}
			receipts(r.Also)
		}
	}
	receipts(s.Receipts)
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
				entry.Canonical = bound.q.QueryCanonical(bound.table, fingerprint)
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
