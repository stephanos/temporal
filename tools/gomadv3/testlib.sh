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
	local output_truncated_file="$result_dir/output-truncated"
	: >"$stdout_file"
	: >"$stderr_file"
	printf '0\n' >"$timed_out_file"
	printf '0\n' >"$output_truncated_file"

	local actual_status
	if perl -e '
		use strict;
		use warnings;
		use IO::Select;
		use POSIX qw(setpgid WNOHANG);
		use Time::HiRes qw(time);
		sub drain_outputs {
			my ($selector, $outputs, $output_limit, $truncated, $wait) = @_;
			my @ready = $selector->can_read($wait);
			for my $input (@ready) {
				my $read = sysread $input, my $buffer, 65536;
				die "read child output failed: $!\n" unless defined $read;
				if ($read == 0) {
					$selector->remove($input);
					close $input;
					next;
				}
				my $output = $outputs->{fileno($input)};
				my $remaining = $output_limit - $output->[1];
				if ($remaining > 0) {
					my $retained = $read < $remaining ? $read : $remaining;
					print {$output->[0]} substr($buffer, 0, $retained) or die "write child output failed: $!\n";
					$output->[1] += $retained;
				}
				$$truncated = 1 if $read > $remaining;
			}
			return scalar @ready;
		}
		sub reap_bounded {
			my ($pid, $seconds) = @_;
			my $deadline = time + $seconds;
			while (time <= $deadline) {
				my $waited = waitpid $pid, WNOHANG;
				die "waitpid failed: $!\n" if $waited < 0;
				return 1 if $waited == $pid;
				select undef, undef, undef, 0.01;
			}
			return 0;
		}
		my ($seconds, $stdout_file, $stderr_file, $timed_out_file, $output_truncated_file, $output_limit, @command) = @ARGV;
		pipe my $stdout_read, my $stdout_write or die "stdout pipe failed: $!\n";
		pipe my $stderr_read, my $stderr_write or die "stderr pipe failed: $!\n";
		my $pid = fork();
		die "fork failed: $!\n" unless defined $pid;
		if ($pid == 0) {
			close $stdout_read;
			close $stderr_read;
			open STDOUT, ">&", $stdout_write or die "redirect stdout: $!\n";
			open STDERR, ">&", $stderr_write or die "redirect stderr: $!\n";
			close $stdout_write;
			close $stderr_write;
			setpgid(0, 0) or die "setpgid failed: $!\n";
			exec {$command[0]} @command or do {
				print STDERR "exec $command[0] failed: $!\n";
				exit 127;
			};
		}
		close $stdout_write;
		close $stderr_write;
		open my $stdout_output, ">", $stdout_file or die "open $stdout_file: $!\n";
		open my $stderr_output, ">", $stderr_file or die "open $stderr_file: $!\n";
		setpgid($pid, $pid);
		my $selector = IO::Select->new($stdout_read, $stderr_read);
		my %outputs = (
			fileno($stdout_read) => [$stdout_output, 0],
			fileno($stderr_read) => [$stderr_output, 0],
		);
		my $truncated = 0;
		my $status;
		my $reaped = 0;
		my $completed = eval {
			local $SIG{ALRM} = sub { die "timeout\n" };
			alarm $seconds;
			while ($selector->count || !$reaped) {
				drain_outputs($selector, \%outputs, $output_limit, \$truncated, 0.1);
				if (!$reaped) {
					my $waited = waitpid $pid, WNOHANG;
					die "waitpid failed: $!\n" if $waited < 0;
					if ($waited == $pid) {
						$status = $?;
						$reaped = 1;
					}
				}
			}
			alarm 0;
			1;
		};
		if (!$completed) {
			my $error = $@;
			alarm 0;
			my $group_signaled = kill "KILL", -$pid;
			my $child_signaled = $reaped ? 0 : kill "KILL", $pid;
			if (!$reaped && !reap_bounded($pid, 2)) {
				die "failed to terminate child $pid (group signal=$group_signaled, child signal=$child_signaled)\n";
			}
			my $drain_deadline = time + 2;
			while ($selector->count && time < $drain_deadline) {
				drain_outputs($selector, \%outputs, $output_limit, \$truncated, 0.05);
			}
			$truncated = 1 if $selector->count;
			if ($error eq "timeout\n") {
				open my $timed_out, ">", $timed_out_file or die "open $timed_out_file: $!\n";
				print {$timed_out} "1\n";
				close $timed_out or die "close $timed_out_file: $!\n";
			} else {
				die $error;
			}
		}
		close $stdout_output or die "close $stdout_file: $!\n";
		close $stderr_output or die "close $stderr_file: $!\n";
		if ($truncated) {
			open my $output_truncated, ">", $output_truncated_file or die "open $output_truncated_file: $!\n";
			print {$output_truncated} "1\n";
			close $output_truncated or die "close $output_truncated_file: $!\n";
		}
		exit 124 unless $completed;
		exit(($status & 127) ? 128 + ($status & 127) : $status >> 8);
	' "$seconds" "$stdout_file" "$stderr_file" "$timed_out_file" "$output_truncated_file" 1048576 "$@"; then
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
