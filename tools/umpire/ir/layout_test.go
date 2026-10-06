package ir

// model/'s folders say what they hold (fn-114.9): the IR generator is model/irgen, the model check
// model/check, the check's build cache model/build, and the Temporal Models are grouped into
// model/temporal/features and model/temporal/shared. The folders they replaced must not come back:
// neither as a directory with sources nor as a path, package or target a live file names. Nor may
// the per-kind files fn-126 folded into one feature file per folder: no Model folder holds one, and
// no live file names one. Nor may the zoom-in folders fn-126 flattened into a feature's level
// folders (decisions 16 and 22): record/ and withTaskQueue/ are files of standaloneactivity/system/,
// closepolicy/ one of nexuscaller/system/.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// retiredModelDirectories are the folders fn-114.9 moved. model/gen, the old build cache, is not
// listed: it is ignored and a checkout may still hold it until it is cleaned, so only a path that
// names it fails.
var retiredModelDirectories = []string{
	"model/lifter", "model/gate", "model/metrics",
	"model/temporal/standaloneactivity", "model/temporal/nexuscaller", "model/temporal/nexusoperation",
	"model/temporal/taskqueue", "model/temporal/worker",
	"model/temporal/features/standaloneactivity/record", "model/temporal/features/standaloneactivity/withTaskQueue",
	"model/temporal/features/nexuscaller/closepolicy",
}

// retiredModelNames matches a retired folder's path from the repository's root, a retired Scala
// package of the tools or of a flattened zoom-in, and a retired Make target. A Definition ID or IR
// type name keeps its old package by design (each Model's DefinitionScope pin, such as
// `temporal.nexuscaller.closepolicy.Model$package$`), so a pinned name in a string is no match; a
// function symbol, which names the package the function is in, is.
var retiredModelNames = regexp.MustCompile(strings.Join([]string{
	`model/(?:lifter|gate|metrics|gen)\b`,
	`model/temporal/(?:standaloneactivity|nexuscaller|nexusoperation|taskqueue|worker)\b`,
	`model/temporal/features/(?:standaloneactivity/(?:record|withTaskQueue)|nexuscaller/closepolicy)\b`,
	`\bfeatures\.(?:standaloneactivity\.(?:record|withTaskQueue)|nexuscaller\.closepolicy)\b`,
	`\bumpire\.(?:lift|gate)\b`,
	`\blint-model-(?:lifter|lifts|gate|metrics)\b`,
	// The same paths joined from their parts, as a Go test reads a file.
	`"model", "(?:lifter|gate|metrics|gen)"`,
	`"model", "temporal", "(?:standaloneactivity|nexuscaller|nexusoperation|taskqueue|worker)"`,
	`"standaloneactivity", "(?:record|withTaskQueue)"|"nexuscaller", "closepolicy"`,
	`\bstandaloneactivity/Realization\.scala\b`,
	`"standaloneactivity", "Realization\.scala"`,
}, "|"))

// retiredZoomInFolders matches a flattened zoom-in folder named from its feature's folder, as prose
// names it: `record/`, `withTaskQueue/` or `closepolicy/`, not as the tail of a longer path. A bare
// `record/` is also ordinary prose elsewhere ("the record/replay harness"), so it is read only in
// model/ and the layout documents (zoomInProse); every other live file is held to
// retiredZoomInMarked: the folder in backticks or under its feature's folder.
var retiredZoomInFolders = regexp.MustCompile(`(?:^|[^\w/.-])(?:record|withTaskQueue|closepolicy)/`)

// retiredZoomInMarked matches a flattened zoom-in folder that no prose names by accident: in
// backticks (`record/`) or under its feature's folder (standaloneactivity/record).
var retiredZoomInMarked = regexp.MustCompile(
	"`(?:record|withTaskQueue|closepolicy)/|\\b(?:standaloneactivity/(?:record|withTaskQueue)|nexuscaller/closepolicy)\\b")

// zoomInProse are the live files outside model/ that describe the Models' layout in prose.
var zoomInProse = []string{"AGENTS.md", ".plans/UMPIRE_MODULES.md", ".plans/UMPIRE4_VISION.md"}

// retiredModelRelatives matches, inside model/, a moved folder named from model/: temporal/taskqueue.
// Outside model/ the same words name other trees, such as Testpilot's temporal/worker package.
var retiredModelRelatives = regexp.MustCompile(
	`(?:^|[^\w/.-])temporal/(?:standaloneactivity|nexuscaller|nexusoperation|taskqueue|worker)\b`)

// retiredScalaPackages matches, in a Scala source, a package clause or import of a moved Temporal
// package: `package taskqueue` after `package temporal`, or `import temporal.worker.*`; or of a
// flattened zoom-in, `package record` after `package features.standaloneactivity`, or `import
// closepolicy.*`.
var retiredScalaPackages = regexp.MustCompile(
	`^\s*(?:package (?:temporal\.)?|import (?:temporal\.)?)(?:standaloneactivity|nexuscaller|nexusoperation|taskqueue|worker)\b|` +
		`^\s*(?:package|import) (?:(?:temporal\.)?features\.(?:standaloneactivity|nexuscaller)\.)?(?:record|withTaskQueue|closepolicy)\b`)

// retiredFeatureFileNames are the per-kind files a Model folder held before fn-126 laid each folder
// out as one feature file named after it (model/README.md, "The reading order and its lint").
var retiredFeatureFileNames = []string{"Model.scala", "Properties.scala", "Queries.scala", "Capabilities.scala", "IrFiles.scala"}

// modelFolderRoots hold the Model folders. The kit's model/temporal/capabilities/Capabilities.scala
// is no Model folder's file, and the IR generator's fixtures name their files as they like.
var modelFolderRoots = []string{"model/temporal/features", "model/temporal/shared"}

// retiredFeatureFilePaths matches a retired per-kind file of a Model folder named by its path: from
// features/ or shared/, from a Model folder's name (`worker/Model.scala`) or a level folder's
// (`system/Model.scala`), or joined from its parts.
var retiredFeatureFilePaths = regexp.MustCompile(strings.Join([]string{
	`\b(?:features|shared|standaloneactivity|record|withTaskQueue|admission|compositions|nexuscaller|closepolicy|nexusoperation|taskqueue|worker|product|system)/(?:[\w-]+/)*(?:Model|Properties|Queries|Capabilities|IrFiles)\.scala\b`,
	`"(?:features|shared)"(?:, "[\w-]+")*, "(?:Model|Properties|Queries|Capabilities|IrFiles)\.scala"`,
}, "|"))

// retiredFeatureFileProse matches a retired per-kind file named bare in prose, the way a Model
// folder's file was named: owned by a folder, a feature or a Model ("its Model folder's
// Capabilities.scala"), after "own" ("its own `Capabilities.scala`"), or after a preposition ("read
// off Properties.scala", "in `Queries.scala`").
var retiredFeatureFileProse = regexp.MustCompile(
	"(?:\\b(?:folder|feature|[Mm]odel)'s|\\b(?:off|in|from|beside|see|own)) `?(?:Model|Properties|Queries|Capabilities|IrFiles)\\.scala\\b")

// ownCapabilities are the files that keep the name Capabilities.scala on purpose: the framework's
// mechanism, the kit's capability kinds and the IR generator's expansion and fixture. A file of
// theirs, or a line that names their folder, may name Capabilities.scala bare.
var ownCapabilities = regexp.MustCompile(`\b(?:model/umpire|temporal/capabilities|model/irgen|irgen/testdata)\b`)

// retiredFeatureFileProseIn reports whether a line of the file at path names a retired per-kind file
// bare, apart from Capabilities.scala where its own folders name it.
func retiredFeatureFileProseIn(path, line string) bool {
	for _, found := range retiredFeatureFileProse.FindAllString(line, -1) {
		own := strings.HasSuffix(found, "Capabilities.scala") && (ownCapabilities.MatchString(path) || ownCapabilities.MatchString(line))
		if !own {
			return true
		}
	}
	return false
}

// retiredFeatureFiles lists the files under root's Model folders that bear a retired per-kind name.
func retiredFeatureFiles(root string) ([]string, error) {
	var found []string
	for _, dir := range modelFolderRoots {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == ".scala-build" || d.Name() == ".bsp") {
				return filepath.SkipDir
			}
			if !d.IsDir() && slices.Contains(retiredFeatureFileNames, d.Name()) {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				found = append(found, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return found, nil
}

// liveLayoutRoots are the live files and trees outside model/ that name model paths: the tools that
// read the model, the build, CI, and the documents that describe the layout.
var liveLayoutRoots = []string{
	"tools/umpire", "tools/canary", "common/testing/testpilot", "tests/testcore/testpilot",
	"api/umpire", "proto/internal/temporal/server/api/umpire",
	"Makefile", ".github/workflows", "AGENTS.md", ".plans/UMPIRE_MODULES.md", ".plans/UMPIRE4_VISION.md",
}

// retiredModelVocabulary is the task-8 rename batch. It is word-bounded so the variants that keep
// their names, such as LossyMatchingQueue, are not caught by the MatchingQueue entry. The fault
// kind admissionResponseLoss and the forgedCompletion Query keep their names and are deliberately
// absent.
var retiredModelVocabulary = regexp.MustCompile(`\b(?:` + strings.Join([]string{
	"ActivityProtocol", "activityProtocol", "NexusProtocol", "nexusProtocol",
	"ProtocolState", "ProtocolFact", "ProtocolStep",
	"CurrentAdmission", "currentAdmission", "StaleAdmission", "staleAdmission",
	"HeldAdmission", "heldAdmission", "AdmissionResponseLoss",
	"CurrentRecord", "currentRecord", "StaleRecord", "staleRecord",
	"CurrentOverQueue", "currentOverQueue", "StaleOverQueue", "staleOverQueue",
	"CurrentOverMatching", "currentOverMatching", "StaleOverMatching", "staleOverMatching",
	"CurrentOverForgetful", "currentOverForgetful", "CurrentOverVolatile", "currentOverVolatile",
	"CurrentOverLossyMatching", "currentOverLossyMatching",
	"DispatchQueue", "dispatchQueue", "DispatchQueueUnderStorageLoss", "dispatchQueueUnderStorageLoss",
	"MatchingQueue", "matchingQueue",
	"attemptStart", "attemptResult", "answerDelivery", "handlerReply", "transportFault",
	"callerClose", "handlerFinish", "workerStop", "workerResume", "ForgedCompletion",
}, "|") + `)\b`)

var retiredVocabularyRoots = []string{
	"tools/umpire", "tools/canary", "tests/testcore/testpilot", "common/testing/testpilot",
	"model", "AGENTS.md", ".plans/UMPIRE_MODULES.md",
	".plans/UMPIRE4_VISION.md", ".plans/DSL_OPERATORS.md", ".plans/DSL_SIMPLIFICATION.md",
	".plans/QUINT_MODULE_LAYOUT.md",
}

// These are historical Cases retained to prove compatibility with another model or with a prior
// pinned canary. They are archives, not live artifacts generated from model/temporal.
var retiredVocabularyArchives = map[string]bool{
	"tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-case.json": true,
	"tests/testcore/testpilot/testdata/nexusPairTests-bothComplete-case.json":     true,
	"tests/testcore/testpilot/testdata/workerOutageTests-survived-case.json":      true,
}

var retiredLevelProse = regexp.MustCompile(`(?i)\b(?:protocol machine|system contract)\b`)

func withoutKeptRuntimeVocabulary(line string) string {
	for _, kept := range []string{
		"FaultKind.workerStop", "FaultKind.workerResume", "FaultKind.admissionResponseLoss",
		"case workerStop, workerResume, admissionResponseLoss", "CauseKind.handlerReply",
		"case handlerReply", "cause.handlerReply", "visibility.handlerReply",
	} {
		line = strings.ReplaceAll(line, kept, "")
	}
	return line
}

func TestRetiredModelVocabularyStaysRetired(t *testing.T) {
	self := filepath.ToSlash(filepath.Join("tools", "umpire", "ir", "layout_test.go"))
	var mentions []string
	collect := func(rel, content string) {
		if rel == self || retiredVocabularyArchives[rel] || strings.HasSuffix(rel, ".semanticdb") {
			return
		}
		for i, line := range strings.Split(content, "\n") {
			line = withoutKeptRuntimeVocabulary(line)
			if retiredModelVocabulary.MatchString(line) ||
				(strings.HasPrefix(rel, "model/temporal/") && strings.Contains(line, `scenario("`)) ||
				((rel == "model/README.md" || rel == "model/SEMANTICS.md" || strings.HasPrefix(rel, ".plans/")) &&
					retiredLevelProse.MatchString(line)) {
				mentions = append(mentions, fmt.Sprintf("%s:%d", rel, i+1))
			}
		}
	}
	for _, root := range retiredVocabularyRoots {
		require.NoError(t, filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if root == "model" && path != filepath.Join(repoRoot, root) &&
					(d.Name() == "build" || strings.HasPrefix(d.Name(), ".")) {
					return fs.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			collect(filepath.ToSlash(rel), string(content))
			return nil
		}))
	}
	if len(mentions) != 0 {
		t.Fatalf("use the Product/System and task-8 vocabulary in live Model surfaces: %v", mentions)
	}
}

func TestRetiredModelVocabularyMentionsAreFound(t *testing.T) {
	for _, old := range []string{"ActivityProtocol", "activityProtocol", "ProtocolState", "CurrentAdmission", "attemptStart", "ForgedCompletion"} {
		require.True(t, retiredModelVocabulary.MatchString(old), old)
		require.False(t, retiredModelVocabulary.MatchString("x"+old), old)
	}
	require.False(t, retiredModelVocabulary.MatchString("LossyMatchingQueue"))
	require.False(t, retiredModelVocabulary.MatchString("admissionResponseLoss"))
	require.False(t, retiredModelVocabulary.MatchString("forgedCompletion"))
}

func TestRetiredVocabularyCoversTheWholeModelTree(t *testing.T) {
	require.Contains(t, retiredVocabularyRoots, "model")
}

func retiredModelMentions(path, content string) []string {
	var found []string
	scala, inModel := strings.HasSuffix(path, ".scala"), strings.HasPrefix(path, modelRoot+"/")
	zoomIn := retiredZoomInMarked
	if inModel || slices.Contains(zoomInProse, path) {
		zoomIn = retiredZoomInFolders
	}
	for i, line := range strings.Split(content, "\n") {
		if retiredModelNames.MatchString(line) || inModel && retiredModelRelatives.MatchString(line) ||
			zoomIn.MatchString(line) ||
			scala && retiredScalaPackages.MatchString(line) || retiredFeatureFilePaths.MatchString(line) ||
			retiredFeatureFileProseIn(path, line) {
			found = append(found, fmt.Sprintf("%s:%d", path, i+1))
		}
	}
	return found
}

// liveLayoutFiles visits every file under liveLayoutRoots but this one.
func liveLayoutFiles(visit func(rel, content string)) error {
	self := filepath.ToSlash(filepath.Join("tools", "umpire", "ir", "layout_test.go"))
	for _, root := range liveLayoutRoots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if d.Name() == ".scala-build" || d.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if rel == self {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			visit(rel, string(content))
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func TestRetiredModelPathsStayRetired(t *testing.T) {
	var present []string
	for _, dir := range retiredModelDirectories {
		err := filepath.WalkDir(filepath.Join(repoRoot, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// scala-cli and Bloop may leave build state behind a moved project; it holds no source.
			if d.IsDir() && (d.Name() == ".scala-build" || d.Name() == ".bsp") {
				return filepath.SkipDir
			}
			if !d.IsDir() && !strings.HasSuffix(path, ".semanticdb") {
				present = append(present, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			require.NoError(t, err)
		}
	}
	require.Empty(t, present, "these folders moved (fn-114.9): model/irgen, model/check and model/temporal/{features,shared} hold them")
	retired, err := retiredFeatureFiles(repoRoot)
	require.NoError(t, err)
	require.Empty(t, retired, "a Model folder holds one feature file named after it (fn-126), not per-kind files")

	scanned := 0
	var mentions []string
	collect := func(rel, content string) {
		// scalafix's semantic databases beside the check's sources are ignored build output.
		if strings.HasSuffix(rel, ".semanticdb") {
			return
		}
		scanned++
		mentions = append(mentions, retiredModelMentions(rel, content)...)
	}
	require.NoError(t, modelFiles(collect))
	require.NoError(t, liveLayoutFiles(collect))
	require.Greater(t, scanned, 500, "the walk reaches the model, the tools, the build and the documents")
	require.Empty(t, mentions, "name the folders, packages and targets fn-114.9 and fn-126 renamed by their new names")
}

func TestRetiredModelMentionsAreFound(t *testing.T) {
	for name, test := range map[string]struct {
		content string
		found   bool
	}{
		"the IR generator's old folder":         {"see model/lifter/Lift.scala", true},
		"the check's old folder":                {"run model/gate", true},
		"the metrics project":                   {"scala-cli run model/metrics --", true},
		"the old build cache":                   {"model/gen/ir-scalapb.jar", true},
		"a moved feature":                       {"model/temporal/nexuscaller/Model.scala:12", true},
		"a moved shared part":                   {"(temporal/taskqueue)", true},
		"a tool package":                        {"import umpire.lift.Syntax", true},
		"the check's old package":               {"package umpire.gate", true},
		"a moved package clause":                {"package standaloneactivity", true},
		"an import of a moved package":          {"import temporal.worker.*", true},
		"a retired target":                      {"make lint-model-lifts", true},
		"a path joined from its parts":          {`filepath.Join("..", "model", "temporal", "worker")`, true},
		"the new folders":                       {"model/irgen, model/check, model/build, model/temporal/features/nexuscaller", false},
		"a shared part's new folder":            {"model/temporal/shared/taskqueue", false},
		"the new packages":                      {"package umpire.irgen\nimport temporal.features.standaloneactivity.*", false},
		"a pinned Definition ID":                {`DefinitionScope("temporal.standaloneactivity.System$package$")`, false},
		"Testpilot's worker":                    {"common/testing/testpilot/temporal/worker/interpreter.go", false},
		"a word that starts like a stem":        {"model/generated, model/gates, lint-model-irgen-lifts", false},
		"the IR generator's lifted files":       {"umpire.lifted", false},
		"a relative import of a moved package":  {"import worker.Phase as WorkerPhase", true},
		"a retired Model.scala":                 {"model/temporal/features/nexuscaller/Model.scala:424", true},
		"a retired Properties.scala":            {"see `nexuscaller/Properties.scala`, line 22", true},
		"a retired Queries.scala":               {"model/temporal/shared/taskqueue/Queries.scala", true},
		"a retired Capabilities.scala":          {"its folder's features/nexusoperation/Capabilities.scala", true},
		"a retired IrFiles.scala":               {"closepolicy/IrFiles.scala", true},
		"a retired file from its folder":        {"from worker/Model.scala", true},
		"a retired file joined from its parts":  {`filepath.Join("model", "temporal", "features", "nexuscaller", "Queries.scala")`, true},
		"a retired file of a level folder":      {"model/temporal/features/nexuscaller/system/Model.scala", true},
		"a retired file of a product folder":    {"the queue's product/Properties.scala", true},
		"a flattened record folder":             {"model/temporal/features/standaloneactivity/record/Record.scala", true},
		"a flattened composition folder":        {"its withTaskQueue/ composes the record", true},
		"a flattened close policy folder":       {"model/temporal/features/nexuscaller/closepolicy", true},
		"a zoom-in folder named bare":           {"read off record/ (heldDispatch)", true},
		"a zoom-in folder joined from parts":    {`filepath.Join("model", "temporal", "features", "nexuscaller", "closepolicy", "*.scala")`, true},
		"a flattened package clause":            {"package closepolicy", true},
		"an import of a flattened package":      {"import record.*", true},
		"a qualified flattened package":         {"features.nexuscaller.closepolicy.exports", true},
		"a function of a flattened package":     {"temporal.features.standaloneactivity.withTaskQueue.RecordOverQueue$.queries$.all", true},
		"the level folders' files":              {"model/temporal/features/standaloneactivity/system/Record.scala, product/Product.scala", false},
		"a level package":                       {"package system\nimport temporal.features.nexuscaller.system.ClosePolicyFamily", false},
		"a pinned former owner":                 {`DefinitionScope("temporal.nexuscaller.closepolicy.Model$package$")`, false},
		"a longer path ending in a folder name": {"common/testing/testpilot/record/run.go, x.record/y", false},
		"a feature file":                        {"model/temporal/features/nexuscaller/NexusCaller.scala:424", false},
		"the activity's former realization":     {"model/temporal/features/standaloneactivity/Realization.scala:33", true},
		"the activity's relative realization":   {"standaloneactivity/Realization.scala", true},
		"the former realization joined":         {`filepath.Join("standaloneactivity", "Realization.scala")`, true},
		"the System realization":                {"standaloneactivity/system/Realization.scala", false},
		"the Nexus caller realization":          {"nexuscaller/Realization.scala", false},
		"the kit's capabilities":                {"model/temporal/capabilities/Capabilities.scala", false},
		"the framework's capabilities":          {"model/umpire/Capabilities.scala", false},
		"a fixture named like a retired file":   {"model/irgen/testdata/lifts/Capabilities.scala, irFileRefusals/IrFiles.scala", false},
		"a folder's file named bare":            {"in its Model folder's Capabilities.scala", true},
		"a feature's file named bare":           {"the feature's `IrFiles.scala`", true},
		"a Model's own file named bare":         {"its own `Capabilities.scala`", true},
		"a file read off by name":               {"The claims are read off Properties.scala", true},
		"a file named after a preposition":      {"declared in `Queries.scala`, beside Model.scala", true},
		"the framework's Capabilities.scala":    {"`cited` in `Capabilities.scala` (model/umpire)", false},
		"the kit's Capabilities.scala":          {"the kinds in temporal/capabilities, in Capabilities.scala", false},
		"a retired file beside the kit's":       {"model/umpire keeps Capabilities.scala; see Queries.scala", true},
		"a file named by its kind alone":        {"no file named by kind (`Model.scala`, `Properties.scala`)", false},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.found, len(retiredModelMentions("model/temporal/A.scala", test.content)) > 0, test.content)
		})
	}
	// The kit's own files name their Capabilities.scala bare, as no other file may.
	require.Empty(t, retiredModelMentions("model/temporal/capabilities/Catalog.scala", "brought in Capabilities.scala"))
	require.NotEmpty(t, retiredModelMentions("model/temporal/capabilities/Catalog.scala", "read off Properties.scala"))
	// A Go package of Testpilot's is named like a moved Scala package, and its folder like a moved
	// folder named from model/; neither is one.
	require.Empty(t, retiredModelMentions("common/testing/testpilot/temporal/worker/api.go", "package worker"))
	require.Empty(t, retiredModelMentions("common/testing/testpilot/README.md", "(`temporal/worker/outage.go`)"))
	// A bare zoom-in folder name is prose outside model/ and the layout documents, unless it is
	// marked as a path.
	require.Empty(t, retiredModelMentions("common/testing/testpilot/README.md", "the record/replay harness"))
	require.Empty(t, retiredModelMentions("tools/umpire/lower/race_test.go", "a run's record/verdict pair"))
	require.NotEmpty(t, retiredModelMentions(".plans/UMPIRE_MODULES.md", "the record/replay harness"))
	require.NotEmpty(t, retiredModelMentions("tools/umpire/lower/race_test.go", "read off `record/`"))
	require.NotEmpty(t, retiredModelMentions("tools/umpire/lower/race_test.go", "features/standaloneactivity/withTaskQueue/"))
}

func TestStandaloneActivityRealizationFollowsSystem(t *testing.T) {
	root := filepath.Join(repoRoot, "model", "temporal", "features", "standaloneactivity")
	require.NoFileExists(t, filepath.Join(root, "Realization.scala"))
	require.FileExists(t, filepath.Join(root, "system", "Realization.scala"))
	require.NoFileExists(t, filepath.Join(root, "product", "Realization.scala"))
}

// Each retired per-kind name is found in a Model folder, at any depth, and nowhere else.
func TestRetiredFeatureFilesAreFound(t *testing.T) {
	for _, name := range retiredFeatureFileNames {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			files := []string{
				"model/temporal/features/nexuscaller/NexusCaller.scala",
				"model/temporal/features/nexuscaller/system/" + name,
				"model/temporal/capabilities/" + name,
				"model/irgen/testdata/lifts/" + name,
			}
			for _, rel := range files {
				require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte("package x"), 0o644))
			}
			found, err := retiredFeatureFiles(root)
			require.NoError(t, err)
			require.Equal(t, []string{"model/temporal/features/nexuscaller/system/" + name}, found)
		})
	}
	found, err := retiredFeatureFiles(t.TempDir())
	require.NoError(t, err)
	require.Empty(t, found, "a checkout without the Model folders has nothing retired")
}

// A build cache under model/, the current one or a stale one of the old name, is read by no check:
// it holds copies of fixtures and generated sources as they were when it was built.
func TestModelFilesLeaveOutTheBuildCaches(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"model/temporal/A.scala":                             "package temporal",
		"model/build/history/x/A.scala":                      "import temporal.worker.*",
		"model/gen/history/lifter.1/A.scala":                 "import temporal.standaloneactivity.*",
		"model/gen/model-scala.classpath":                    "/repo/model/gen/model-scala.jar",
		"model/temporal/features/gen/Generated.scala":        "package gen",
		"model/temporal/features/build/.scala-build/x.scala": "x",
	}
	for rel, content := range files {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644))
	}
	var visited []string
	require.NoError(t, modelFilesUnder(root, func(rel, _ string) { visited = append(visited, rel) }))
	slices.Sort(visited)
	require.Equal(t, []string{"model/temporal/A.scala", "model/temporal/features/gen/Generated.scala"}, visited,
		"only model/'s own build caches are left out, not a folder of the same name deeper down")
}
