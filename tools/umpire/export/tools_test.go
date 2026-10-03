package export

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// The backend agreement is opted into: UMPIRE_BACKENDS=require, as `make umpire-check-backends` sets
// it. Without it every tool run skips and no tool is looked for. With it the tools are the pinned
// ones and no others, none of which the repository's mise.toml or the default `go test` needs:
//
//	Quint 0.33.0   through quint.sh beside this file, which runs that exact version from npm's cache
//	Apalache       0.62.1, which `quint verify` downloads into ~/.quint; a JVM, the repository's
//	P 3.1.0        a .NET tool, in ${UMPIRE_BACKEND_TOOLS:-.build/umpire-backend-tools}/p
//	.NET SDK 8.0   in ${UMPIRE_BACKEND_TOOLS:-.build/umpire-backend-tools}/dotnet
//
// A missing tool, or one of another version, fails the run with the tool named before any test
// starts; nothing is skipped, because a comparison that did not run is no agreement.
const (
	optIn        = "UMPIRE_BACKENDS"
	quintVersion = "0.33.0"
	pVersion     = "3.1.0"
)

func required() bool { return os.Getenv(optIn) == "require" }

// pinned checks the pinned tools and gives the two executables the tests run: the Quint launcher
// and the P installed in tools. look finds an executable, and run gives what a command line prints.
func pinned(launcher, tools string, look func(string) (string, error), run func(string, ...string) (string, error)) (quint, p string, err error) {
	if _, err := look("npm"); err != nil {
		return "", "", fmt.Errorf("npm is not on the path: Quint %s runs from npm's cache", quintVersion)
	}
	if version, err := run(launcher, "--version"); err != nil || strings.TrimSpace(version) != quintVersion {
		return "", "", fmt.Errorf("%s does not run Quint %s", launcher, quintVersion)
	}
	if _, err := run("java", "-version"); err != nil {
		return "", "", errors.New("no JVM: `quint verify` runs Apalache on the repository's JDK (mise.toml); run through make or `mise exec --`")
	}
	p = filepath.Join(tools, "p", "p")
	if _, err := look(p); err != nil {
		return "", "", fmt.Errorf("P is not installed in %s: tools/umpire/export/README.md says how .NET 8 and P %s are installed there", filepath.Dir(p), pVersion)
	}
	// P prints its version with a build number after the pinned release.
	if version, _ := run(p, "--version"); !strings.Contains(version, "P version "+pVersion+".") {
		return "", "", fmt.Errorf("%s is not P %s: move %s aside and install it again", p, pVersion, filepath.Dir(p))
	}
	return launcher, p, nil
}

var receipts struct {
	sync.Mutex
	lines []string
}

// TestMain runs the tests as they are without the opt-in. With it, it pins the tools first, stops
// the Apalache server the checks left running, and prints one receipt per comparison, which
// UMPIRE_BACKENDS_OUT also keeps beside every export, dump and report.
func TestMain(m *testing.M) {
	if !required() {
		m.Run()
		return
	}
	agreement(m)
}

// agreement runs the tests with the pinned tools. It exits by itself on a failure, because a run
// that stopped before the tests has no status of theirs to report.
func agreement(m *testing.M) {
	failed := func(err error) {
		fmt.Fprintln(os.Stderr, "umpire-backends:", err)
		//nolint:revive // deep-exit: without a status of the tests, the test binary would exit 0
		os.Exit(1)
	}
	launcher, err := filepath.Abs("quint.sh")
	if err != nil {
		failed(err)
	}
	// Under .build, which git ignores and a restart keeps.
	tools := os.Getenv("UMPIRE_BACKEND_TOOLS")
	if tools == "" {
		if tools, err = filepath.Abs(filepath.Join("..", "..", "..", ".build", "umpire-backend-tools")); err != nil {
			failed(err)
		}
	}
	// P runs on .NET, which it finds by DOTNET_ROOT or on the path, so its version is asked for in
	// the environment the checks run in.
	dotnet := filepath.Join(tools, "dotnet")
	environment := [][2]string{
		{"DOTNET_ROOT", dotnet},
		{"DOTNET_CLI_TELEMETRY_OPTOUT", "1"},
		{"DOTNET_NOLOGO", "1"},
		{"PATH", dotnet + string(os.PathListSeparator) + os.Getenv("PATH")},
	}
	quint, p, err := pinned(launcher, tools, exec.LookPath, func(name string, arguments ...string) (string, error) {
		command := exec.Command(name, arguments...)
		command.Env = os.Environ()
		for _, variable := range environment {
			command.Env = append(command.Env, variable[0]+"="+variable[1])
		}
		printed, err := command.CombinedOutput()
		return string(printed), err
	})
	if err != nil {
		failed(err)
	}
	for _, variable := range append(environment, [2]string{"UMPIRE_QUINT", quint}, [2]string{"UMPIRE_P", p}) {
		if err := os.Setenv(variable[0], variable[1]); err != nil {
			failed(err)
		}
	}
	out := os.Getenv("UMPIRE_BACKENDS_OUT")
	if out != "" {
		if out, err = filepath.Abs(out); err != nil {
			failed(err)
		}
		if err := os.MkdirAll(out, 0o755); err != nil {
			failed(err)
		}
		if err := os.Setenv("UMPIRE_BACKENDS_OUT", out); err != nil {
			failed(err)
		}
	}

	status := m.Run()
	// The Apalache server `quint verify` starts listens on a port of this gate's own, and is stopped with it.
	StopVerifier()
	slices.Sort(receipts.lines)
	lines := slices.Compact(receipts.lines)
	for _, line := range lines {
		fmt.Println(line)
	}
	if out != "" {
		if err := os.WriteFile(filepath.Join(out, "receipts.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			failed(err)
		}
	}
	if status != 0 {
		failed(errors.New("the backend agreement failed"))
	}
	fmt.Printf("umpire-backends: Quint %s and P %s agree with Go in every comparison that ran; see the receipts above for what was compared and what was not\n", quintVersion, pVersion)
}

// needs gives a backend's tool. Without the opt-in the test skips before any tool is looked for;
// with it a missing tool fails the test: an agreement that was not run is not reported as one.
func needs(t *testing.T, find func() (Tool, bool)) Tool {
	t.Helper()
	if !required() {
		t.Skipf("the backend agreement is not run here: `make umpire-check-backends` runs it (%s=require)", optIn)
	}
	found, ok := find()
	if !ok {
		t.Fatalf("%s is not installed, and %s=require", found.Name, optIn)
	}
	return found
}

// workDir is where a test's tool reads its export and writes its reports: a temporary directory, or,
// under UMPIRE_BACKENDS_OUT, a fresh directory of that one named after the test, which is kept.
func workDir(t *testing.T) string {
	t.Helper()
	out := os.Getenv("UMPIRE_BACKENDS_OUT")
	if out == "" {
		return t.TempDir()
	}
	dir, err := os.MkdirTemp(out, strings.NewReplacer("/", "-", " ", "_", "'", "").Replace(t.Name())+"-*")
	require.NoError(t, err)
	return dir
}

// report logs a receipt on a line of its own, and keeps it for the list the opted-in run prints.
func report(t *testing.T, r Receipt) {
	t.Helper()
	t.Logf("RECEIPT %s", r)
	receipts.Lock()
	defer receipts.Unlock()
	receipts.lines = append(receipts.lines, r.String())
}

func TestNoToolIsLookedForWithoutTheOptIn(t *testing.T) {
	t.Setenv(optIn, "")
	looked := false
	skipped := false
	t.Run("a tool run", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		needs(t, func() (Tool, bool) {
			looked = true
			return Tool{Name: "quint"}, true
		})
	})
	require.True(t, skipped)
	require.False(t, looked)
}

func TestTheOptInFailsOnAMissingToolInsteadOfSkipping(t *testing.T) {
	t.Setenv(optIn, "require")
	found := needs(t, func() (Tool, bool) { return Tool{Name: "quint", Command: "/pinned/quint"}, true })
	require.Equal(t, Tool{Name: "quint", Command: "/pinned/quint"}, found)
}

func TestPinnedToolsAreNamedWhenMissingOrOfAnotherVersion(t *testing.T) {
	const launcher, tools = "/repository/tools/umpire/export/quint.sh", "/tools"
	p := filepath.Join(tools, "p", "p")
	installed := map[string]string{
		launcher + " --version": quintVersion + "\n",
		"java -version":         "openjdk\n",
		p + " --version":        "P version " + pVersion + ".0\n",
	}
	// printed is what each command line prints where it runs at all; change replaces one of them,
	// or removes it when it names no output.
	check := func(present []string, change ...string) (string, string, error) {
		printed := map[string]string{}
		for line, out := range installed {
			printed[line] = out
		}
		switch len(change) {
		case 1:
			delete(printed, change[0])
		case 2:
			printed[change[0]] = change[1]
		default:
		}
		look := func(name string) (string, error) {
			if !slices.Contains(present, name) {
				return "", exec.ErrNotFound
			}
			return name, nil
		}
		run := func(name string, arguments ...string) (string, error) {
			out, ok := printed[strings.Join(append([]string{name}, arguments...), " ")]
			if !ok {
				return "", exec.ErrNotFound
			}
			return out, nil
		}
		return pinned(launcher, tools, look, run)
	}

	quint, found, err := check([]string{"npm", p})
	require.NoError(t, err)
	require.Equal(t, []string{launcher, p}, []string{quint, found})

	for name, test := range map[string]struct {
		present []string
		change  []string
		message string
	}{
		"no npm":                {[]string{p}, nil, "npm is not on the path: Quint 0.33.0 runs from npm's cache"},
		"no Quint":              {[]string{"npm", p}, []string{launcher + " --version"}, launcher + " does not run Quint 0.33.0"},
		"another Quint":         {[]string{"npm", p}, []string{launcher + " --version", "0.34.0\n"}, launcher + " does not run Quint 0.33.0"},
		"no JVM":                {[]string{"npm", p}, []string{"java -version"}, "no JVM: `quint verify` runs Apalache on the repository's JDK (mise.toml); run through make or `mise exec --`"},
		"no P":                  {[]string{"npm"}, nil, "P is not installed in /tools/p: tools/umpire/export/README.md says how .NET 8 and P 3.1.0 are installed there"},
		"another P":             {[]string{"npm", p}, []string{p + " --version", "P version 3.2.0.0\n"}, p + " is not P 3.1.0: move /tools/p aside and install it again"},
		"a P that does not run": {[]string{"npm", p}, []string{p + " --version"}, p + " is not P 3.1.0: move /tools/p aside and install it again"},
	} {
		t.Run(name, func(t *testing.T) {
			quint, found, err := check(test.present, test.change...)
			require.EqualError(t, err, test.message)
			require.Equal(t, []string{"", ""}, []string{quint, found})
		})
	}
}
