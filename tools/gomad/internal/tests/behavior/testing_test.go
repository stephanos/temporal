//go:build gomad

package behavior

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/temporalio/gomad"
)

// TODO: t.Fail, t.FailNow, log.Fatal, fmt.Print, slog.Log, os.Exit

func TestTFatalAborts(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	log.Println("before")
	t.Fatal("help")
	log.Println("after")
}

func TestPanicAbortsOther(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	ch := make(chan struct{})
	go func() {
		log.Println("before")
		panic("help")
		log.Println("after")
		close(ch)
	}()

	<-ch
	log.Println("never")
}

func TestPanic(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	log.Println("before")
	panic("help")
	log.Println("never")
}

func TestPanicInsideMachine(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	m := gomad.NewSimpleMachine(func() {
		log.Println("before")
		panic("help")
		log.Println("after")
	})
	m.Wait()
	log.Println("never")
}

func TestTErrorDoesNotAbort(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	ch := make(chan struct{})

	go func() {
		log.Println("before")
		t.Error("help")
		log.Println("after")
		close(ch)
	}()

	<-ch
	log.Println("done")
}

/*
// TODO: Port
func TestNestedNondeterminismLog(t *testing.T) {
	config := gomad.SimConfig{
		Seeds:            []int64{1},
		DontReportFail:   true,
		CaptureLog:       true,
		LogLevelOverride: "INFO",
	}

	result := gomad.RunNestedTest(t, config, gomad.Test{Test: func(t *testing.T) {
		config := gomad.SimConfig{
			Seeds:                []int64{1},
			EnableTracer:         true,
			CheckDeterminismRuns: 1,
			CaptureLog:           true,
			LogLevelOverride:     "INFO",
		}

		result := gomad.RunNestedTest(t, config, gomad.Test{Test: func(t *testing.T) {
			t.Log(gomadruntime.Nondeterministic())
		}})

		if len(result) != 2 {
			t.Error("expected 2 runs, got", len(result))
		}
		if result[0].Failed || result[0].Err != nil {
			t.Error("expected success for first run")
		}
		if result[1].Failed || result[1].Err != nil {
			t.Error("expected success second run")
		}
	}})

	if !result[0].Failed {
		t.Error("expected failure")
	}

	if diff := cmp.Diff(gomad.SimplifyParsedLog(gomad.ParseLog(result[0].LogOutput)), []string{
		"ERROR traces differ: non-determinism found",
		"ERROR logs differ: non-determinism found",
	}); diff != "" {
		t.Error(diff)
	}
}

func TestNondeterminism(t *testing.T) {
	if gomadruntime.Nondeterministic()%2 == 0 {
		go func() {
		}()
	}
}

func TestNestedNondeterminismTrace(t *testing.T) {
	config := gomad.SimConfig{
		Seeds:            []int64{1},
		DontReportFail:   true,
		CaptureLog:       true,
		LogLevelOverride: "INFO",
	}

	result := gomad.RunNestedTest(t, config, gomad.Test{Test: func(t *testing.T) {
		config := gomad.SimConfig{
			Seeds:                []int64{1},
			EnableTracer:         true,
			CheckDeterminismRuns: 1,
			CaptureLog:           true,
			LogLevelOverride:     "INFO",
		}

		result := gomad.RunNestedTest(t, config, gomad.Test{Test: func(t *testing.T) {
			if gomadruntime.Nondeterministic()%2 == 0 {
				go func() {
				}()
			}
		}})

		if len(result) != 2 {
			t.Error("expected 2 runs, got", len(result))
		}
		if result[0].Failed || result[0].Err != nil {
			t.Error("expected success for first run")
		}
		if result[1].Failed || result[1].Err != nil {
			t.Error("expected success second run")
		}
	}})

	if !result[0].Failed {
		t.Error("expected failure")
	}
	if result[0].Err != nil {
		t.Error("expected no error")
	}

	if diff := cmp.Diff(gomad.SimplifyParsedLog(gomad.ParseLog(result[0].LogOutput)), []string{
		"ERROR traces differ: non-determinism found",
	}); diff != "" {
		t.Error(diff)
	}
}
*/

func TestTName(t *testing.T) {
	// use Meta prefix so we don't run under TestDeterminism which gives a different name
	if t.Name() != "TestTName" {
		t.Error(t.Name())
	}
}

func TestNestedTName(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		if t.Name() != "TestNestedTName/subtest" {
			t.Error(t.Name())
		}
	})
}

func TestCleanup(t *testing.T) {
	var order []int

	// TODO: help, what about a clean-up inside of an aborted machine?

	t.Run("subtest", func(t *testing.T) {
		t.Cleanup(func() {
			order = append(order, 2)
		})
		t.Cleanup(func() {
			order = append(order, 1)
		})
		order = append(order, 0)
	})

	if diff := cmp.Diff([]int{0, 1, 2}, order); diff != "" {
		t.Error(diff)
	}
}

func TestIsSim(t *testing.T) {
	if !gomad.IsSim() {
		t.Error("should be sim")
	}
}

/*
// TODO: Port
func TestDeadlock(t *testing.T) {
	c := gomad.SimConfig{
		Seeds:          []int64{1},
		DontReportFail: true,
	}

	result := gomad.RunNestedTest(t, c, gomad.Test{Test: func(t *testing.T) {
		var mu1, mu2 sync.Mutex

		go func() {
			mu2.Lock()
			time.Sleep(time.Second)
			mu1.Lock()
		}()

		mu1.Lock()
		time.Sleep(time.Second)
		mu2.Lock()
	}})

	if !result[0].Failed {
		t.Error("need fail")
	}
	if result[0].Err != gomadruntime.ErrTimeout {
		t.Error(result[0].Err)
	}
}
*/

func TestHello(t *testing.T) {
	t.Log(os.Getenv("HELLO"))
}

func TestSimulationTimeout(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	log.Println("before")
	start := time.Now()
	time.Sleep(30 * time.Second)
	for range 20 {
		log.Println(time.Since(start))
		time.Sleep(time.Minute)
	}
	log.Println("after")
}

func TestSimulationTimeoutOverride(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	gomad.SetSimulationTimeout(5 * time.Minute)

	log.Println("before")
	start := time.Now()
	time.Sleep(30 * time.Second)
	for range 20 {
		log.Println(time.Since(start))
		time.Sleep(time.Minute)
	}
	log.Println("after")
}
