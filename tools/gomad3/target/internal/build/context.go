package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type Context struct {
	Directory string
	Package   string
	Tags      []string
}

func Resolve(workingDirectory, source string, suppliedTags []string) (Context, error) {
	tags, err := NormalizeTags(suppliedTags)
	if err != nil {
		return Context{}, err
	}
	if !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
		return Context{Directory: workingDirectory, Package: source, Tags: tags}, nil
	}
	packagePath := source
	if !filepath.IsAbs(packagePath) {
		packagePath = filepath.Join(workingDirectory, packagePath)
	}
	packagePath, err = filepath.Abs(packagePath)
	if err != nil {
		return Context{}, fmt.Errorf("resolve Go target package: %w", err)
	}
	info, err := os.Stat(packagePath)
	if err != nil {
		return Context{}, fmt.Errorf("stat Go target package: %w", err)
	}
	if !info.IsDir() {
		return Context{}, fmt.Errorf("Go target package %s is not a directory", source)
	}
	for directory := packagePath; ; directory = filepath.Dir(directory) {
		moduleFile := filepath.Join(directory, "go.mod")
		if moduleInfo, statErr := os.Stat(moduleFile); statErr == nil && moduleInfo.Mode().IsRegular() {
			relative, relErr := filepath.Rel(directory, packagePath)
			if relErr != nil {
				return Context{}, fmt.Errorf("resolve Go target within module: %w", relErr)
			}
			if relative == "." {
				return Context{Directory: directory, Package: ".", Tags: tags}, nil
			}
			return Context{Directory: directory, Package: "./" + filepath.ToSlash(relative), Tags: tags}, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return Context{}, fmt.Errorf("Go target package %s has no owning go.mod", source)
}

func NormalizeTags(supplied []string) ([]string, error) {
	set := make(map[string]struct{}, len(supplied))
	for _, tag := range supplied {
		if tag == "" || tag == "race" || strings.Contains(tag, ",") || strings.IndexFunc(tag, unicode.IsSpace) >= 0 {
			return nil, fmt.Errorf("unsupported build tag %q", tag)
		}
		set[tag] = struct{}{}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func Environment() []string {
	reserved := map[string]struct{}{
		"CGO_ENABLED": {}, "GOMADSEED": {}, "GOMAD3_CHILD_SEED": {},
		"GOCACHE": {}, "GOENV": {}, "GOEXPERIMENT": {}, "GOFLAGS": {}, "GOROOT": {}, "GOTOOLCHAIN": {}, "GOWORK": {}, "TZ": {},
	}
	environment := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, found := reserved[name]; !found {
			environment = append(environment, entry)
		}
	}
	return append(environment, "CGO_ENABLED=0", "GOENV=off", "GOEXPERIMENT=", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "TZ=UTC")
}

func PrepareCache(toolchainRoot, buildKey string) (string, error) {
	cache := filepath.Join(toolchainRoot, "builds", buildKey, "target-cache")
	info, err := os.Lstat(cache)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(cache, 0o700); err != nil {
			return "", fmt.Errorf("create target build cache: %w", err)
		}
		return cache, nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect target build cache: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("target build cache is not a directory")
	}
	if err := os.Chmod(cache, 0o700); err != nil {
		return "", fmt.Errorf("make target build cache private: %w", err)
	}
	return cache, nil
}
