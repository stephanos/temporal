//go:build unix

package conformance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
)

func TestExecWrapperRequiresSeedAndCommand(t *testing.T) {
	wrapper := execWrapper(t)
	for _, test := range []struct {
		name    string
		command []string
		env     []string
		want    string
	}{
		{name: "seed", command: []string{wrapper, "true"}, env: withoutEnvironment("GOMADV3_CHILD_SEED"), want: "GOMADV3_CHILD_SEED is required"},
		{name: "command", command: []string{wrapper}, env: withEnvironment("GOMADV3_CHILD_SEED", "1"), want: "target command is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runCommand(t, test.command, test.env)
			if result.Termination != hostexec.TerminationExit || result.ExitCode != 125 || !strings.Contains(string(result.Stderr.Bytes), test.want) {
				t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
			}
		})
	}
}

func TestExecWrapperTransfersSeedAndArguments(t *testing.T) {
	command := []string{
		execWrapper(t), "/bin/sh", "-c",
		`printf "seed=%s child=%s arg1=%s arg2=%s" "$GOMADSEED" "${GOMADV3_CHILD_SEED-unset}" "$1" "$2"`,
		"sh", "two words", "*",
	}
	environment := withEnvironment("GOMADV3_CHILD_SEED", "18446744073709551615")
	environment = replaceEnvironment(environment, "GOMADSEED", "inherited")
	result := runCommand(t, command, environment)
	if result.ExitCode != 0 || string(result.Stdout.Bytes) != "seed=18446744073709551615 child=unset arg1=two words arg2=*" {
		t.Fatalf("result = %#v, stdout = %q", result, result.Stdout.Bytes)
	}
}

func TestExecWrapperPreservesTargetExitAndSignal(t *testing.T) {
	wrapper := execWrapper(t)
	environment := withEnvironment("GOMADV3_CHILD_SEED", "1")
	exited := runCommand(t, []string{wrapper, "/bin/sh", "-c", "exit 37"}, environment)
	if exited.Termination != hostexec.TerminationExit || exited.ExitCode != 37 {
		t.Fatalf("exit result = %#v", exited)
	}
	signaled := runCommand(t, []string{wrapper, "/bin/sh", "-c", "kill -TERM $$"}, environment)
	if signaled.Termination != hostexec.TerminationSignal || signaled.SignalNumber != 15 {
		t.Fatalf("signal result = %#v", signaled)
	}
}

func execWrapper(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "exec.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func runCommand(t *testing.T, command, environment []string) hostexec.Result {
	t.Helper()
	result, err := hostexec.Run(context.Background(), hostexec.Request{
		Command: command, Dir: t.TempDir(), Env: environment, Timeout: 5 * time.Second,
		TerminateGrace: 100 * time.Millisecond, OutputLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func withoutEnvironment(name string) []string {
	prefix := name + "="
	result := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func withEnvironment(name, value string) []string {
	return replaceEnvironment(os.Environ(), name, value)
}

func replaceEnvironment(environment []string, name, value string) []string {
	result := withoutEnvironmentFrom(environment, name)
	return append(result, name+"="+value)
}

func withoutEnvironmentFrom(environment []string, name string) []string {
	prefix := name + "="
	result := make([]string, 0, len(environment))
	for _, value := range environment {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}
