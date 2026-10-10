use strict;
use warnings;
my @argv = ('/usr/bin/ps', '-eo', 'pid,ppid,etime,args');
printf "process_snapshot_argv=%s\n", join(' ', @argv);
open my $processes, '-|', @argv or die $!;
my @snapshot = <$processes>;
close $processes or die "ps failed: $?";
print @snapshot;
my @active;
for my $line (@snapshot) {
    next unless $line =~ /^\s*(\d+)\s+(\d+)\s+\S+\s+(\S+)(.*)$/;
    my ($pid, $executable, $rest) = ($1, $3, $4);
    $executable =~ s{.*/}{};
    if ($executable =~ /^(?:go|gofmt|make|gmake|gomadtool|errortype|golangci-lint(?:-v[\d.]+)?|.+\.test)$/ || ($executable eq 'timeout' && $rest =~ /\s(?:\S*\/)?(?:go|make|gomadtool|errortype|golangci-lint(?:-v[\d.]+)?)(?:\s|$)/)) {
        push @active, $line;
    }
}
die "active shared-lane children observed:\n" . join('', @active) if @active;
print "PASS actual process snapshot: no Go/build/lint/vet/generator/test executable or timed gate remains\n";
