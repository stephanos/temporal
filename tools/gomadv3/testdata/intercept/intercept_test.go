package intercept

import "testing"

type adder interface {
	Add(int) int
}

func TestCompilerDefinitionInterceptionCoversInvocationForms(t *testing.T) {
	handled = true
	valueToNotify := 1
	Notify(&valueToNotify)
	if valueToNotify != 11 {
		t.Fatalf("Notify() value = %d", valueToNotify)
	}
	if result, source := Function(2); result != 12 || source != "intercepted" {
		t.Fatalf("Function(2) = %d, %q", result, source)
	}
	value := &Value{Base: 3}
	methodValue := value.Add
	for name, invoke := range map[string]func() int{
		"direct":            func() int { return value.Add(4) },
		"interface":         func() int { var abstract adder = value; return abstract.Add(4) },
		"method value":      func() int { return methodValue(4) },
		"method expression": func() int { return (*Value).Add(value, 4) },
	} {
		if result := invoke(); result != 107 {
			t.Errorf("%s Add() = %d", name, result)
		}
	}
	var nilValue *Value
	if result := nilValue.Add(4); result != -1 {
		t.Fatalf("nil Value.Add(4) = %d", result)
	}
}

func TestCompilerDefinitionInterceptionFallsThrough(t *testing.T) {
	handled = false
	valueToNotify := 1
	Notify(&valueToNotify)
	if valueToNotify != 2 {
		t.Fatalf("Notify() value = %d", valueToNotify)
	}
	if result, source := Function(2); result != 3 || source != "original" {
		t.Fatalf("Function(2) = %d, %q", result, source)
	}
	if result := (&Value{Base: 3}).Add(4); result != 7 {
		t.Fatalf("Value.Add(4) = %d", result)
	}
}
