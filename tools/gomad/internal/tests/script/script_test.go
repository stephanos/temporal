package script_test

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"

	"github.com/temporalio/gomad/internal/gomadtool"
)

func TestScript(t *testing.T) {
	p := testscript.Params{
		Dir: "testdata",
	}
	gotooltest.Setup(&p)

	curModDir, err := gomadtool.FindGoModDir()
	if err != nil {
		t.Fatal(err)
	}
	extractedModDir := filepath.Join(curModDir, gomadtool.OutputDirectory, "scripttest", "mod")

	// add test dependencies to the copied gomad module
	if err := filepath.Walk(extractedModDir, func(path string, info fs.FileInfo, err error) error {
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	origSetup := p.Setup
	p.Setup = func(e *testscript.Env) error {
		// put a fake go.mod in the work directory
		// this include a copy of the gomad module (not the entire module because we want to minimize test dependencies)
		if err := gomadtool.MakeGoModForTest(extractedModDir, e.WorkDir, []string{
			"github.com/dave/dst",
			"github.com/google/go-cmp",
			"github.com/mattn/go-isatty",
			"github.com/mattn/go-sqlite3",
			"golang.org/x/mod",
			"golang.org/x/sync",
			"golang.org/x/sys",
			"golang.org/x/tools",
		}); err != nil {
			return err
		}

		// add /bin to PATH in tests
		binDir := filepath.Join(curModDir, gomadtool.OutputDirectory, "scripttest", "bin")
		// add dependencies on /bin
		if err := filepath.Walk(binDir, func(path string, info fs.FileInfo, err error) error {
			return nil
		}); err != nil {
			log.Fatal(err)
		}
		e.Setenv("PATH", binDir+string(filepath.ListSeparator)+e.Getenv("PATH"))
		e.Setenv("GOMADTOOL", filepath.Join(binDir, "gomad"))

		// allow the module to access our shared cache
		e.Vars = append(e.Vars, "GOMADCACHE="+filepath.Join(curModDir, gomadtool.OutputDirectory))

		if origSetup != nil {
			return origSetup(e)
		}
		return nil
	}
	testscript.Run(t, p)
}

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, nil))
}
