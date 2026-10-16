package main

import (
	"os"
	"runtime"
	"strconv"
	"sync"

	"go.temporal.io/server/common/gomaddebug"
)

var (
	progress        = gomaddebug.DeclareProbe("fixture.progress")
	allocating      = gomaddebug.DeclareProbe("fixture.allocating")
	oversized       = gomaddebug.DeclareProbe("fixture.oversized")
	applicationLock sync.Mutex
	perturbation    []byte
	largeString     string
	completed       uint64
)

func init() {
	largeString = string(make([]byte, 32<<10))
	gomaddebug.RegisterDump("fixture.snapshot", func(w *gomaddebug.DumpWriter) {
		w.String("application lock is held; callback reads only frozen scalar state\n")
		if completed > 0 {
			w.String("progress observed\n")
		}
	})
	gomaddebug.RegisterDump("fixture.panic", func(*gomaddebug.DumpWriter) { panic("dump fixture") })
	gomaddebug.RegisterDump("fixture.truncated", func(w *gomaddebug.DumpWriter) {
		for range 4 {
			w.String(largeString)
		}
	})
	gomaddebug.RegisterDump("fixture.gc", func(w *gomaddebug.DumpWriter) {
		memory := make([]byte, 32<<20)
		for i := range memory {
			memory[i] = byte(i)
		}
		runtime.GC()
		runtime.KeepAlive(memory)
		w.String("32MiB allocation and collection completed with application frozen\n")
	})
	gomaddebug.RegisterInvariant("fixture.before_three", true, func() bool { return completed < 3 })
	gomaddebug.RegisterInvariant("fixture.recovering", false, func() bool { return completed != 2 && completed != 4 })
	gomaddebug.RegisterDump("fixture.timeout", func(*gomaddebug.DumpWriter) {
		for {
		}
	})
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "early":
			panic("early initialization fixture")
		case "duplicate":
			gomaddebug.DeclareProbe("fixture.progress")
		case "capacity":
			for i := range 256 {
				gomaddebug.DeclareProbe("fixture.extra." + strconv.Itoa(i))
			}
		}
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "late" {
		gomaddebug.DeclareProbe("fixture.late")
	}
	applicationLock.Lock()
	done := make(chan struct{})
	go func() {
		for range 8 {
			completed++
			if progress.Enabled() {
				progress.Emit(gomaddebug.Uint64(completed), gomaddebug.String("existing"), gomaddebug.Bool(true))
			}
			if oversized.Enabled() {
				oversized.Emit(gomaddebug.String(largeString))
			}
			if allocating.Enabled() {
				perturbation = make([]byte, 37)
			}
			runtime.Gosched()
		}
		close(done)
	}()
	for range 8 {
		runtime.Gosched()
	}
	<-done
	applicationLock.Unlock()
	os.Exit(1)
}
