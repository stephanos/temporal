package backend

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/scanner"

	"go.temporal.io/server/tools/gomad3/hostfs"
)

func assemblyIncludes(file, packageDirectory, sourceDirectory, includeDirectory string, files, visited map[string]bool) error {
	if visited[file] {
		return nil
	}
	visited[file] = true
	data, err := hostfs.ReadBounded(file, 512<<20)
	if err != nil {
		return err
	}
	var lexer scanner.Scanner
	lexer.Init(bytes.NewReader(data))
	lexer.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanChars | scanner.ScanRawStrings | scanner.ScanComments | scanner.SkipComments
	var lexicalError error
	lexer.Error = func(_ *scanner.Scanner, message string) {
		lexicalError = fmt.Errorf("assembler source %s: %s", file, message)
	}
	for token := lexer.Scan(); token != scanner.EOF; token = lexer.Scan() {
		if token != '#' || lexer.Scan() != scanner.Ident || lexer.TokenText() != "include" {
			continue
		}
		if lexer.Scan() != scanner.String {
			return fmt.Errorf("unsupported assembler include in %s", file)
		}
		name, err := strconv.Unquote(lexer.TokenText())
		if err != nil {
			return err
		}
		candidates := []string{name}
		if !filepath.IsAbs(name) {
			candidates = []string{filepath.Join(packageDirectory, name), filepath.Join(sourceDirectory, name)}
			if name != "go_asm.h" {
				candidates = append(candidates, filepath.Join(includeDirectory, name))
			}
		}
		selected := ""
		for _, path := range candidates {
			info, err := os.Stat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("assembler include is not a regular file: %s", path)
			}
			selected = path
			break
		}
		// cmd/go derives this header using the pinned compiler only when a
		// package-local or source-local header did not shadow it first.
		if selected == "" && name == "go_asm.h" {
			continue
		}
		if selected == "" {
			return fmt.Errorf("assembler include cannot be captured: %s", name)
		}
		files[selected] = true
		if err := assemblyIncludes(selected, packageDirectory, sourceDirectory, includeDirectory, files, visited); err != nil {
			return err
		}
	}
	return lexicalError
}
