package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Inventory struct {
	Files        []string      `json:"files"`
	Manifests    []string      `json:"manifests"`
	Instructions []Instruction `json:"instructions,omitempty"`
}

type Instruction struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func Discover(ctx context.Context, root string, instructions []string, maxFiles int, maxBytes int64) (Inventory, error) {
	if maxFiles <= 0 || maxBytes <= 0 {
		return Inventory{}, errors.New("project discovery bounds must be positive")
	}
	root, err := filepath.Abs(root)
	if err != nil || root == string(filepath.Separator) {
		return Inventory{}, errors.Join(errors.New("project discovery root is invalid"), err)
	}
	seenInstructions := make(map[string]struct{}, len(instructions))
	for _, path := range instructions {
		seenInstructions[filepath.ToSlash(filepath.Clean(path))] = struct{}{}
	}
	discovery := discoverer{
		ctx: ctx, root: root, pendingInstructions: seenInstructions,
		maxFiles: maxFiles, maxBytes: maxBytes,
	}
	err = filepath.WalkDir(root, discovery.visit)
	if err != nil {
		return Inventory{}, err
	}
	for path := range discovery.pendingInstructions {
		return Inventory{}, fmt.Errorf("declared instruction file %q does not exist", path)
	}
	result := discovery.result
	slices.Sort(result.Files)
	slices.Sort(result.Manifests)
	slices.SortFunc(result.Instructions, func(left, right Instruction) int { return strings.Compare(left.Path, right.Path) })
	return result, nil
}

type discoverer struct {
	ctx                 context.Context
	root                string
	pendingInstructions map[string]struct{}
	maxFiles            int
	maxBytes            int64
	files               int
	retainedBytes       int64
	result              Inventory
}

func (discovery *discoverer) visit(path string, item fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := discovery.ctx.Err(); err != nil {
		return err
	}
	relative, err := relativePath(discovery.root, path)
	if err != nil || relative == "." {
		return err
	}
	if item.IsDir() {
		if relative == ".git" || relative == ".agentworkflow" {
			return filepath.SkipDir
		}
		return nil
	}
	discovery.files++
	if discovery.files > discovery.maxFiles {
		return errors.New("project discovery file limit exceeded")
	}
	discovery.result.Files = append(discovery.result.Files, relative)
	_, explicit := discovery.pendingInstructions[relative]
	if explicit || recognizedManifest(filepath.Base(relative)) {
		discovery.result.Manifests = append(discovery.result.Manifests, relative)
	}
	if !explicit {
		return nil
	}
	content, err := readBounded(path, discovery.maxBytes-discovery.retainedBytes)
	if err != nil {
		return fmt.Errorf("read instruction %q: %w", relative, err)
	}
	discovery.retainedBytes += int64(len(content))
	discovery.result.Instructions = append(discovery.result.Instructions, Instruction{Path: relative, Content: string(content)})
	delete(discovery.pendingInstructions, relative)
	return nil
}

func relativePath(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.Join(errors.New("project discovery path escaped root"), err)
	}
	return filepath.ToSlash(relative), nil
}

func recognizedManifest(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "readme") || strings.HasSuffix(lower, ".csproj") {
		return true
	}
	switch lower {
	case "agents.md", "claude.md", "go.mod", "go.work", "package.json", "pyproject.toml", "requirements.txt",
		"cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts", "makefile", "justfile", "workspace", "flake.nix":
		return true
	default:
		return false
	}
}

func readBounded(path string, remaining int64) ([]byte, error) {
	if remaining <= 0 {
		return nil, errors.New("project discovery byte limit exceeded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, remaining+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(data)) > remaining {
		return nil, errors.New("project discovery byte limit exceeded")
	}
	return data, nil
}
