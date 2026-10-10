use strict;
use warnings;
use JSON::PP qw(decode_json);
use Digest::SHA qw(sha256_hex);
use Time::HiRes qw(time);
use POSIX qw(strftime);
my $root = '/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/seed-completion';
die 'wrong cwd' unless readlink("/proc/$$/cwd") eq $root;
sub read_bytes {
    my ($path) = @_;
    open my $file, '<', $path or die "$path: $!";
    binmode $file;
    my $bytes = do { local $/; <$file> };
    close $file or die $!;
    return $bytes;
}
my @bound = ($0, '/usr/bin/perl', '/usr/bin/ps', 'tools/gomad3/runner/seed_completion_characterization_test.go', @ARGV);
my %before = map { $_ => sha256_hex(read_bytes($_)) } @bound;
my $start = time;
my $started = strftime('%Y-%m-%dT%H:%M:%SZ', gmtime);
open my $ps, '-|', '/usr/bin/ps', '-eo', 'pid,ppid,args' or die $!;
my @snapshot = <$ps>;
close $ps or die 'ps command failed';
my (%parent, %command);
for my $line (@snapshot) {
    next unless $line =~ /^\s*(\d+)\s+(\d+)\s+(.*)$/;
    ($parent{$1}, $command{$1}) = ($2, $3);
}
my %ancestors;
for (my $pid = $$; $pid && !$ancestors{$pid}; $pid = $parent{$pid} // 0) { $ancestors{$pid} = 1; }
my (@attributed, @foreign_gates);
my $races = 0;
for my $pid (sort { $a <=> $b } keys %command) {
    next if $ancestors{$pid};
    my $exe = readlink("/proc/$pid/exe");
    my $cwd = readlink("/proc/$pid/cwd");
    unless (defined $exe && defined $cwd) { $races++; next; }
    next unless $exe =~ m{/(?:go|compile|link|vet|errortype|golangci-lint[^/]*|make|gomadtool|gofmt|[^/]+\.test)$};
    my $scope = $cwd eq $root || index($cwd, "$root/") == 0 || index($command{$pid}, $root) >= 0;
    if (open my $environment, '<', "/proc/$pid/environ") {
        my $bytes = do { local $/; <$environment> };
        close $environment or die $!;
        $scope ||= $bytes =~ /(?:^|\0)SANDBOX_START_DIR=\Q$root\E(?:\0|$)/;
    }
    my $record = {pid=>0+$pid, ppid=>0+$parent{$pid}, executable=>$exe, cwd=>$cwd, argv=>$command{$pid}};
    push @{ $scope ? \@attributed : \@foreign_gates }, $record;
}
my @receipt_exits;
for my $path (@ARGV) {
    my $receipt = decode_json(read_bytes($path));
    die "nonterminal receipt $path" unless defined $receipt->{exit} && defined $receipt->{ended} && $receipt->{pre_post_match_exit} == 0;
    push @receipt_exits, {path=>$path, exit=>0+$receipt->{exit}, ended=>$receipt->{ended}};
}
my %after = map { $_ => sha256_hex(read_bytes($_)) } @bound;
for my $path (@bound) { die "stand-down input/tool drift $path" unless $before{$path} eq $after{$path}; }
my $elapsed = time - $start;
my $output = {
    cwd=>$root, argv=>['/usr/bin/perl', $0, @ARGV], started=>$started,
    ended=>strftime('%Y-%m-%dT%H:%M:%SZ', gmtime), elapsed_seconds=>0+$elapsed,
    exit=>(@attributed ? 1 : 0), ps_argv=>['/usr/bin/ps', '-eo', 'pid,ppid,args'],
    ps_exit=>0, snapshot_process_count=>scalar(keys %command), ignored_ancestors=>[sort {$a<=>$b} keys %ancestors],
    process_races_or_unreadable=>$races, attributable_gate_children=>\@attributed,
    foreign_gate_processes_untouched=>\@foreign_gates, terminal_receipts=>\@receipt_exits,
    source_tools_inputs_before=>\%before, source_tools_inputs_after=>\%after, input_stability_exit=>0,
    limit=>'Current process snapshot attributed by executable plus cwd/argv/SANDBOX_START_DIR. Excludes checker ancestors; process races are disclosed, not global process absence.'
};
print JSON::PP->new->canonical->pretty->encode($output);
exit(@attributed ? 1 : 0);
