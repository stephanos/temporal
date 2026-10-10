use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
my $base = 'fd9cfc4026db966a877597a9658a0af589f18aea';
my $path = 'tools/gomad3/runner/choice_exploration_divergence_test.go';
open my $original, '-|', 'git', 'show', "$base:$path" or die $!;
my $before = do { local $/; <$original> };
close $original or die 'git show failed';
die 'BASE hash differs' unless sha256_hex($before) eq '93b138931736026645f0c4f05940a95ef8fe301de36bc3dbab3bea9c4c4b23df';
open my $candidate, '<', $path or die $!;
binmode $candidate;
my $after = do { local $/; <$candidate> };
close $candidate or die $!;
my $reconstructed = $after;
for my $name (qw(TestRunChoiceExplorationDivergencePoliciesAndInspection TestRunChoiceExplorationKeepsOtherErrorsAsHostErrors TestRunChoiceExplorationRetainsOnlyPrefixMismatchReasons)) {
    my $count = $reconstructed =~ s{(func \Q$name\E\(t \*testing\.T\) \{.*?\n)(?=func |\z)}{
        my $body = $1;
        my $removed = $body =~ s{\t\t\tconfigDependencies = scriptedPreparationDependencies\(t, config\.Preparer, configDependencies\.executor\)\n(?=\t\t\tsummary, err := exploreWith)}{}g;
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
