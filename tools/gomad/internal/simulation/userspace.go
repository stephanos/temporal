package simulation

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/temporalio/gomad/gomadruntime"
	"github.com/temporalio/gomad/internal/gomadlog"
)

// Per-machine globals initialized in setupUserspace:

var (
	linuxOS          *LinuxOS
	gomadOS          *GomadOS // XXX: elsewhere? in machine itself?
	currentMachineID int
)

func CurrentMachineID() int {
	return currentMachineID
}

type gomadSlogHandler struct {
	inner slog.Handler
}

func (w gomadSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return w.inner.Enabled(ctx, level)
}

func (w gomadSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.Int("goroutine", gomadruntime.GetGoroutine()))
	hasStep := false
	for attr := range r.Attrs {
		if attr.Key == "step" {
			hasStep = true
		}
	}
	if !hasStep {
		r.AddAttrs(slog.Int("step", gomadruntime.Step()))
	}
	if gomadruntime.TraceStack.Enabled() {
		r.AddAttrs(gomadlog.Stack(0, r.PC))
	}
	return w.inner.Handle(ctx, r)
}

func (w gomadSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return gomadSlogHandler{
		inner: w.inner.WithAttrs(attrs),
	}
}

func (w gomadSlogHandler) WithGroup(name string) slog.Handler {
	return gomadSlogHandler{
		inner: w.inner.WithGroup(name),
	}
}

type gomadLogWriter struct{}

func (w gomadLogWriter) Write(b []byte) (n int, err error) {
	gomadruntime.WriteLog(b)
	return len(b), nil
}

var (
	logInitialized = false
	logSyscalls    = false
)

func makeBaseSlogHandler() slog.Handler {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("GOMAD_LOG_LEVEL"))); err != nil {
		panic(err)
	}

	ho := slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}
	return slog.NewJSONHandler(gomadLogWriter{}, &ho)
}

func setupSlog(machineLabel string) {
	// We play a funny game with the logger. There exists a default slog.Logger in
	// every machine, since they all have their own set of globals. All loggers
	// point to the same LogOut writer and are set up using the configuration set
	// here.

	// TODO racedetector: make sure that using slog doesn't introduce sneaky
	// happens-before (this currently happens in the default handler which
	// uses both a pool and a lock)

	// stdout and stderr are currently captured with some special logic in
	// LinuxOS for writes to their file descriptors, see TestStdoutStderr.

	time.Local = time.UTC

	handler := makeBaseSlogHandler()

	// set short file flag so that we'll capture source info. see slog.SetDefault internals
	// XXX: test this?
	log.SetFlags(log.Lshortfile)
	slog.SetDefault(slog.New(gomadSlogHandler{inner: handler}).With("machine", machineLabel))

	logInitialized = true
	logSyscalls = gomadruntime.TraceSyscall.Enabled()
}

func setupUserspace(gomadOS_ *GomadOS, linuxOS_ *LinuxOS, machineID int, label string) {
	// initialize gomadOS etc. before invoking initializers so that init() calls
	// can make syscalls, see TestSyscallsDuringInit.
	gomadOS = gomadOS_
	linuxOS = linuxOS_
	currentMachineID = machineID

	gomadruntime.InitGlobals(false, false)

	// setupSlog only works once globals are initialized.  logs during init are
	// printed to stdout/stderr, see TestLogDuringInit.
	setupSlog(label)
}
