//go:build gomad

package behavior_test

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/temporalio/gomad"
	"github.com/temporalio/gomad/gomadruntime"
)

func TestMachineCrash(t *testing.T) {
	m := gomad.NewSimpleMachine(func() {
		// sleep a second in both the machine and with the crash call to make
		// sure they start "at the same time". all global initialization code
		// runs before.
		time.Sleep(time.Second)
		for i := 1; i <= 3; i++ {
			gomadruntime.Yield()
			slog.Info("output", "n", i)
		}
	})
	time.Sleep(time.Second)
	m.Crash()
}

func TestMachineGo(t *testing.T) {
	global := gomad.CurrentMachine()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if global != gomad.CurrentMachine() {
			t.Error("expected current machine to stay same")
		}
	}()

	wg.Add(1)
	go func() {
		go func() {
			go func() {
				go func() {
					defer wg.Done()
					if global != gomad.CurrentMachine() {
						t.Error("expected current machine to stay same")
					}
				}()
			}()
		}()
	}()

	wg.Add(1)
	gomad.NewSimpleMachine(func() {
		inner := gomad.CurrentMachine()
		if inner == global {
			t.Error("expected global to be different from inner")
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			defer wg.Done()
			if inner != gomad.CurrentMachine() {
				t.Error("expected current machine to stay same")
			}
		}()

		// wait until done; if the machine main returns, all goroutines will stop
		// XXX: test that also somehow???
		<-done
	})

	wg.Wait()
}

func TestMachineCurrent(t *testing.T) {
	global := gomad.CurrentMachine()

	if global != gomad.CurrentMachine() {
		t.Error("expected current machine to stay same")
	}

	var inner gomad.Machine
	gomadruntime.TestRaceToken.Release()
	m := gomad.NewSimpleMachine(func() {
		gomadruntime.TestRaceToken.Acquire()
		inner = gomad.CurrentMachine()
		// if inner == nil {
		// t.Error("expected non-nil inner")
		// }
		if inner != gomad.CurrentMachine() {
			t.Error("expected current machine to stay same")
		}
		if inner == global {
			t.Error("expected different global and inner")
		}
		gomadruntime.TestRaceToken.Release()
	})
	m.Wait()
	gomadruntime.TestRaceToken.Acquire()
	if inner != m {
		t.Error("expected inner to match m")
	}

	var inner2 gomad.Machine
	gomadruntime.TestRaceToken.Release()
	m2 := gomad.NewSimpleMachine(func() {
		gomadruntime.TestRaceToken.Acquire()
		inner2 = gomad.CurrentMachine()
		if inner2 != gomad.CurrentMachine() {
			t.Error("expected current machine to stay same")
		}
		// if inner2 == nil {
		// t.Error("expected non-nil inner2")
		// }
		if inner2 == global {
			t.Error("expected different global and inner2")
		}
		gomadruntime.TestRaceToken.Release()
	})
	m2.Wait()
	gomadruntime.TestRaceToken.Acquire()
	if inner2 != m2 {
		t.Error("expected inner2 to match m2")
	}

	var inner3 gomad.Machine
	m2.SetMainFunc(func() {
		gomadruntime.TestRaceToken.Acquire()
		inner3 = gomad.CurrentMachine()
		gomadruntime.TestRaceToken.Release()
	})
	gomadruntime.TestRaceToken.Release()
	m2.Restart()
	m2.Wait()
	gomadruntime.TestRaceToken.Acquire()
	if inner3 != m2 {
		t.Error("expected current machine to stay same")
	}
}

func TestMachineTimer(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	gomad.NewSimpleMachine(func() {
		inner := gomad.CurrentMachine()
		go func() {
			time.AfterFunc(time.Second, func() {
				if gomad.CurrentMachine() != inner {
					t.Error("expected timer machine to match inner")
				}
				wg.Done()
			})
		}()
		select {}
	})

	global := gomad.CurrentMachine()
	wg.Add(1)
	go func() {
		time.AfterFunc(time.Second, func() {
			if gomad.CurrentMachine() != global {
				t.Error("expected timer machine to match global")
			}
			wg.Done()
		})
		select {}
	}()

	wg.Wait()
}

var testGlobal = 0

func init() {
	testGlobal = 3
}

func TestMachineGlobals(t *testing.T) {
	if testGlobal != 3 {
		t.Error("expected 3")
	}
	testGlobal = 1

	gomad.NewSimpleMachine(func() {
		if testGlobal != 3 {
			t.Error("expected 3")
		}
		testGlobal = 2
	}).Wait()

	if testGlobal != 1 {
		t.Error("expected 1")
	}
}

func TestMachineLabel(t *testing.T) {
	m := gomad.NewMachine(gomad.MachineConfig{
		Label:    "hello",
		MainFunc: func() {},
	})
	if got := m.Label(); got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}
