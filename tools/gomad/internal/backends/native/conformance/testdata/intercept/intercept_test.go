package intercept

import "testing"

// These tests only pass when the conformance compiler applied the hooks; under
// a stock compiler the targets run their original bodies.
func TestFunctionIsIntercepted(t *testing.T) {
	if got := Function(1); got != 1001 {
		t.Fatalf("Function(1) = %d, want the hook result 1001", got)
	}
	if hookCalls["Function"] != 1 {
		t.Fatalf("hook calls = %d, want 1", hookCalls["Function"])
	}
}

func TestNotifyIsIntercepted(t *testing.T) {
	if got := Notify("a", "b"); got != "hooked:a|b" {
		t.Fatalf("Notify() = %q, want the hook result", got)
	}
	if got := Notify(); got != "hooked:" {
		t.Fatalf("Notify() with no values = %q", got)
	}
}

func TestMethodIsInterceptedAndCanDecline(t *testing.T) {
	value := &Value{}
	if got := value.Add(3); got != 6 {
		t.Fatalf("Add(3) = %d, want the hook result 6", got)
	}
	if got := value.Add(-1); got != 5 {
		t.Fatalf("declined Add(-1) = %d, want the original body result 5", got)
	}
	if hookCalls["Add"] != 2 {
		t.Fatalf("hook calls = %d, want 2", hookCalls["Add"])
	}
}
