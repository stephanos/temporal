package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/recipe"
	"gopkg.in/yaml.v3"
)

const (
	configSchema   = "agentworkflow.config/v1"
	resolvedSchema = "agentworkflow.resolved-config/v1"
	maxConfigBytes = 1 << 20
)

type Profile struct {
	Schema         string            `yaml:"schema"`
	Source         Source            `yaml:"source"`
	Instructions   []string          `yaml:"instructions"`
	Checks         []ProfileCheck    `yaml:"checks"`
	Environment    Environment       `yaml:"environment"`
	ForbiddenPaths []string          `yaml:"forbidden_paths,omitempty"`
	Policy         ProfilePolicy     `yaml:"policy"`
	Workflow       recipe.Workflow   `yaml:"workflow"`
	Targets        map[string]Target `yaml:"targets"`
}

type Source struct {
	Mode    string   `yaml:"mode"`
	Exclude []string `yaml:"exclude,omitempty"`
}

type ProfileCheck struct {
	Name      string   `yaml:"name"`
	Command   []string `yaml:"command"`
	Directory string   `yaml:"directory,omitempty"`
	Timeout   Duration `yaml:"timeout,omitempty"`
	Required  bool     `yaml:"required"`
	Enabled   *bool    `yaml:"enabled"`
}

type Environment struct {
	Allow []string `yaml:"allow,omitempty"`
}

type ProfilePolicy struct {
	Assurance        string   `yaml:"assurance,omitempty"`
	MaxRepairs       int      `yaml:"max_repairs,omitempty"`
	Reviewers        []string `yaml:"reviewers,omitempty"`
	BlockingSeverity string   `yaml:"blocking_severity,omitempty"`
}

type Target struct {
	Instructions   []string       `yaml:"instructions,omitempty"`
	Checks         []ProfileCheck `yaml:"checks,omitempty"`
	ForbiddenPaths []string       `yaml:"forbidden_paths,omitempty"`
}

type Resolved struct {
	Root           string
	Source         Source
	Instructions   []string
	Checks         []ProfileCheck
	Environment    Environment
	ForbiddenPaths []string
	Policy         ProfilePolicy
	Workflow       recipe.Workflow
	Target         string
}

type Duration struct {
	time.Duration
}

func (duration *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return errors.New("duration must be a string such as 10m")
	}
	if node.Value == "" {
		return errors.New("duration cannot be empty; omit it to use the default")
	}
	parsed, err := time.ParseDuration(node.Value)
	if err != nil || parsed < 0 {
		return errors.Join(errors.New("duration is invalid"), err)
	}
	duration.Duration = parsed
	return nil
}

func (duration Duration) MarshalYAML() (any, error) {
	if duration.Duration == 0 {
		return "", nil
	}
	return duration.String(), nil
}

func Load(path, root, target string) (Resolved, error) {
	root, err := projectRoot(root)
	if err != nil {
		return Resolved{}, err
	}
	path, err = configPath(path, root)
	if err != nil {
		return Resolved{}, err
	}
	data, err := readConfig(path)
	if err != nil {
		legacy := filepath.Join(root, ".agentworkflow", "project.json")
		if errors.Is(err, os.ErrNotExist) && fileExists(legacy) {
			return Resolved{}, fmt.Errorf("legacy JSON configuration %s found; run agentworkflow init to create %s, merge your settings, then remove the legacy file", legacy, filepath.Join(root, ".spec", "agentworkflow.yaml"))
		}
		if errors.Is(err, os.ErrNotExist) {
			return Resolved{}, fmt.Errorf("%w; run agentworkflow init --project %s to create .spec/agentworkflow.yaml", err, root)
		}
		return Resolved{}, err
	}
	var profile Profile
	if err := decodeConfig(data, &profile); err != nil {
		return Resolved{}, err
	}
	if profile.Schema != configSchema {
		return Resolved{}, fmt.Errorf("unsupported configuration schema %q", profile.Schema)
	}
	if err := validateProfile(&profile, root); err != nil {
		return Resolved{}, err
	}
	workflow, err := recipe.Normalize(profile.Workflow)
	if err != nil {
		return Resolved{}, err
	}
	resolved := Resolved{
		Root: root, Source: profile.Source,
		Instructions: append([]string(nil), profile.Instructions...), Checks: append([]ProfileCheck(nil), profile.Checks...),
		Environment: profile.Environment, ForbiddenPaths: append([]string(nil), profile.ForbiddenPaths...),
		Policy: profile.Policy, Workflow: workflow, Target: target,
	}
	if target != "" {
		selected, found := profile.Targets[target]
		if !found {
			return Resolved{}, fmt.Errorf("project target %q does not exist", target)
		}
		resolved.Instructions = append(resolved.Instructions, selected.Instructions...)
		resolved.Checks = append(resolved.Checks, selected.Checks...)
		resolved.ForbiddenPaths = append(resolved.ForbiddenPaths, selected.ForbiddenPaths...)
	}
	resolved.Instructions, err = normalizePaths("instruction", resolved.Instructions)
	if err != nil {
		return Resolved{}, err
	}
	for _, instruction := range resolved.Instructions {
		if !fileExists(filepath.Join(root, filepath.FromSlash(instruction))) {
			return Resolved{}, fmt.Errorf("declared instruction file %q does not exist", instruction)
		}
	}
	resolved.ForbiddenPaths, err = normalizePaths("forbidden", resolved.ForbiddenPaths)
	if err != nil {
		return Resolved{}, err
	}
	resolved.ForbiddenPaths = compact(append(resolved.ForbiddenPaths, ".spec", relativeToRoot(root, path)))
	resolved.ForbiddenPaths = compact(append(resolved.ForbiddenPaths, resolved.Instructions...))
	checks := resolved.Checks[:0]
	for _, check := range resolved.Checks {
		if check.Enabled != nil && !*check.Enabled {
			continue
		}
		check.Command = append([]string(nil), check.Command...)
		checks = append(checks, check)
	}
	resolved.Checks = checks
	return resolved, nil
}

func Starter(root string) (Profile, error) {
	root, err := projectRoot(root)
	if err != nil {
		return Profile{}, err
	}
	profile := Profile{
		Schema: configSchema, Source: Source{Mode: "directory-copy", Exclude: []string{".cache", "node_modules", "target"}},
		Environment:    Environment{Allow: []string{"HOME", "LANG", "LC_ALL", "PATH", "TEMP", "TMP", "TMPDIR"}},
		ForbiddenPaths: []string{".env", ".git"},
		Policy:         ProfilePolicy{Assurance: "standard", MaxRepairs: 1, BlockingSeverity: "medium"},
		Workflow:       recipe.Default(), Targets: map[string]Target{},
	}
	for _, instruction := range []string{"AGENTS.md", "CLAUDE.md"} {
		if fileExists(filepath.Join(root, instruction)) {
			profile.Instructions = append(profile.Instructions, instruction)
		}
	}
	addCheck := func(name string, command []string, timeout time.Duration, enabled bool) {
		enabledCopy := enabled
		profile.Checks = append(profile.Checks, ProfileCheck{
			Name: name, Command: command, Directory: ".", Timeout: Duration{Duration: timeout}, Required: true, Enabled: &enabledCopy,
		})
	}
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		addCheck("test", []string{"go", "test", "./..."}, 15*time.Minute, true)
	case fileExists(filepath.Join(root, "Cargo.toml")):
		addCheck("test", []string{"cargo", "test"}, 15*time.Minute, true)
	case fileExists(filepath.Join(root, "pyproject.toml")):
		addCheck("test", []string{"python", "-m", "pytest"}, 15*time.Minute, false)
	case fileExists(filepath.Join(root, "package.json")):
		addCheck("test", []string{"npm", "test"}, 15*time.Minute, false)
	default:
	}
	return profile, nil
}

func WriteStarter(root, path string) (string, error) {
	profile, err := Starter(root)
	if err != nil {
		return "", err
	}
	root, err = projectRoot(root)
	if err != nil {
		return "", err
	}
	path, err = configPath(path, root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create configuration directory: %w", err)
	}
	data, err := marshalYAML(profile)
	if err != nil {
		return "", fmt.Errorf("encode starter configuration: %w", err)
	}
	if err := publishExclusive(path, data); err != nil {
		return "", fmt.Errorf("publish starter configuration: %w", err)
	}
	return path, nil
}

func publishExclusive(path string, data []byte) (returnedErr error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".agentworkflow-config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnedErr = errors.Join(returnedErr, err)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(data); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, path); err != nil {
		return err
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return err
	}
	return errors.Join(directoryHandle.Sync(), directoryHandle.Close())
}

func Explain(resolved Resolved) ([]byte, error) {
	return marshalYAML(struct {
		Schema         string          `yaml:"schema"`
		Root           string          `yaml:"root"`
		Target         string          `yaml:"target,omitempty"`
		Source         Source          `yaml:"source"`
		Instructions   []string        `yaml:"instructions"`
		Checks         []ProfileCheck  `yaml:"checks"`
		Environment    Environment     `yaml:"environment"`
		ForbiddenPaths []string        `yaml:"forbidden_paths"`
		Policy         ProfilePolicy   `yaml:"policy"`
		Workflow       recipe.Workflow `yaml:"workflow"`
	}{
		Schema: resolvedSchema, Root: resolved.Root, Target: resolved.Target, Source: resolved.Source,
		Instructions: resolved.Instructions, Checks: resolved.Checks, Environment: resolved.Environment,
		ForbiddenPaths: resolved.ForbiddenPaths, Policy: resolved.Policy, Workflow: resolved.Workflow,
	})
}

func marshalYAML(value any) ([]byte, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	useLiteralPrompts(&node)
	return yaml.Marshal(&node)
}

func useLiteralPrompts(node *yaml.Node) {
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			value := node.Content[index+1]
			if (key.Value == "prompt" || key.Value == "review_prompt" || key.Value == "revision_prompt") && value.Kind == yaml.ScalarNode {
				value.Style = yaml.LiteralStyle
			}
			useLiteralPrompts(value)
		}
		return
	}
	for _, child := range node.Content {
		useLiteralPrompts(child)
	}
}

func decodeConfig(data []byte, target *Profile) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return errors.New("configuration is empty")
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return errors.New("JSON configuration is unsupported; use YAML")
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode YAML configuration: %w", err)
	}
	if err := validateYAMLNode(&document, "configuration"); err != nil {
		return err
	}
	if err := requireExplicitWorkflow(&document); err != nil {
		return err
	}
	if err := decoder.Decode(new(yaml.Node)); err != io.EOF {
		return errors.New("configuration must contain exactly one YAML document")
	}
	decoder = yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode YAML configuration: %w", err)
	}
	return nil
}

func requireExplicitWorkflow(document *yaml.Node) error {
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return errors.New("configuration must be a YAML mapping")
	}
	workflow := mappingValue(document.Content[0], "workflow")
	if workflow == nil || workflow.Kind != yaml.MappingNode {
		return errors.New("configuration.workflow is required and must be a mapping")
	}
	stages := mappingValue(workflow, "stages")
	if stages == nil || stages.Kind != yaml.SequenceNode {
		return errors.New("configuration.workflow.stages is required and must be a sequence")
	}
	for index, stage := range stages.Content {
		if stage.Kind != yaml.MappingNode {
			return fmt.Errorf("configuration.workflow.stages[%d] must be a mapping", index)
		}
		if mappingValue(stage, "enabled") == nil {
			return fmt.Errorf("configuration.workflow.stages[%d].enabled is required", index)
		}
	}
	return nil
}

func mappingValue(mapping *yaml.Node, name string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == name {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func validateYAMLNode(node *yaml.Node, path string) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return fmt.Errorf("%s at line %d, column %d uses an alias or anchor", path, node.Line, node.Column)
	}
	if node.Style&yaml.TaggedStyle != 0 {
		return fmt.Errorf("%s at line %d, column %d uses an explicit tag", path, node.Line, node.Column)
	}
	if node.Tag == "!!null" {
		return fmt.Errorf("%s at line %d, column %d cannot be null", path, node.Line, node.Column)
	}
	if node.Kind == yaml.MappingNode {
		return validateYAMLMapping(node, path)
	}
	for index, child := range node.Content {
		childPath := path
		if node.Kind == yaml.SequenceNode {
			childPath = fmt.Sprintf("%s[%d]", path, index)
		}
		if err := validateYAMLNode(child, childPath); err != nil {
			return err
		}
	}
	return nil
}

func validateYAMLMapping(node *yaml.Node, path string) error {
	seen := make(map[string]struct{}, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		value := node.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return fmt.Errorf("%s at line %d, column %d has a non-string mapping key", path, key.Line, key.Column)
		}
		if key.Value == "<<" || key.Tag == "!!merge" {
			return fmt.Errorf("%s at line %d, column %d uses a merge key", path, key.Line, key.Column)
		}
		if _, found := seen[key.Value]; found {
			return fmt.Errorf("%s.%s at line %d, column %d is duplicated", path, key.Value, key.Line, key.Column)
		}
		seen[key.Value] = struct{}{}
		if err := validateYAMLNode(value, path+"."+key.Value); err != nil {
			return err
		}
	}
	return nil
}

func readConfig(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open project configuration: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if len(data) > maxConfigBytes {
		return nil, errors.New("project configuration exceeds 1 MiB")
	}
	return data, nil
}

func projectRoot(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil || root == string(filepath.Separator) {
		return "", errors.Join(errors.New("configuration project root is invalid"), err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.Join(errors.New("configuration project root is not a directory"), err)
	}
	return root, nil
}

func validateProfile(profile *Profile, root string) error {
	if err := validateSourceAndPaths(profile, root); err != nil {
		return err
	}
	if err := validateEnvironmentAndPolicy(profile); err != nil {
		return err
	}
	return validateTargets(profile, root)
}

func validateSourceAndPaths(profile *Profile, root string) error {
	if profile.Source.Mode != "directory-copy" {
		return fmt.Errorf("source mode %q is unsupported", profile.Source.Mode)
	}
	exclusions, err := normalizePaths("source exclusion", profile.Source.Exclude)
	if err != nil {
		return err
	}
	for _, exclusion := range exclusions {
		if exclusion == "." || exclusion == ".spec" || strings.HasPrefix(exclusion, ".spec/") {
			return errors.New("source.exclude cannot exclude .spec or its contents")
		}
	}
	profile.Source.Exclude = exclusions
	instructions, err := normalizePaths("instruction", profile.Instructions)
	if err != nil {
		return err
	}
	if err := validateInstructions(root, instructions); err != nil {
		return err
	}
	profile.Instructions = instructions
	forbidden, err := normalizePaths("forbidden", profile.ForbiddenPaths)
	if err != nil {
		return err
	}
	profile.ForbiddenPaths = forbidden
	return validateChecks(profile.Checks)
}

func validateEnvironmentAndPolicy(profile *Profile) error {
	for _, name := range profile.Environment.Allow {
		if strings.TrimSpace(name) == "" || strings.ContainsRune(name, '=') {
			return fmt.Errorf("environment name %q is invalid", name)
		}
	}
	if profile.Policy.MaxRepairs < 0 {
		return errors.New("policy.max_repairs cannot be negative")
	}
	switch profile.Policy.Assurance {
	case "fast", "standard", "high":
	default:
		return fmt.Errorf("policy.assurance %q is invalid", profile.Policy.Assurance)
	}
	switch profile.Policy.BlockingSeverity {
	case "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("policy.blocking_severity %q is invalid", profile.Policy.BlockingSeverity)
	}
	for _, reviewer := range profile.Policy.Reviewers {
		if !validComponent(reviewer) {
			return fmt.Errorf("policy reviewer %q is invalid", reviewer)
		}
	}
	return nil
}

func validateTargets(profile *Profile, root string) error {
	for name, target := range profile.Targets {
		if !validComponent(name) {
			return fmt.Errorf("target name %q is invalid", name)
		}
		instructions, err := normalizePaths("target instruction", target.Instructions)
		if err != nil {
			return err
		}
		if err := validateInstructions(root, instructions); err != nil {
			return err
		}
		forbidden, err := normalizePaths("target forbidden", target.ForbiddenPaths)
		if err != nil {
			return err
		}
		target.Instructions = instructions
		target.ForbiddenPaths = forbidden
		if err := validateChecks(append(append([]ProfileCheck(nil), profile.Checks...), target.Checks...)); err != nil {
			return fmt.Errorf("target %q: %w", name, err)
		}
		profile.Targets[name] = target
	}
	return nil
}

func validateInstructions(root string, instructions []string) error {
	for _, instruction := range instructions {
		if err := validateFilePath(root, filepath.Join(root, filepath.FromSlash(instruction)), false); err != nil {
			return fmt.Errorf("declared instruction file %q: %w", instruction, err)
		}
	}
	return nil
}

func validateFilePath(root, path string, allowMissing bool) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("path must stay inside the project root")
	}
	current := root
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) && allowMissing {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %q is a symlink", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("path component %q is not a directory", current)
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return fmt.Errorf("path %q is not a regular file", current)
		}
	}
	return nil
}

func validateChecks(checks []ProfileCheck) error {
	if len(checks) > 32 {
		return errors.New("configuration cannot declare more than 32 checks")
	}
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if !validComponent(check.Name) || len(check.Command) == 0 || strings.TrimSpace(check.Command[0]) == "" {
			return errors.New("each check requires a valid name and command")
		}
		if _, found := seen[check.Name]; found {
			return fmt.Errorf("check %q is duplicated", check.Name)
		}
		seen[check.Name] = struct{}{}
		if check.Directory != "" {
			if _, err := normalizePaths("check directory", []string{check.Directory}); err != nil {
				return err
			}
		}
	}
	return nil
}

func validComponent(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizePaths(kind string, values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
		if value == "" || filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") {
			return nil, fmt.Errorf("%s path %q must stay inside the project root", kind, value)
		}
		result = append(result, value)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func configPath(path, root string) (string, error) {
	if path == "" {
		path = filepath.Join(root, ".spec", "agentworkflow.yaml")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("configuration file must be inside the project root")
	}
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".yaml" && extension != ".yml" {
		return "", errors.New("configuration file must use a .yaml or .yml extension")
	}
	if err := validateFilePath(root, path, true); err != nil {
		return "", fmt.Errorf("configuration file: %w", err)
	}
	return path, nil
}

func relativeToRoot(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

func compact(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, filepath.ToSlash(filepath.Clean(value)))
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func fileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}
