//go:build gomad

package behavior_test

/*
func TestNemesisDelayInner(t *testing.T) {
	go nemesis.Nemesis(1)

	ch := make(chan struct{}, 2)

	var done1, done2 time.Time

	go func() {
		defer func() { ch <- struct{}{} }()
		time.Sleep(time.Second)
		done1 = time.Now()
	}()

	go func() {
		defer func() { ch <- struct{}{} }()
		time.Sleep(time.Second)
		done2 = time.Now()
	}()

	<-ch
	<-ch

	slog.Info("delay", "val", float64(done2.Sub(done1)/time.Second))
}

func TestNemesisCrashInner(t *testing.T) {
	slog.Info("start", "now", time.Now().Unix())
	f := func() {
		for i := 0; i < 5; i++ {
			gomad.Yield()
		}
		slog.Info("hello", "now", time.Now().Unix())
		// slog.Info("hello", "id", gomadruntime.CurrentMachine()) // logs current machine ID, reboot iteration?
		// time.Sleep(time.Second)
	}
	m := gomad.NewSimpleMachine(f)
	go nemesis.Restarter(0, []gomad.Machine{m})
	// let test do its thing
	time.Sleep(time.Minute)
}
*/
