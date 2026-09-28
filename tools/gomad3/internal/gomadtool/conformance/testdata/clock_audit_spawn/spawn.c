// clock_audit_spawn starts PROGRAM suspended and unslid, publishes its pid to
// PIDFILE, and exits with PROGRAM's status. An activated Gomad target
// re-executes itself in place when the kernel slid its image, which discards
// the pid-provider probes DTrace placed on the first image; starting the target
// unslid makes that re-execution a no-op, and starting it suspended lets DTrace
// attach before its first instruction. The caller resumes it with SIGCONT.
#include <spawn.h>
#include <stdio.h>
#include <string.h>
#include <sys/wait.h>

// The private flag debuggers set to load an image unslid, as in the Gomad
// runtime's gomad_aslr_darwin.go.
#define GOMAD_POSIX_SPAWN_DISABLE_ASLR 0x0100

extern char **environ;

int main(int argc, char **argv) {
	if (argc != 3) {
		fprintf(stderr, "usage: clock_audit_spawn PIDFILE PROGRAM\n");
		return 2;
	}
	posix_spawnattr_t attributes;
	if (posix_spawnattr_init(&attributes) != 0 ||
		posix_spawnattr_setflags(&attributes, POSIX_SPAWN_START_SUSPENDED | GOMAD_POSIX_SPAWN_DISABLE_ASLR) != 0) {
		fprintf(stderr, "clock_audit_spawn: cannot prepare spawn attributes\n");
		return 2;
	}
	char *child[] = {argv[2], NULL};
	pid_t pid;
	int err = posix_spawn(&pid, argv[2], NULL, &attributes, child, environ);
	if (err != 0) {
		fprintf(stderr, "clock_audit_spawn: posix_spawn %s: %s\n", argv[2], strerror(err));
		return 2;
	}
	char staged[1024];
	if (snprintf(staged, sizeof staged, "%s.tmp", argv[1]) >= (int)sizeof staged) {
		fprintf(stderr, "clock_audit_spawn: pid file path is too long\n");
		return 2;
	}
	FILE *file = fopen(staged, "w");
	if (file == NULL || fprintf(file, "%d\n", pid) < 0 || fclose(file) != 0 || rename(staged, argv[1]) != 0) {
		fprintf(stderr, "clock_audit_spawn: cannot publish pid %d\n", pid);
		return 2;
	}
	int status;
	if (waitpid(pid, &status, 0) != pid) {
		fprintf(stderr, "clock_audit_spawn: cannot wait for pid %d\n", pid);
		return 2;
	}
	return WIFEXITED(status) ? WEXITSTATUS(status) : 128 + WTERMSIG(status);
}
