// Package installation owns the on-disk layout of a Gomad toolchain
// installation. The toolchain builder publishes this layout, and target
// preparation and deterministic I/O read their locations from it instead of
// joining installation-relative paths themselves.
package installation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckoutDirectory is the installation directory a source checkout builds
// beside the Gomad module and a checkout's executables are resolved against.
const CheckoutDirectory = ".toolchain"

const repairGuidance = "set --toolchain-root or GOMAD3_TOOLCHAIN_DIR to a complete Gomad installation"

// Layout names the locations of an installation root without checking that
// any of them exist; the builder uses it to publish an installation.
type Layout struct {
	root string
}

// At returns the layout rooted at root, made absolute against the working
// directory.
func At(root string) (Layout, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Layout{}, err
	}
	return Layout{root: root}, nil
}

func (layout Layout) Root() string { return layout.root }

// GoCommand is the stable launcher that executes the published build.
func (layout Layout) GoCommand() string { return filepath.Join(layout.root, "bin", "go") }

func (layout Layout) Bin() string { return filepath.Join(layout.root, "bin") }

// BuildKeyFile holds the key of the published build.
func (layout Layout) BuildKeyFile() string { return filepath.Join(layout.root, "build-key") }

func (layout Layout) Builds() string { return filepath.Join(layout.root, "builds") }

// Build names the immutable build published under key.
func (layout Layout) Build(key string) Build {
	return Build{directory: filepath.Join(layout.root, "builds", key)}
}

func (layout Layout) Locks() string { return filepath.Join(layout.root, "locks") }

func (layout Layout) Lock(key string) string { return filepath.Join(layout.root, "locks", key+".lock") }

func (layout Layout) Downloads() string { return filepath.Join(layout.root, "downloads") }

// Adapters holds deterministic I/O adapter replacements. The go command
// records a directory replacement's path in the target binary's module
// information, so this location is part of every adapted target's identity.
func (layout Layout) Adapters() string { return filepath.Join(layout.root, "adapters") }

// Build names the locations inside one immutable toolchain build.
type Build struct {
	directory string
}

// Directory is the build's GOROOT.
func (build Build) Directory() string { return build.directory }

func (build Build) GoCommand() string { return filepath.Join(build.directory, "bin", "go") }

// TargetCache is the go build cache shared by targets built with this build.
func (build Build) TargetCache() string { return filepath.Join(build.directory, "target-cache") }

// PreparedTargets caches complete prepared targets beside the target cache.
func (build Build) PreparedTargets() string {
	return filepath.Join(build.directory, "prepared-targets")
}

// Description is a validated installation: its launcher is executable, its
// build key is well formed, and the build that key names is present.
type Description struct {
	Layout
	buildKey string
}

// Describe validates the installation at root. Failures that a complete
// installation repairs carry that guidance.
func Describe(root string) (Description, error) {
	if root == "" {
		return Description{}, fmt.Errorf("toolchain root is required")
	}
	layout, err := At(root)
	if err != nil {
		return Description{}, fmt.Errorf("resolve toolchain root: %w", err)
	}
	root = layout.Root()
	info, err := os.Lstat(layout.GoCommand())
	if err != nil {
		return Description{}, fmt.Errorf("stat pinned Go command in %s: %w; %s", root, err, repairGuidance)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return Description{}, fmt.Errorf("pinned Go command is not a regular executable")
	}
	buildKeyBytes, err := os.ReadFile(layout.BuildKeyFile())
	if err != nil {
		return Description{}, fmt.Errorf("read toolchain build key in %s: %w; %s", root, err, repairGuidance)
	}
	buildKey := strings.TrimSuffix(string(buildKeyBytes), "\n")
	if len(buildKey) != sha256.Size*2 || !isLowerHex(buildKey) || string(buildKeyBytes) != buildKey+"\n" {
		return Description{}, fmt.Errorf("toolchain build key is malformed")
	}
	builtGo := layout.Build(buildKey).GoCommand()
	if builtInfo, statErr := os.Stat(builtGo); statErr != nil || !builtInfo.Mode().IsRegular() || builtInfo.Mode()&0o111 == 0 {
		return Description{}, fmt.Errorf("toolchain build %s is missing or stale in %s; %s", buildKey, root, repairGuidance)
	}
	return Description{Layout: layout, buildKey: buildKey}, nil
}

func (description Description) BuildKey() string { return description.buildKey }

// PinnedBuild is the build the installation's build key names.
func (description Description) PinnedBuild() Build { return description.Build(description.buildKey) }

func isLowerHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
