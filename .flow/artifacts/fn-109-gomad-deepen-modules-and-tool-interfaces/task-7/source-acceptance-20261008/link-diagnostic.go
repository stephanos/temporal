package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type observation struct {
	Mode      string
	Phase     string
	Path      string
	RootSys   string
	RootNlink uint64
	PathSys   string
	PathNlink uint64
	Remaining []string
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var observations []observation
	for _, mode := range []string{"os.Root.RemoveAll", "os.RemoveAll"} {
		root, err := os.MkdirTemp("", ".fn1097-link-diagnostic-")
		must(err)
		func() {
			defer func() { must(os.RemoveAll(root)) }()
			shared := filepath.Join(root, "targets", "shared")
			must(os.MkdirAll(filepath.Dir(shared), 0o700))
			must(os.WriteFile(shared, []byte("evidence\n"), 0o600))
			for _, name := range []string{"campaign-first", "campaign-second"} {
				artifact := filepath.Join(root, "v1", name, "successes", "sha256-x")
				must(os.MkdirAll(artifact, 0o700))
				must(os.WriteFile(filepath.Join(artifact, "manifest.json"), []byte("evidence\n"), 0o600))
				must(os.Link(shared, filepath.Join(artifact, "target")))
			}
			pinned, err := os.OpenRoot(root)
			must(err)
			defer func() { must(pinned.Close()) }()
			observe := func(phase string) {
				rootInfo, err := pinned.Lstat("targets/shared")
				must(err)
				pathInfo, err := os.Lstat(shared)
				must(err)
				actual := observation{Mode: mode, Phase: phase, Path: shared, RootSys: fmt.Sprintf("%T", rootInfo.Sys()), PathSys: fmt.Sprintf("%T", pathInfo.Sys())}
				if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
					actual.RootNlink = uint64(stat.Nlink)
				}
				if stat, ok := pathInfo.Sys().(*syscall.Stat_t); ok {
					actual.PathNlink = uint64(stat.Nlink)
				}
				must(filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !entry.IsDir() {
						actual.Remaining = append(actual.Remaining, path)
					}
					return nil
				}))
				observations = append(observations, actual)
			}
			observe("three links")
			for _, name := range []string{"campaign-first", "campaign-second"} {
				if mode == "os.Root.RemoveAll" {
					must(pinned.RemoveAll(filepath.Join("v1", name)))
				} else {
					must(os.RemoveAll(filepath.Join(root, "v1", name)))
				}
				observe("removed " + name)
			}
		}()
	}
	must(json.NewEncoder(os.Stdout).Encode(observations))
}
