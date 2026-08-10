package gomadmain

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/temporalio/gomad/internal/gomadtool"
	"github.com/temporalio/gomad/internal/translate"
)

//go:embed doc.go
var rawDoc string

func doc() string {
	doc := strings.TrimPrefix(rawDoc, "/*\n")
	doc = doc[:strings.LastIndex(doc, "*/")]
	return doc
}

func commandName(cmd string) string {
	return fmt.Sprintf("%s %s", path.Base(os.Args[0]), cmd)
}

type buildOptions struct {
	race bool
	tags string
}

func (o *buildOptions) register(flags *flag.FlagSet) {
	flags.BoolVar(&o.race, "race", false, "build in -race mode")
	flags.StringVar(&o.tags, "tags", "", "comma-separated list of additional build tags")
}

func (o buildOptions) apply(cfg *gomadtool.BuildConfig) {
	cfg.Race = o.race
	cfg.UserTags = gomadtool.ParseBuildTags(o.tags)
}

func translatedBuildFlags(cfg gomadtool.BuildConfig) []string {
	return []string{"-ldflags=-checklinkname=0", "-tags=" + cfg.BuildTags()}
}

func configureGoBuildCache(cmd string) error {
	switch cmd {
	case "translate", "test", "build-tests", "debug", "prepare-selftest":
	default:
		return nil
	}
	if _, ok := os.LookupEnv("GOCACHE"); ok {
		return nil
	}

	modDir, err := gomadtool.FindGoModDir()
	if err != nil {
		return fmt.Errorf("find module root for Go build cache: %w", err)
	}
	if err := os.Setenv("GOCACHE", filepath.Join(modDir, gomadtool.OutputDirectory, "go-build")); err != nil {
		return fmt.Errorf("set GOCACHE: %w", err)
	}
	return nil
}

func packageName(pkgPath string) string {
	return pkgPath[strings.LastIndex(pkgPath, "/")+1:]
}

func batchPackagesWithDifferentNames(packages []string) [][]string {
	var groups [][]string
	count := make(map[string]int)

	for _, pkg := range packages {
		lastName := packageName(pkg)
		group := count[lastName]
		count[lastName]++
		if group >= len(groups) {
			groups = append(groups, nil)
		}
		groups[group] = append(groups[group], pkg)
	}

	return groups
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func copyFile(from, to string, perm fs.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func copyFileIfChanged(src, dst string, perm fs.FileMode) error {
	// skip unchanged so we keep the old modified timestamp
	dstHash, err := hashFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if dstHash != nil {
		srcHash, err := hashFile(src)
		if err != nil {
			return err
		}
		if bytes.Equal(srcHash, dstHash) {
			// skip because unchanged, maybe log?
			return nil
		}
	}

	if err := copyFile(src, dst, perm); err != nil {
		return err
	}

	return nil
}

func writeFileIfDifferent(b []byte, dst string, perm fs.FileMode) error {
	// skip unchanged so we keep the old modified timestamp
	dstBytes, err := os.ReadFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if bytes.Equal(b, dstBytes) {
		// skip because unchanged, maybe log?
		return nil
	}

	if err := os.WriteFile(dst, b, perm); err != nil {
		return err
	}

	return nil
}

const toolsgoTemplate = `//go:build never

package main

import (
	_ "` + gomadtool.Module + `"
)
`

func Main() {
	flag.Usage = func() {
		fmt.Print(doc())
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := flag.Args()[0]
	cmdArgs := flag.Args()[1:]

	if err := configureGoBuildCache(cmd); err != nil {
		log.Fatal(err)
	}

	cfg := gomadtool.BuildConfig{
		GOOS:   "linux",
		GOARCH: runtime.GOARCH,
		Race:   false,
	}

	switch cmd {
	case "translate":
		translateflags := flag.NewFlagSet(commandName("translate"), flag.ExitOnError)
		var build buildOptions
		build.register(translateflags)
		translateflags.Parse(cmdArgs)

		build.apply(&cfg)

		_, err := translate.Translate(&translate.TranslateInput{
			Packages: translateflags.Args(),
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

	case "test":
		testflags := flag.NewFlagSet(commandName("test"), flag.ExitOnError)
		verbose := testflags.Bool("v", false, "verbose output")
		var build buildOptions
		build.register(testflags)
		run := testflags.String("run", "", "tests to run (as in go test -run)")
		logformat := testflags.String("logformat", "pretty", "gomad log formatting: raw|indented|pretty")
		simtrace := testflags.String("simtrace", "", "set of a comma-separated traces to enable")
		seeds := testflags.String("seeds", "1", "a comma separated list of seeds and ranges to run, such as 1,2,10-100,99")
		testflags.Parse(cmdArgs)

		build.apply(&cfg)

		packages := testflags.Args()
		if len(packages) == 0 {
			packages = []string{"."}
		}

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: packages,
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		name := "go"
		args := []string{"test"}

		// Translated standard-library packages retain runtime linknames.
		args = append(args, translatedBuildFlags(cfg)...)
		if *verbose {
			args = append(args, "-v")
		}
		if build.race {
			args = append(args, "-race")
		}
		args = append(args, "-trimpath")
		if *run != "" {
			args = append(args, "-run", *run)
		}
		args = append(args, output.Packages...)
		if *logformat != "pretty" {
			// TODO: configure this default somewhere?
			args = append(args, "-logformat", *logformat)
		}
		if *simtrace != "" {
			args = append(args, "-simtrace", *simtrace)
		}
		if *seeds != "" {
			args = append(args, "-seeds", *seeds)
		}

		cmd := exec.Command(name, args...)
		cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
		cmd.Dir = output.RootOutputDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			log.Fatal(err)
		}

	case "build-tests":
		// do we supply packages or does it just work?
		// for now, let's supply packages...

		modDir, err := gomadtool.FindGoModDir()
		if err != nil {
			log.Fatal(err)
		}

		testflags := flag.NewFlagSet(commandName("build-tests"), flag.ExitOnError)
		var build buildOptions
		build.register(testflags)
		testflags.Parse(cmdArgs)

		build.apply(&cfg)

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: testflags.Args(),
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		// tests are named after the last part of the directory, even if they
		// have a different internal name (nice.)

		groups := batchPackagesWithDifferentNames(output.Packages)

		// log.Println(groups)

		dstDir := filepath.Join(modDir, gomadtool.OutputDirectory, "metatest", cfg.AsDirname())
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			log.Fatal(err)
		}

		buildDir, err := os.MkdirTemp("", "gomadbuild")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(buildDir)

		for _, group := range groups {
			// copy over existing binaries to make it faster
			for _, pkg := range group {
				binName := packageName(pkg) + ".test"
				src := filepath.Join(dstDir, gomadtool.PreparedTestBinName(pkg))
				dst := filepath.Join(buildDir, binName)
				if err := copyFile(src, dst, 0o755); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						// ignore missing source files, this is just an optimization
						continue
					}
					log.Fatal(err)
				}
			}

			// Translated standard-library packages retain runtime linknames.
			name := "go"
			args := []string{"test"}
			args = append(args, translatedBuildFlags(cfg)...)
			if build.race {
				args = append(args, "-race")
			}
			args = append(args, "-trimpath")

			args = append(args, "-c")
			args = append(args, "-o", buildDir)

			args = append(args, group...)

			cmd := exec.Command(name, args...)
			cmd.Dir = output.RootOutputDir

			// TODO: consider if we want to do this?
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				log.Fatal(err)
			}

			for _, pkg := range group {
				binName := packageName(pkg) + ".test"
				src := filepath.Join(buildDir, binName)
				dst := filepath.Join(dstDir, gomadtool.PreparedTestBinName(pkg))

				// skip unchanged so we keep the old modified timestamp
				if err := copyFileIfChanged(src, dst, 0o755); err != nil {
					log.Fatal(err)
				}

				depBytes, err := json.MarshalIndent(output.Deps[pkg], "", "  ") // log.Println(binName, pkg, output.Deps[pkg])
				if err != nil {
					log.Fatal(err)
				}

				if err := writeFileIfDifferent(depBytes, filepath.Join(dstDir, gomadtool.PreparedTestInfoName(pkg)), 0o644); err != nil {
					log.Fatal(err)
				}
			}
		}

	case "debug":
		// TODO: make -headless flag write launch configuration?
		// TODO: for -headless, make ctrl-c work?
		// TODO: for -headless, use delve api to send initial continue?

		debugflags := flag.NewFlagSet(commandName("debug"), flag.ExitOnError)
		var build buildOptions
		build.register(debugflags)
		pkg := debugflags.String("package", "", "package path to debug")
		test := debugflags.String("test", "", "full test name to debug")
		seed := debugflags.Int("seed", 1, "seed to debug")
		step := debugflags.Int("step", 0, "step to break at")
		headless := debugflags.Bool("headless", false, "run headless for IDE debugging")
		debugflags.Parse(cmdArgs)

		modDir, err := gomadtool.FindGoModDir()
		if err != nil {
			log.Fatal(err)
		}

		build.apply(&cfg)

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: []string{*pkg},
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		if len(output.Packages) != 1 {
			log.Fatalf("expected 1 output packages, got %v", output.Packages)
		}
		translated := output.Packages[0]

		script := fmt.Sprintf(`continue
stepout`)
		scriptPath := filepath.Join(modDir, gomadtool.OutputDirectory, "debug-script")
		if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
			log.Fatal(err)
		}

		name := "dlv"
		flags := []string{
			"test",
			translated,
			"--build-flags=" + strings.Join(translatedBuildFlags(cfg), " "),
			// TODO: does this actually set linkname?
			// TODO: pass on -race
		}

		if *headless {
			flags = append(flags,
				"--listen=:2345",
				"--accept-multiclient",
				"--headless",
			)
			log.Printf("running delve in headless mode, connect with vscode using a launch configuration (.vscode/launch.json) like:\n" +
				`{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to dlv-dap server on localhost:2345",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}",
            "port": 2345,
            "host": "127.0.0.1",
            "debugAdapter": "dlv-dap",
            "stopOnEntry": true,
        }
    ]
}`)
		} else {
			flags = append(flags,
				"--init="+scriptPath,
			)
			log.Println("running delve in the terminal...")
		}

		flags = append(flags,
			"--",
			"-test.run",
			"^"+*test+"$",
			"-test.v",
			"-step-breakpoint="+fmt.Sprint(*step),
			"-seeds="+fmt.Sprint(*seed),
		)

		cmd := exec.Command(name, flags...)
		cmd.Dir = path.Join(modDir, gomadtool.OutputDirectory, "translated", cfg.AsDirname())
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		log.Println(cmd.Args)
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			log.Fatal(err)
		}

	case "prepare-selftest":
		prepareSelftest()

	default:
		flag.Usage()
		os.Exit(2)
	}
}

// TODO: test with and without gomad dependency? (script test?)
