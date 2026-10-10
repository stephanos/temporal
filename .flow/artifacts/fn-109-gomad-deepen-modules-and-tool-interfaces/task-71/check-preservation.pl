use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
my $base = '30a8847362779670311486706701233d0043be07';
my $path = 'tools/gomad3/runner/completion_characterization_test.go';
open my $original, '-|', 'git', 'show', "$base:$path" or die $!;
my $before = do { local $/; <$original> };
close $original or die 'git show failed';
die 'BASE hash differs' unless sha256_hex($before) eq 'b865856c22c519d3b9af29b65cbc5cf0c72b288b5380875f6809801c7a794e3f';
open my $candidate, '<', $path or die $!;
binmode $candidate;
my $after = do { local $/; <$candidate> };
close $candidate or die $!;
my $reconstructed = $after;
for my $name (qw(TestCompletionFaultsKeepReasonPrecedenceAndEvidence TestCancellationIsAHostFailure TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy)) {
    my $count = $reconstructed =~ s{(func \Q$name\E\(t \*testing\.T\) \{.*?\n)(?=func |\z)}{
        my $body = $1;
        my $removed = $body =~ s{\t+configDependencies = scriptedPreparationDependencies\(t, config\.Preparer, configDependencies\.executor\)\n(?=\t+summary, err := exploreWith)}{}g;
        die "$name assignment/location differs" unless $removed == 1;
        $body;
    }gse;
    die "$name function count differs" unless $count == 1;
}
die 'whole file reconstruction differs' unless $reconstructed eq $before;
open my $diff, '-|', 'git', 'diff', '--numstat', $base, '--', $path or die $!;
my $numstat = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die 'not exactly three insertions and zero deletions' unless $numstat eq "3\t0\t$path\n";
open my $paths, '-|', 'git', 'diff', '--name-only', $base, '--', 'tools', 'tests', 'cmd/tools/lintcode', '.github/.golangci.yml', 'Makefile', 'go.mod', 'go.sum' or die $!;
my $changed = do { local $/; <$paths> };
close $paths or die 'git diff failed';
die "excluded product changed: $changed" unless $changed eq "$path\n";
printf "PASS: three exact assignments; original=%s candidate=%s reconstructed=%s; whole BASE restored; all excluded product paths unchanged\n", sha256_hex($before), sha256_hex($after), sha256_hex($reconstructed);
