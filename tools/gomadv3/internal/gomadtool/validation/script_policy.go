package validation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/simulation/parity"
)

type scriptOwner struct {
	role         string
	allowBash    bool
	requiredText string
	maxLines     int
}

var reviewedScripts = map[string]scriptOwner{
	"internal/gomadtool/conformance/scripts/clock_audit_test.sh":   {role: "Darwin DTrace platform adapter", allowBash: true, requiredText: "dtrace", maxLines: 100},
	"internal/gomadtool/conformance/scripts/compiler_test_exec.sh": {role: "upstream toolexec adapter", requiredText: "GOMADV3_TEST_COMPILE", maxLines: 40},
	"internal/gomadtool/conformance/scripts/exec.sh":               {role: "upstream exec adapter", requiredText: "GOMADV3_CHILD_SEED", maxLines: 25},
}

func Validate(root string) error {
	if _, err := parity.Current(); err != nil {
		return fmt.Errorf("validate simulation parity contract: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil || absolute == string(filepath.Separator) {
		return errors.Join(errors.New("gomadv3 script-policy root must be an absolute non-root directory"), err)
	}
	var paths []string
	err = filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != absolute && (entry.Name() == ".git" || entry.Name() == ".toolchain") {
				return filepath.SkipDir
			}
			return nil
		}
		extension := filepath.Ext(entry.Name())
		if extension != ".sh" && extension != ".bash" && extension != ".pl" {
			return nil
		}
		relative, err := filepath.Rel(absolute, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk gomadv3 scripts: %w", err)
	}
	slices.Sort(paths)
	for _, path := range paths {
		owner, found := reviewedScripts[path]
		if !found {
			return fmt.Errorf("gomadv3 has an unowned script: %s", path)
		}
		contents, err := os.ReadFile(filepath.Join(absolute, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("read reviewed script %s: %w", path, err)
		}
		source := string(contents)
		if strings.Contains(source, "perl ") || strings.Contains(source, "/perl") || strings.HasSuffix(path, ".pl") {
			return fmt.Errorf("reviewed script %s must not invoke Perl", path)
		}
		if !owner.allowBash {
			if !strings.HasPrefix(source, "#!/bin/sh\n") || strings.Contains(source, "#!/usr/bin/env bash") || strings.Contains(source, "[[") || strings.Contains(source, "]]") || strings.Contains(source, "BASH_") || strings.Contains(source, "pipefail") {
				return fmt.Errorf("reviewed script %s must be strict POSIX shell", path)
			}
		} else if !strings.HasPrefix(source, "#!/usr/bin/env bash\n") {
			return fmt.Errorf("reviewed platform adapter %s must declare Bash explicitly", path)
		}
		if !strings.Contains(source, owner.requiredText) {
			return fmt.Errorf("reviewed script %s no longer implements its %s role", path, owner.role)
		}
		if lines := strings.Count(source, "\n"); lines > owner.maxLines {
			return fmt.Errorf("reviewed script %s grew beyond its %s adapter budget: %d lines", path, owner.role, lines)
		}
	}
	for path := range reviewedScripts {
		if _, err := os.Stat(filepath.Join(absolute, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("reviewed script %s is missing: %w", path, err)
		}
	}
	return nil
}
