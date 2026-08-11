#!/usr/bin/env bash

gomad_run_checked() {
	if [[ $# -lt 6 || $5 != -- ]]; then
		printf 'usage: gomad_run_checked <seconds> <expected-status> <label> <result-dir> -- <command> [args...]\n' >&2
		return 125
	fi
	local seconds=$1
	local expected_status=$2
	local label=$3
	local result_dir=$4
	shift 5
	if [[ ! "$seconds" =~ ^[1-9][0-9]*$ || ! "$expected_status" =~ ^[0-9]+$ ]]; then
		printf 'gomadv3 checked runner requires a positive timeout and numeric expected status\n' >&2
		return 125
	fi

	mkdir -p "$result_dir"
	local stdout_file="$result_dir/stdout"
	local stderr_file="$result_dir/stderr"
	local status_file="$result_dir/status"
	local timed_out_file="$result_dir/timed-out"
	: >"$stdout_file"
	: >"$stderr_file"
	printf '0\n' >"$timed_out_file"

	local actual_status
	if perl -MPOSIX=setpgid -e '
		use strict;
		use warnings;
		my ($seconds, $stdout_file, $stderr_file, $timed_out_file, @command) = @ARGV;
		my $pid = fork();
		die "fork failed: $!\n" unless defined $pid;
		if ($pid == 0) {
			open STDOUT, ">", $stdout_file or die "open $stdout_file: $!\n";
			open STDERR, ">", $stderr_file or die "open $stderr_file: $!\n";
			setpgid(0, 0) or die "setpgid failed: $!\n";
			exec {$command[0]} @command or do {
				print STDERR "exec $command[0] failed: $!\n";
				exit 127;
			};
		}
		setpgid($pid, $pid);
		my $status;
		my $completed = eval {
			local $SIG{ALRM} = sub { die "timeout\n" };
			alarm $seconds;
			waitpid $pid, 0;
			$status = $?;
			alarm 0;
			1;
		};
		if (!$completed) {
			alarm 0;
			kill "KILL", -$pid;
			waitpid $pid, 0;
			open my $timed_out, ">", $timed_out_file or die "open $timed_out_file: $!\n";
			print {$timed_out} "1\n";
			close $timed_out or die "close $timed_out_file: $!\n";
			exit 124;
		}
		exit(($status & 127) ? 128 + ($status & 127) : $status >> 8);
	' "$seconds" "$stdout_file" "$stderr_file" "$timed_out_file" "$@"; then
		actual_status=0
	else
		actual_status=$?
	fi
	printf '%d\n' "$actual_status" >"$status_file"
	if [[ $actual_status -eq $expected_status ]]; then
		if [[ $expected_status -eq 124 && $(<"$timed_out_file") != 1 ]]; then
			printf 'gomadv3 process failed: %s: status 124 was not a timeout\n' "$label" >&2
			return 1
		fi
		return 0
	fi

	printf 'gomadv3 process failed: %s: status %d, want %d\n' \
		"$label" "$actual_status" "$expected_status" >&2
	if [[ -s "$stdout_file" ]]; then
		printf '%s\n' '--- stdout ---' >&2
		while IFS= read -r line || [[ -n "$line" ]]; do
			printf '%s\n' "$line" >&2
		done <"$stdout_file"
	fi
	if [[ -s "$stderr_file" ]]; then
		printf '%s\n' '--- stderr ---' >&2
		while IFS= read -r line || [[ -n "$line" ]]; do
			printf '%s\n' "$line" >&2
		done <"$stderr_file"
	fi
	return 1
}
