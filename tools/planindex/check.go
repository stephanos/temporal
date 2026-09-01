package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var markdownLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]*)\)`)
var markdownReferencePattern = regexp.MustCompile(`(?m)^[ \t]{0,3}\[[^]\n]+\]:[ \t]*(\S+)`)
var flowIDPattern = regexp.MustCompile(`^fn-[0-9]+-[a-z0-9]+(?:-[a-z0-9]+)*$`)

type repositoryChecker struct {
	root      string
	findings  []string
	documents map[string]documentEntry
	flowSpecs map[string]flowSpecEntry
}

type trackedFlowSpec struct {
	ID                     string   `json:"id"`
	Status                 string   `json:"status"`
	Ready                  *bool    `json:"ready"`
	CompletionReviewStatus *string  `json:"completion_review_status"`
	Dependencies           []string `json:"depends_on_epics"`
}

func checkRepository(repositoryRoot string, index planIndex) []string {
	root, err := resolveRoot(repositoryRoot)
	if err != nil {
		return []string{fmt.Sprintf("repository root: %v", err)}
	}
	checker := repositoryChecker{
		root: root, documents: make(map[string]documentEntry), flowSpecs: make(map[string]flowSpecEntry),
	}
	checker.checkDocuments(index.Documents)
	checker.checkFlowSpecs(index.FlowSpecs)
	slices.Sort(checker.findings)
	checker.findings = slices.Compact(checker.findings)
	if checker.findings == nil {
		return []string{}
	}
	return checker.findings
}

func resolveRoot(repositoryRoot string) (string, error) {
	absolute, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return resolved, nil
}

func (c *repositoryChecker) checkDocuments(entries []documentEntry) {
	if !sortedBy(entries, func(entry documentEntry) string { return entry.Path }) {
		c.add("documents: entries must be sorted by path")
	}
	counts := make(map[string]int)
	for _, entry := range entries {
		counts[entry.Path]++
		if counts[entry.Path] == 1 {
			c.documents[entry.Path] = entry
		}
	}
	for documentPath, count := range counts {
		if count > 1 {
			c.add("document %s: registered more than once", documentPath)
		}
	}

	discovered, err := discoverFiles(c.root, ".plans", ".md")
	if err != nil {
		c.add("documents: %v", err)
	} else {
		for _, file := range discovered {
			if _, ok := c.documents[file]; !ok {
				c.add("document %s: not registered", file)
			}
		}
	}

	paths := sortedKeys(c.documents)
	for _, documentPath := range paths {
		c.checkDocument(c.documents[documentPath])
	}
	c.checkAuthorityRoots()
	c.checkAuthorityCycles()
}

func (c *repositoryChecker) checkDocument(entry documentEntry) {
	if !validDocumentPath(entry.Path) {
		c.add("document %s: path must be a normalized repository-relative .plans/*.md path", entry.Path)
	} else {
		resolved, err := resolveRepositoryFile(c.root, entry.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.add("document %s: registered file does not exist", entry.Path)
			} else {
				c.add("document %s: %v", entry.Path, err)
			}
		} else {
			c.checkMarkdown(entry, resolved)
		}
	}

	if !sortedUnique(entry.AuthorityParents, func(value string) string { return value }) {
		c.add("document %s: authorityParents must be sorted and unique", entry.Path)
	}
	c.checkDocumentGraph(entry)
	if !sortedUnique(entry.AllowedMissingLinks, missingLinkSortKey) {
		c.add("document %s: allowedMissingLinks must be sorted and unique", entry.Path)
	}
}

func (c *repositoryChecker) checkDocumentGraph(entry documentEntry) {
	for _, parent := range entry.AuthorityParents {
		if !validDocumentPath(parent) {
			c.add("document %s: authority parent %q is not a normalized registered document path", entry.Path, parent)
			continue
		}
		if parent == entry.Path {
			c.add("document %s: authority parent must not reference itself", entry.Path)
			continue
		}
		if _, ok := c.documents[parent]; !ok {
			c.add("document %s: authority parent %q is not registered", entry.Path, parent)
		}
	}
	if entry.Lifecycle == "unclassified" {
		c.add("document %s: lifecycle unclassified is not permitted in a checked registry", entry.Path)
	}
	if entry.Authority == "unclassified" {
		c.add("document %s: authority unclassified is not permitted in a checked registry", entry.Path)
	}
	if entry.Lifecycle == "superseded" {
		if entry.SupersededBy == nil {
			c.add("document %s: superseded lifecycle requires supersededBy", entry.Path)
		}
	} else if entry.SupersededBy != nil {
		c.add("document %s: supersededBy must be null unless lifecycle is superseded", entry.Path)
	}
	if entry.SupersededBy != nil {
		if !validDocumentPath(*entry.SupersededBy) {
			c.add("document %s: supersededBy target %q is not a normalized registered document path", entry.Path, *entry.SupersededBy)
		} else if _, ok := c.documents[*entry.SupersededBy]; !ok {
			c.add("document %s: supersededBy target %q is not registered", entry.Path, *entry.SupersededBy)
		}
	}
}

func (c *repositoryChecker) checkAuthorityRoots() {
	var normative int
	var delivery int
	for _, entry := range c.documents {
		switch entry.Authority {
		case "normative-rules":
			normative++
		case "delivery-order":
			delivery++
		default:
		}
	}
	if normative != 1 {
		c.add("authority graph: expected exactly one normative-rules document; found %d", normative)
	}
	if delivery != 1 {
		c.add("authority graph: expected exactly one delivery-order document; found %d", delivery)
	}
}

func (c *repositoryChecker) checkAuthorityCycles() {
	state := make(map[string]uint8)
	stack := make([]string, 0, len(c.documents))
	var visit func(string)
	visit = func(documentPath string) {
		state[documentPath] = 1
		stack = append(stack, documentPath)
		entry := c.documents[documentPath]
		for _, parent := range entry.AuthorityParents {
			if parent == documentPath {
				continue
			}
			if _, ok := c.documents[parent]; !ok {
				continue
			}
			switch state[parent] {
			case 0:
				visit(parent)
			case 1:
				start := slices.Index(stack, parent)
				cycle := append(slices.Clone(stack[start:]), parent)
				c.add("authority graph: cycle %s", strings.Join(cycle, " -> "))
			default:
			}
		}
		stack = stack[:len(stack)-1]
		state[documentPath] = 2
	}
	for _, documentPath := range sortedKeys(c.documents) {
		if state[documentPath] == 0 {
			visit(documentPath)
		}
	}
}

func (c *repositoryChecker) checkMarkdown(entry documentEntry, resolvedPath string) {
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		c.add("document %s: read file: %v", entry.Path, err)
		return
	}
	usedAllowances := make(map[string]bool)
	for _, destination := range markdownDestinations(string(content)) {
		if destination == "" || externalDestination(destination) {
			continue
		}
		c.checkMarkdownLink(entry, destination, usedAllowances)
	}
	c.checkMissingLinkAllowances(entry, usedAllowances)
}

func (c *repositoryChecker) checkMarkdownLink(entry documentEntry, destination string, usedAllowances map[string]bool) {
	target, anchor, err := resolveMarkdownDestination(entry.Path, destination)
	if err != nil {
		c.add("document %s: local link %q is not repository-confined", entry.Path, destination)
		return
	}
	resolved, err := resolveRepositoryFile(c.root, target)
	if err != nil {
		c.checkUnresolvedMarkdownLink(entry, target, anchor, err, usedAllowances)
		return
	}
	if anchor == "" || filepath.Ext(target) != ".md" {
		return
	}
	anchors, err := markdownAnchors(resolved)
	if err != nil {
		c.add("document %s: read linked document %s: %v", entry.Path, target, err)
		return
	}
	if !anchors[anchor] {
		c.add("document %s: anchor %q is missing from %s", entry.Path, anchor, target)
	}
}

func (c *repositoryChecker) checkUnresolvedMarkdownLink(
	entry documentEntry,
	target string,
	anchor string,
	err error,
	usedAllowances map[string]bool,
) {
	if errors.Is(err, os.ErrNotExist) {
		if validatePotentialRepositoryPath(c.root, target) != nil {
			c.add("document %s: local link %q is not repository-confined", entry.Path, displayLink(target, anchor))
			return
		}
		if allowedMissing(entry.AllowedMissingLinks, target, anchor) {
			usedAllowances[missingLinkIdentity(target, anchor)] = true
			return
		}
		c.add("document %s: local link %q is missing and not allowlisted", entry.Path, displayLink(target, anchor))
		return
	}
	if isConfinementError(err) {
		c.add("document %s: local link %q is not repository-confined", entry.Path, displayLink(target, anchor))
		return
	}
	c.add("document %s: local link %q: %v", entry.Path, displayLink(target, anchor), err)
}

func (c *repositoryChecker) checkMissingLinkAllowances(entry documentEntry, usedAllowances map[string]bool) {
	for _, allowance := range entry.AllowedMissingLinks {
		anchor := ""
		if allowance.Anchor != nil {
			anchor = *allowance.Anchor
		}
		if !validRepositoryRelativePath(allowance.Target) {
			c.add("document %s: allowed missing target %q is not repository-confined", entry.Path, allowance.Target)
			continue
		}
		if validatePotentialRepositoryPath(c.root, allowance.Target) != nil {
			c.add("document %s: allowed missing target %q is not repository-confined", entry.Path, allowance.Target)
			continue
		}
		if !usedAllowances[missingLinkIdentity(allowance.Target, anchor)] {
			c.add("document %s: allowed missing link %q is not used", entry.Path, displayLink(allowance.Target, anchor))
		}
	}
}

func (c *repositoryChecker) checkFlowSpecs(entries []flowSpecEntry) {
	if !sortedBy(entries, func(entry flowSpecEntry) string { return entry.ID }) {
		c.add("flowSpecs: entries must be sorted by id")
	}
	counts := make(map[string]int)
	for _, entry := range entries {
		counts[entry.ID]++
		if counts[entry.ID] == 1 {
			c.flowSpecs[entry.ID] = entry
		}
	}
	for id, count := range counts {
		if count > 1 {
			c.add("flow spec %s: registered more than once", id)
		}
	}
	discovered, err := discoverFiles(c.root, ".flow/specs", ".json")
	if err != nil {
		c.add("flow specs: %v", err)
	} else {
		for _, file := range discovered {
			id := strings.TrimSuffix(path.Base(file), ".json")
			if _, ok := c.flowSpecs[id]; !ok {
				c.add("flow spec %s: not registered", id)
			}
		}
	}
	for _, id := range sortedKeys(c.flowSpecs) {
		c.checkFlowSpec(c.flowSpecs[id])
	}
}

func (c *repositoryChecker) checkFlowSpec(entry flowSpecEntry) {
	if !validFlowID(entry.ID) {
		c.add("flow spec %s: id is not canonical", entry.ID)
		return
	}
	if !sortedUnique(entry.SpecDependencies, func(value string) string { return value }) {
		c.add("flow spec %s: specDependencies must be sorted and unique", entry.ID)
	}
	for _, dependency := range entry.SpecDependencies {
		if !validFlowID(dependency) {
			c.add("flow spec %s: dependency %q is not a canonical Flow spec ID", entry.ID, dependency)
			continue
		}
		if _, ok := c.flowSpecs[dependency]; !ok {
			c.add("flow spec %s: dependency %q is not registered", entry.ID, dependency)
		}
	}
	c.checkFlowCrossFields(entry)

	relativePath := path.Join(".flow/specs", entry.ID+".json")
	resolved, err := resolveRepositoryFile(c.root, relativePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.add("flow spec %s: registered file does not exist", entry.ID)
		} else {
			c.add("flow spec %s: %v", entry.ID, err)
		}
		return
	}
	encoded, err := os.ReadFile(resolved)
	if err != nil {
		c.add("flow spec %s: read %s: %v", entry.ID, relativePath, err)
		return
	}
	var tracked trackedFlowSpec
	if err := json.Unmarshal(encoded, &tracked); err != nil {
		message := err.Error()
		if strings.Contains(message, "unexpected end of JSON input") {
			message = "unexpected end of JSON input"
		}
		c.add("flow spec %s: decode %s: %s", entry.ID, relativePath, message)
		return
	}
	if tracked.ID != entry.ID {
		c.add("flow spec %s: Flow body id is %q", entry.ID, tracked.ID)
	}
	if tracked.Status != entry.Status {
		c.add("flow spec %s: status %s; Flow records %s", entry.ID, entry.Status, normalizedText(tracked.Status, "missing"))
	}
	ready := false
	if tracked.Ready != nil {
		ready = *tracked.Ready
	}
	if ready != entry.Ready {
		c.add("flow spec %s: ready is %t; Flow records %t", entry.ID, entry.Ready, ready)
	}
	review := "unknown"
	if tracked.CompletionReviewStatus != nil {
		review = *tracked.CompletionReviewStatus
	}
	if review != entry.CompletionReview {
		c.add("flow spec %s: completionReview %s; Flow records %s", entry.ID, entry.CompletionReview, review)
	}
	actualDependencies := slices.Clone(tracked.Dependencies)
	if actualDependencies == nil {
		actualDependencies = []string{}
	}
	slices.Sort(actualDependencies)
	if !slices.Equal(entry.SpecDependencies, actualDependencies) {
		c.add("flow spec %s: specDependencies %v; Flow records %v", entry.ID, entry.SpecDependencies, actualDependencies)
	}
}

func (c *repositoryChecker) checkFlowCrossFields(entry flowSpecEntry) {
	if entry.Disposition == "unclassified" {
		c.add("flow spec %s: disposition unclassified is not permitted in a checked registry", entry.ID)
	}
	switch entry.Scope {
	case "other":
		if entry.Disposition != "out-of-scope" || entry.Phase != "none" {
			c.add("flow spec %s: scope other requires disposition out-of-scope and phase none", entry.ID)
		}
	case "umpire-roadmap":
		if entry.Disposition == "out-of-scope" || !contains([]string{"p0", "p1", "p2", "p3", "verification"}, entry.Phase) {
			c.add("flow spec %s: scope umpire-roadmap requires a roadmap phase and non-out-of-scope disposition", entry.ID)
		}
	case "umpire-support":
		if entry.Disposition == "out-of-scope" || entry.Phase != "support" {
			c.add("flow spec %s: scope umpire-support requires phase support and non-out-of-scope disposition", entry.ID)
		}
	default:
	}
	switch entry.Disposition {
	case "retained":
		if entry.Status != "open" {
			c.add("flow spec %s: retained disposition requires status open", entry.ID)
		}
	case "deferred", "superseded":
		if entry.Status != "open" || entry.Ready || entry.CompletionReview == "ship" {
			c.add("flow spec %s: %s disposition requires status open, ready false, and completionReview other than ship", entry.ID, entry.Disposition)
		}
	case "completed-prerequisite":
		validOpen := entry.Status == "open" && !entry.Ready && entry.CompletionReview == "ship"
		if entry.Status != "done" && !validOpen {
			c.add("flow spec %s: completed-prerequisite requires status done or open with ready false and completionReview ship", entry.ID)
		}
	default:
	}
}

func discoverFiles(root, relativeDirectory, extension string) ([]string, error) {
	directory := filepath.Join(root, filepath.FromSlash(relativeDirectory))
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relativeDirectory, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != extension {
			continue
		}
		files = append(files, path.Join(relativeDirectory, entry.Name()))
	}
	slices.Sort(files)
	return files, nil
}

func validDocumentPath(value string) bool {
	return validRepositoryRelativePath(value) && path.Dir(value) == ".plans" && path.Ext(value) == ".md"
}

func validRepositoryRelativePath(value string) bool {
	if value == "" || strings.Contains(value, "\\") || path.IsAbs(value) || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned != "." && cleaned == value && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func resolveRepositoryFile(root, relativePath string) (string, error) {
	if !validRepositoryRelativePath(relativePath) {
		return "", errors.New("path is not repository-confined")
	}
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, resolved) {
		return "", errors.New("path resolves outside repository root")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	linkInfo, err := os.Lstat(target)
	if err != nil {
		return "", err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("path is a symlink alias")
	}
	if hasSymlinkComponent(root, relativePath) {
		return "", errors.New("path uses a symlink alias")
	}
	return resolved, nil
}

func validatePotentialRepositoryPath(root, relativePath string) error {
	if !validRepositoryRelativePath(relativePath) {
		return errors.New("path is not repository-confined")
	}
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	for {
		_, err := os.Lstat(target)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(target)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, target)
			if err != nil || !pathWithin(root, resolved) || hasSymlinkComponent(root, filepath.ToSlash(relative)) {
				return errors.New("path is not repository-confined")
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(target)
		if parent == target || !pathWithin(root, parent) {
			return errors.New("path is not repository-confined")
		}
		target = parent
	}
}

func hasSymlinkComponent(root, relativePath string) bool {
	current := root
	for _, component := range strings.Split(filepath.ToSlash(relativePath), "/") {
		if component == "" {
			continue
		}
		current = filepath.Join(current, filepath.FromSlash(component))
		info, err := os.Lstat(current)
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func isConfinementError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "outside repository root") || strings.Contains(message, "symlink alias") || strings.Contains(message, "not repository-confined")
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func validFlowID(id string) bool {
	return flowIDPattern.MatchString(id)
}

func markdownDestination(raw string) string {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "<") {
		if end := strings.Index(value, ">"); end >= 0 {
			return value[1:end]
		}
	}
	if separator := strings.IndexAny(value, " \t\n"); separator >= 0 {
		value = value[:separator]
	}
	return value
}

func markdownDestinations(content string) []string {
	var destinations []string
	for _, pattern := range []*regexp.Regexp{markdownLinkPattern, markdownReferencePattern} {
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			destinations = append(destinations, markdownDestination(match[1]))
		}
	}
	return destinations
}

func externalDestination(destination string) bool {
	parsed, err := url.Parse(destination)
	return err == nil && (parsed.Scheme != "" || parsed.Host != "")
}

func resolveMarkdownDestination(sourcePath, destination string) (target string, anchor string, err error) {
	parsed, err := url.Parse(destination)
	if err != nil {
		return "", "", err
	}
	decodedPath, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", "", err
	}
	anchor, err = url.PathUnescape(parsed.Fragment)
	if err != nil {
		return "", "", err
	}
	target = sourcePath
	if decodedPath != "" {
		if path.IsAbs(decodedPath) || filepath.IsAbs(decodedPath) {
			return "", "", errors.New("destination is absolute")
		}
		target = path.Join(path.Dir(sourcePath), decodedPath)
	}
	if !validRepositoryRelativePath(target) {
		return "", "", errors.New("destination escapes repository")
	}
	return target, anchor, nil
}

func markdownAnchors(markdownPath string) (map[string]bool, error) {
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		return nil, err
	}
	anchors := make(map[string]bool)
	counts := make(map[string]int)
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimLeft(trimmed, "#")
		if heading == trimmed || (heading != "" && !unicode.IsSpace(rune(heading[0]))) {
			continue
		}
		heading = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(heading), "#"))
		base := markdownSlug(heading)
		if base == "" {
			continue
		}
		anchor := base
		if counts[base] > 0 {
			anchor = fmt.Sprintf("%s-%d", base, counts[base])
		}
		counts[base]++
		anchors[anchor] = true
	}
	return anchors, nil
}

func markdownSlug(heading string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(character), unicode.IsNumber(character), character == '-', character == '_':
			result.WriteRune(character)
		case unicode.IsSpace(character):
			result.WriteByte('-')
		default:
		}
	}
	return result.String()
}

func allowedMissing(allowances []allowedMissingLink, target, anchor string) bool {
	for _, allowance := range allowances {
		allowedAnchor := ""
		if allowance.Anchor != nil {
			allowedAnchor = *allowance.Anchor
		}
		if allowance.Target == target && allowedAnchor == anchor {
			return true
		}
	}
	return false
}

func displayLink(target, anchor string) string {
	if anchor == "" {
		return target
	}
	return target + "#" + anchor
}

func missingLinkIdentity(target, anchor string) string {
	return target + "\x00" + anchor
}

func missingLinkSortKey(link allowedMissingLink) string {
	anchor := ""
	if link.Anchor != nil {
		anchor = *link.Anchor
	}
	return missingLinkIdentity(link.Target, anchor) + "\x00" + link.Reason
}

func sortedBy[T any](values []T, key func(T) string) bool {
	for index := 1; index < len(values); index++ {
		if key(values[index-1]) >= key(values[index]) {
			return false
		}
	}
	return true
}

func sortedUnique[T any](values []T, key func(T) string) bool {
	return sortedBy(values, key)
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (c *repositoryChecker) add(format string, args ...any) {
	c.findings = append(c.findings, fmt.Sprintf(format, args...))
}

func contains(values []string, wanted string) bool {
	return slices.Contains(values, wanted)
}

func normalizedText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
