use strict;
use warnings;
my @argv = ('/usr/bin/ps', '-eo', 'pid,ppid,etime,args');
printf "process_snapshot_argv=%s\n", join(' ', @argv);
open my $processes, '-|', @argv or die $!;
my @snapshot = <$processes>;
close $processes or die "ps failed: $?";
printf "snapshot_rows=%d; only gate candidates retained below\n", scalar @snapshot;
my @active;
for my $line (@snapshot) {
    next unless $line =~ /^\s*(\d+)\s+(\d+)\s+\S+\s+(\S+)(.*)$/;
    my ($pid, $executable, $rest) = ($1, $3, $4);
    $executable =~ s{.*/}{};
    if ($executable =~ /^(?:go|gofmt|make|gmake|gomadtool|errortype|golangci-lint(?:-v[\d.]+)?|.+\.test)$/ || ($executable eq 'timeout' && $rest =~ /\s(?:\S*\/)?(?:go|make|gomadtool|errortype|golangci-lint(?:-v[\d.]+)?)(?:\s|$)/)) {
        my $cwd = readlink "/proc/$pid/cwd";
        die "candidate process cwd unobservable PID=$pid" unless defined $cwd;
        printf "candidate PID=%d cwd=%s argv=%s%s\n", $pid, $cwd, $executable, $rest;
        if ($cwd eq '/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence' || index($cwd, '/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence/') == 0 || $rest =~ m{/Users/stephan/Workspace/skunkworks/\.gomad-scripted-and-spin-corrections\.gFiXmTVr/choice-divergence(?:/|\s|$)}) {
            push @active, $line;
        } else {
            print "foreign candidate excluded by actual cwd/argv; no mutation\n";
        }
    }
}
die "active shared-lane children observed:\n" . join('', @active) if @active;
print "PASS actual process snapshot: no task-attributable Go/build/lint/vet/generator/test executable or timed gate remains\n";
