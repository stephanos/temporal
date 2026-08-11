package world

import "testing"

func BenchmarkWorldReadyAndDeliver(b *testing.B) {
	limits := testLimits()
	limits.MaxRequests = uint64(b.N) + 1
	limits.MaxEvents = uint64(b.N) + 1
	limits.MaxQueuedEvents = uint64(b.N) + 1
	limits.MaxTransitions = uint64(b.N)*3 + 1
	w, err := New(Config{Seed: 1, Limits: limits})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		id, registerErr := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "benchmark"}})
		if registerErr != nil {
			b.Fatal(registerErr)
		}
		if _, readyErr := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done"}); readyErr != nil {
			b.Fatal(readyErr)
		}
		if _, quiesceErr := w.Quiesce(); quiesceErr != nil {
			b.Fatal(quiesceErr)
		}
	}
}
