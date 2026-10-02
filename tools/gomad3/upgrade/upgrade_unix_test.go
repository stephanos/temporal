//go:build unix

package upgrade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

const limitedPublicationRoot = "GOMAD3_UPGRADE_LIMITED_PUBLICATION_ROOT"

func TestRunKeepsPriorDossierWhenWriteFails(t *testing.T) {
	if root := os.Getenv(limitedPublicationRoot); root != "" {
		runLimitedPublication(t, root)
		return
	}
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, "evidence", "upgrade-dossier.json")
	prior := []byte("{\"prior\":\"complete dossier\"}\n")
	if err := os.Mkdir(filepath.Dir(output), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	// The file-size limit is process-wide and would also fail the test log of
	// this process, so a child process publishes under it.
	executed, err := hostexec.Run(context.Background(), hostexec.Request{
		Command: []string{os.Args[0], "-test.run=^TestRunKeepsPriorDossierWhenWriteFails$"}, Dir: root,
		Env:     append(os.Environ(), limitedPublicationRoot+"="+root),
		Timeout: time.Minute, TerminateGrace: time.Second, OutputLimit: 1 << 20,
	})
	if err != nil || executed.Termination != hostexec.TerminationExit || executed.ExitCode != 0 {
		t.Fatalf("limited publication = %#v, error = %v\n%s%s", executed.Termination, err, executed.Stdout.Bytes, executed.Stderr.Bytes)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(prior) {
		t.Fatalf("prior dossier = %q, want %q", contents, prior)
	}
	if names := directoryNames(t, filepath.Dir(output)); fmt.Sprint(names) != fmt.Sprint([]string{"upgrade-dossier.json"}) {
		t.Fatalf("output directory entries = %v", names)
	}
}

func runLimitedPublication(t *testing.T, root string) {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Fatal(err)
	}
	// Exceeding the limit raises SIGXFSZ, whose default action ends the process
	// before the write can return EFBIG.
	signal.Ignore(syscall.SIGXFSZ)
	// A limit below the dossier size lets the replacement accept a prefix and
	// then fail, which is the partial output that must not be published.
	limit.Cur = 16
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Spec{Root: root, Output: filepath.Join(root, "evidence", "upgrade-dossier.json")})
	if !errors.Is(err, syscall.EFBIG) {
		t.Fatalf("Run() error = %v, want %v", err, syscall.EFBIG)
	}
}
