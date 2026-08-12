package intercept_test

import (
	"testing"

	"gomadv3.test/intercept"
	"gomadv3.test/interceptcaller"
)

func TestCompilerDefinitionInterceptionCoversCrossPackageCalls(t *testing.T) {
	intercept.SetHandled(true)
	t.Cleanup(func() { intercept.SetHandled(false) })
	if result, source := interceptcaller.Function(2); result != 12 || source != "intercepted" {
		t.Fatalf("Function(2) = %d, %q", result, source)
	}
	if result := interceptcaller.Add(&intercept.Value{Base: 3}, 4); result != 107 {
		t.Fatalf("Value.Add(4) = %d", result)
	}
}
