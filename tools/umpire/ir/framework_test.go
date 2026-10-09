package ir

// model/framework is the framework the models are written in, and it describes no one system: the
// words of the system a model checks belong beside that model, in model/temporal. The framework's
// prose and identifiers use neutral examples instead, so a reader learns the framework on its own
// terms and a new kind of model needs nothing from it that names another.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const frameworkRoot = modelRoot + "/framework"

// TestFrameworkLayout keeps the generic framework's source path and Scala namespace distinct from
// the Umpire product and its tooling namespaces. Production IR and Cases are intentionally omitted:
// the batch seal regenerates those once all structural renames have landed.
func TestFrameworkLayout(t *testing.T) {
	require.DirExists(t, filepath.Join(repoRoot, frameworkRoot))
	require.NoDirExists(t, filepath.Join(repoRoot, modelRoot, "umpire"))

	var mentions []string
	require.NoError(t, modelFiles(func(rel, content string) {
		if strings.HasPrefix(rel, modelRoot+"/ir/") || strings.HasPrefix(rel, modelRoot+"/cases/") {
			return
		}
		for i, line := range strings.Split(content, "\n") {
			if retiredFrameworkMention(line) {
				mentions = append(mentions, fmt.Sprintf("%s:%d", rel, i+1))
			}
		}
	}))
	require.Empty(t, mentions, "framework sources use model/framework and the framework.* Scala namespace")
}

var preservedUmpireProtobufNamespace = regexp.MustCompile(`(?:io\.)?temporal\.server\.api\.umpire\.v1\b`)

func retiredFrameworkMention(line string) bool {
	allowed := strings.NewReplacer(
		"umpire.irgen", "",
		"umpire.check", "",
		"umpire.case.service", "",
	).Replace(preservedUmpireProtobufNamespace.ReplaceAllString(line, ""))
	return strings.Contains(allowed, "model/umpire") || strings.Contains(allowed, "umpire.") ||
		strings.TrimSpace(allowed) == "package umpire"
}

func TestFrameworkNamespaceDistinguishesPreservedProtobufFromRetiredDSL(t *testing.T) {
	for _, line := range []string{"temporal.server.api.umpire.v1.Model", "io.temporal.server.api.umpire.v1.Model"} {
		require.False(t, retiredFrameworkMention(line), line)
	}
	for _, line := range []string{"import umpire.*", "umpire.Finite", "package umpire", "model/umpire", "temporal.server.api.umpire.v2.Model", "temporal.server.api.umpire.v10.Model", "temporal.server.api.umpire.v1other.Model", "temporal.server.api.umpire.v1.Model; umpire.Finite"} {
		require.True(t, retiredFrameworkMention(line), line)
	}
}

// temporalTerms are the words that name a Temporal concept, lowercase and with their plurals: the
// server, its services and entities, and the capability kinds a realization declares. They are
// spelled here, outside model, so that the check itself is no mention.
var temporalTerms = map[string]bool{
	"temporal": true,
	"workflow": true, "workflows": true,
	"activity": true, "activities": true,
	"nexus":     true,
	"namespace": true, "namespaces": true,
	"taskqueue": true, "taskqueues": true,
	"worker": true, "workers": true,
	"history": true, "histories": true,
	"chasm":    true,
	"matching": true,
	"frontend": true, "frontends": true,
	"closable": true, "terminable": true, "pausable": true, "cancelable": true, "pollable": true, "describable": true,
	"retries": true, "deadline": true,
}

// temporalPairs are the terms spelled as two words, by their first word and the words that may
// follow it.
var temporalPairs = map[string][]string{"task": {"queue", "queues"}}

// letterRun is a run of letters and digits; anything else, an underscore too, separates two runs.
var letterRun = regexp.MustCompile(`[A-Za-z0-9]+`)

// frameworkAllowance keeps the mentions in path, a file or, with a trailing slash, a directory under
// the repository's root, for reason.
type frameworkAllowance struct {
	path, reason string
}

// frameworkAllowances are the mentions that stay for now. Each one moves out of the framework with
// the task its reason names, which removes its entry; a generic use that no rewording can serve may
// be added with its reason.
var frameworkAllowances = []frameworkAllowance{}

func (a frameworkAllowance) covers(rel string) bool {
	if strings.HasSuffix(a.path, "/") {
		return strings.HasPrefix(rel, a.path)
	}
	return rel == a.path
}

// identifierWords splits a run of letters and digits at the boundaries an identifier marks: a
// lowercase letter before an uppercase one (taskQueue), an uppercase letter before one that starts a
// capitalized word (RPCWorker), and a change between letters and digits (worker2).
func identifierWords(run string) []string {
	isUpper := func(c byte) bool { return 'A' <= c && c <= 'Z' }
	isLower := func(c byte) bool { return 'a' <= c && c <= 'z' }
	isDigit := func(c byte) bool { return '0' <= c && c <= '9' }
	var words []string
	start := 0
	for i := 1; i < len(run); i++ {
		prev, cur := run[i-1], run[i]
		capitalized := i+1 < len(run) && isUpper(prev) && isUpper(cur) && isLower(run[i+1])
		if isLower(prev) && isUpper(cur) || capitalized || isDigit(prev) != isDigit(cur) {
			words = append(words, run[start:i])
			start = i
		}
	}
	return append(words, run[start:])
}

// namesTemporal reports whether text names a Temporal concept: whether one of its words, after
// splitting identifiers into theirs, is a term, whole and ignoring case, or two adjacent words spell
// a pair. Two words are adjacent within one identifier, or across spaces, underscores and hyphens;
// any other character between them separates the two. So `caseWorker`, `TASK_QUEUE` and `Workers`
// are mentions, and `Coworker`, `workerless` and `task, queue` are not.
func namesTemporal(text string) bool {
	previous, end := "", 0
	for _, at := range letterRun.FindAllStringIndex(text, -1) {
		if strings.Trim(text[end:at[0]], " \t_-") != "" {
			previous = ""
		}
		for _, word := range identifierWords(text[at[0]:at[1]]) {
			word = strings.ToLower(word)
			if temporalTerms[word] {
				return true
			}
			for _, second := range temporalPairs[previous] {
				if word == second {
					return true
				}
			}
			previous = word
		}
		end = at[1]
	}
	return false
}

// temporalMentions returns where the file at path, with this content, names a Temporal concept:
// "path" for its own name, and "path:line" for each line that does.
func temporalMentions(path, content string) []string {
	var found []string
	if namesTemporal(path) {
		found = append(found, path)
	}
	for i, line := range strings.Split(content, "\n") {
		if namesTemporal(line) {
			found = append(found, fmt.Sprintf("%s:%d", path, i+1))
		}
	}
	return found
}

// TestFrameworkNamesNoTemporal keeps model/framework free of Temporal's vocabulary, in prose and in
// identifiers alike, so the framework stays one any system's model can be written in and Temporal
// is described where its models are, in model/temporal. A mention stays only under an allowance with
// its reason, and an allowance that keeps no mention any more fails too, so the list only shrinks.
func TestFrameworkNamesNoTemporal(t *testing.T) {
	scanned := 0
	var mentions []string
	kept := make(map[string]bool)
	require.NoError(t, modelFiles(func(rel, content string) {
		if !strings.HasPrefix(rel, frameworkRoot+"/") {
			return
		}
		scanned++
		found := temporalMentions(rel, content)
		for _, a := range frameworkAllowances {
			if a.covers(rel) {
				kept[a.path] = kept[a.path] || len(found) > 0
				return
			}
		}
		mentions = append(mentions, found...)
	}))
	require.Greater(t, scanned, 10, "the walk reaches the framework's sources")
	require.Empty(t, mentions, "the framework names no Temporal concept: reword these with a neutral example, keeping the rule each explains")

	var stale []string
	for _, a := range frameworkAllowances {
		if _, err := os.Stat(filepath.Join(repoRoot, a.path)); err != nil || !kept[a.path] {
			stale = append(stale, a.path)
		}
	}
	require.Empty(t, stale, "these allowances keep no mention any more: remove them")
}

func TestTemporalTermsAreFound(t *testing.T) {
	const path = "model/framework/Table.scala"
	for name, test := range map[string]struct {
		path, content string
		mentions      []string
	}{
		"the server":            {path, "// as Temporal runs it", []string{path + ":1"}},
		"in lowercase":          {path, "ok\n// the temporal table", []string{path + ":2"}},
		"in uppercase":          {path, "// a CHASM component", []string{path + ":1"}},
		"a workflow":            {path, "// one workflow", []string{path + ":1"}},
		"workflows":             {path, "// two Workflows", []string{path + ":1"}},
		"an activity":           {path, "// the activity", []string{path + ":1"}},
		"activities":            {path, "// the activities", []string{path + ":1"}},
		"nexus":                 {path, "// a Nexus operation", []string{path + ":1"}},
		"a namespace":           {path, "// per namespace", []string{path + ":1"}},
		"namespaces":            {path, "// across namespaces", []string{path + ":1"}},
		"a task queue":          {path, "// on a task queue", []string{path + ":1"}},
		"task queues":           {path, "// two Task  Queues", []string{path + ":1"}},
		"a hyphenated pair":     {path, "// task-queue routing", []string{path + ":1"}},
		"a joined pair":         {path, "// see taskqueue and TaskQueues", []string{path + ":1"}},
		"a worker":              {path, "// the worker polls", []string{path + ":1"}},
		"workers":               {path, "// all Workers", []string{path + ":1"}},
		"a history":             {path, "// its history", []string{path + ":1"}},
		"histories":             {path, "// two histories", []string{path + ":1"}},
		"matching":              {path, "// handed to matching", []string{path + ":1"}},
		"a frontend":            {path, "// the frontend", []string{path + ":1"}},
		"frontends":             {path, "// all frontends", []string{path + ":1"}},
		"a closable entity":     {path, "Closable(status, terminal, rejected)", []string{path + ":1"}},
		"a terminable entity":   {path, "Terminable(", []string{path + ":1"}},
		"a pausable entity":     {path, "Pausable(", []string{path + ":1"}},
		"a cancelable entity":   {path, "Cancelable(", []string{path + ":1"}},
		"a pollable entity":     {path, "Pollable(", []string{path + ":1"}},
		"a describable entity":  {path, "Describable(", []string{path + ":1"}},
		"a retries capability":  {path, "Retries(", []string{path + ":1"}},
		"a camelCase field":     {path, "val caseWorker = 1", []string{path + ":1"}},
		"a camelCase pair":      {path, "s.taskQueue", []string{path + ":1"}},
		"a leading word":        {path, "workflowService.start()", []string{path + ":1"}},
		"a trailing word":       {path, "val sendToNexus = 1", []string{path + ":1"}},
		"an acronym before":     {path, "RPCWorker()", []string{path + ":1"}},
		"a snake_case constant": {path, "METHOD_PAUSE_ACTIVITY_EXECUTION", []string{path + ":1"}},
		"a snake_case pair":     {path, "TASK_QUEUE_KIND", []string{path + ":1"}},
		"a digit suffix":        {path, "val worker2 = 1", []string{path + ":1"}},
		"a qualified name":      {path, "temporal.nexus.caller", []string{path + ":1"}},
		"an endpoint":           {path, "nexusEndpoint", []string{path + ":1"}},
		"every line":            {path, "history\nfine\nworkers", []string{path + ":1", path + ":3"}},
		"a file name":           {"model/framework/WorkflowTable.scala", "package framework", []string{"model/framework/WorkflowTable.scala"}},
		"a directory":           {"model/framework/nexus/Table.scala", "// Nexus", []string{"model/framework/nexus/Table.scala", "model/framework/nexus/Table.scala:1"}},
		"words containing a term": {
			path, "a Coworker; workerless; temporally; historical; Matchings; frontendy; nexuses; reactivity", nil,
		},
		"the words of a pair alone": {path, "a task; a queue; queue tasks; the task, queued; task. Queue it; taskmaster", nil},
		"capability verbs":          {path, "close, terminate, pause, cancel, poll and describe; cancellable", nil},
		"the framework": {
			path, "package framework\nimport framework.realize.Operand.*\n// realize a Machine; cleanup\nval f = Family(\"example.orders\")", nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.mentions, temporalMentions(test.path, test.content))
		})
	}
}
