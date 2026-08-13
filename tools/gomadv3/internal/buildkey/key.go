package buildkey

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Input struct {
	GoVersion        string
	ArchiveSHA256    string
	PatchSHA256      string
	OverlaySHA256    string
	HostOS           string
	HostArch         string
	BootstrapVersion string
	RecipeVersion    string
	BuildPath        string
	BashPath         string
	BashVersion      string
}

func Compute(input Input) (string, error) {
	values := []string{
		input.GoVersion, input.ArchiveSHA256, input.PatchSHA256, input.OverlaySHA256,
		input.HostOS, input.HostArch, input.BootstrapVersion, input.RecipeVersion,
		input.BuildPath, input.BashPath, input.BashVersion,
	}
	for index, value := range values {
		if value == "" || strings.ContainsRune(value, '\n') {
			return "", fmt.Errorf("build identity field %d is empty or contains a newline", index)
		}
	}
	hash := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(hash, value)
		_, _ = io.WriteString(hash, "\n")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func FileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func TreeDigest(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("overlay entry is not a regular file: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("overlay tree contains no files")
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		digest, err := FileDigest(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hash, relative)
		_, _ = hash.Write([]byte{0})
		_, _ = io.WriteString(hash, digest)
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
