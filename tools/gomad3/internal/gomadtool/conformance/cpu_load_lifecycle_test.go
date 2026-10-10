//go:build unix

package conformance

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func TestCPULoadWorkerLifecycle(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			result := runCommand(t, []string{os.Args[0], "-test.run=^TestCPULoadWorkerLifecycleHelper$", "-test.count=1"}, []string{"GOMAD3_CPU_LOAD_LIFECYCLE=" + strconv.Itoa(count)})
			if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || result.WatchdogTimeout || result.Cancelled || !result.GroupGone {
				t.Fatalf("load worker lifecycle = %#v", result)
			}
			if result.Stdout.Truncated || result.Stderr.TotalBytes != 0 || string(result.Stdout.Bytes) != "PASS\n" {
				t.Fatalf("load worker output = %q/%q", result.Stdout.Bytes, result.Stderr.Bytes)
			}
		})
	}
}

func TestCPULoadWorkerLifecycleHelper(t *testing.T) {
	value := os.Getenv("GOMAD3_CPU_LOAD_LIFECYCLE")
	if value == "" {
		t.Skip("load worker subprocess only")
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	stop, err := startCPULoadWorkers(count)
	if err != nil {
		t.Fatal(err)
	}
	var callers sync.WaitGroup
	release := make(chan struct{})
	for range 4 {
		callers.Go(func() {
			<-release
			if err := stop(); err != nil {
				t.Error(err)
			}
		})
	}
	close(release)
	callers.Wait()
	for range 2 {
		if err := stop(); err != nil {
			t.Fatal(err)
		}
	}
}
