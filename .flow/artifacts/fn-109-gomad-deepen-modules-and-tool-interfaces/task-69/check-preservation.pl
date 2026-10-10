use strict;
use warnings;
use Digest::SHA qw(sha256_hex);

my $base = 'c506713ce063759c5d24129d775d2fefc6314618';
my $path = 'tools/gomad3/runner/retention_test.go';
open my $original, '-|', 'git', 'show', "$base:$path" or die $!;
my $before = do { local $/; <$original> };
close $original or die 'git show failed';
die 'BASE hash differs' unless sha256_hex($before) eq '35a5599194d809a364751743318dc3c9e7304810bbeb8fd819e69dc8d3f14194';
open my $candidate, '<', $path or die $!;
binmode $candidate;
my $after = do { local $/; <$candidate> };
close $candidate or die $!;
my $reconstructed = $after;
my @names = qw(TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts);
for my $name (@names) {
    my $count = $reconstructed =~ s{(func \Q$name\E\(t \*testing\.T\) \{.*?\n)(?=func |\z)}{
        my $body = $1;
        my $removed = $body =~ s{(\t+config\.SuccessBytesLimit = [^\n]+\n)\t+configDependencies = scriptedPreparationDependencies\(t, config\.Preparer, configDependencies\.executor\)\n(?=\t+(?:return|summary, err :=) exploreWith\()}{ $1 }e;
        die "$name assignment/location differs" unless $removed == 1;
        $body;
    }gse;
    die "$name function count differs" unless $count == 1;
}
die 'whole file reconstruction differs' unless $reconstructed eq $before;
open my $diff, '-|', 'git', 'diff', '--name-only', $base, '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'cmd/tools/lintcode', '.github/.golangci.yml', 'Makefile', 'go.mod', 'go.sum', 'tests/mixedbrain' or die $!;
my $changed = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die "excluded source changed: $changed" unless $changed eq "$path\n";
printf "PASS: two exact assignments; original=%s candidate=%s reconstructed=%s; whole BASE bytes restored; excluded sources unchanged\n", sha256_hex($before), sha256_hex($after), sha256_hex($reconstructed);
