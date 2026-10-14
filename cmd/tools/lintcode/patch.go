package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var patchHunk = regexp.MustCompile(`^@@ -[0-9]+(?:,([0-9]+))? \+[0-9]+(?:,([0-9]+))? @@`)

func lintPatch(ctx context.Context, root, base, module string) ([]byte, error) {
	changed, err := gitPaths(ctx, root, "diff", "--name-only", "-z", base, "--", "*.go")
	if err != nil {
		return nil, err
	}
	for _, path := range changed {
		if strings.ContainsAny(path, "\r\n") {
			return nil, fmt.Errorf("unsupported Git patch path %q", path)
		}
	}
	// Scope only after Git has matched renames against the whole repository.
	data, err := commandOutput(ctx, root, "git", "diff", "--find-renames", "--no-relative", "--no-ext-diff", "--no-textconv", "--color=never", "--src-prefix=a/", "--dst-prefix=b/", "--unified=0", base, "--")
	if err != nil {
		return nil, err
	}
	patch, err := scopePatch(string(data), module)
	if err != nil {
		return nil, err
	}
	untracked, err := gitPaths(ctx, root, "ls-files", "-z", "--others", "--exclude-standard", "--", "*.go")
	if err != nil {
		return nil, err
	}
	for _, path := range untracked {
		if !within(path, module) {
			continue
		}
		name, err := patchPath("b/"+path, module)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			continue
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		patch += fmt.Sprintf("+++ b/%s\n@@ -0,0 +1,%d @@\n", name, len(lines))
		for range lines {
			patch += "+\n"
		}
	}
	return []byte(patch), nil
}

func patchPath(header, module string) (string, error) {
	header = strings.TrimSuffix(header, "\t")
	if header == "/dev/null" {
		return "", nil
	}
	if strings.HasPrefix(header, "\"") {
		var err error
		header, err = strconv.Unquote(header)
		if err != nil {
			return "", fmt.Errorf("invalid Git patch path: %w", err)
		}
	}
	if strings.ContainsAny(header, "\r\n") || !strings.HasPrefix(header, "b/") {
		return "", fmt.Errorf("unsupported Git patch path %q", header)
	}
	path := strings.TrimPrefix(header, "b/")
	if !within(path, module) || !strings.HasSuffix(path, ".go") {
		return "", nil
	}
	return strings.TrimPrefix(path, module+"/"), nil
}

func scopePatch(patch, module string) (string, error) {
	state := patchScope{module: module}
	for _, line := range strings.Split(strings.TrimSuffix(patch, "\n"), "\n") {
		if err := state.consume(line); err != nil {
			return "", err
		}
	}
	if state.old != 0 || state.next != 0 {
		return "", errors.New("truncated Git patch hunk")
	}
	return state.result.String(), nil
}

type patchScope struct {
	module, path string
	hasHeader    bool
	old, next    int
	result       strings.Builder
}

func (p *patchScope) consume(line string) error {
	if line == "\\ No newline at end of file" {
		return nil
	}
	if p.old != 0 || p.next != 0 {
		var err error
		p.old, p.next, err = patchLineCounts(line, p.old, p.next)
		if err != nil {
			return err
		}
		if p.path != "" {
			// Only positions matter to revgrep; source text can mimic a +++ header.
			p.result.WriteString(line[:1] + "\n")
		}
		return nil
	}
	switch {
	case strings.HasPrefix(line, "diff --git "):
		p.path = ""
		p.hasHeader = false
	case strings.HasPrefix(line, "+++ "):
		if p.hasHeader {
			return errors.New("duplicate Git patch file header")
		}
		p.hasHeader = true
		var err error
		p.path, err = patchPath(strings.TrimPrefix(line, "+++ "), p.module)
		if err != nil {
			return err
		}
		if p.path != "" {
			// revgrep consumes raw +++ paths; it does not decode Git quoting.
			p.result.WriteString("+++ b/" + p.path + "\n")
		}
	case strings.HasPrefix(line, "@@"):
		if !p.hasHeader {
			return errors.New("git patch hunk lacks a file header")
		}
		var err error
		p.old, p.next, err = patchHunkCounts(line)
		if err != nil {
			return err
		}
		if p.path != "" {
			p.result.WriteString(line + "\n")
		}
	case patchMetadata(line):
	default:
		return fmt.Errorf("unsupported Git patch line: %q", line)
	}
	return nil
}

func patchMetadata(line string) bool {
	if line == "" || line == "\\ No newline at end of file" {
		return true
	}
	for _, prefix := range []string{"--- ", "index ", "new file mode ", "deleted file mode ", "old mode ", "new mode ", "similarity index ", "dissimilarity index ", "rename from ", "rename to ", "Binary files "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func patchLineCounts(line string, old, next int) (oldRemaining, nextRemaining int, err error) {
	switch {
	case strings.HasPrefix(line, "+"):
		next--
	case strings.HasPrefix(line, "-"):
		old--
	case strings.HasPrefix(line, " "):
		old--
		next--
	default:
		return 0, 0, fmt.Errorf("malformed Git patch hunk: %q", line)
	}
	if old < 0 || next < 0 {
		return 0, 0, errors.New("git patch hunk exceeds declared line counts")
	}
	return old, next, nil
}

func patchHunkCounts(line string) (old, next int, err error) {
	match := patchHunk.FindStringSubmatch(line)
	if match == nil {
		return 0, 0, fmt.Errorf("malformed Git patch header: %q", line)
	}
	for i, count := range []*int{&old, &next} {
		*count = 1
		if match[i+1] != "" {
			*count, err = strconv.Atoi(match[i+1])
			if err != nil {
				return 0, 0, err
			}
		}
	}
	return old, next, nil
}
