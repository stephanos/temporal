use strict;
use warnings;
use Digest::SHA qw(sha256_hex);

my $path = 'tools/gomad3/runner/runner_test.go';
open my $base_file, '<', '.flow/tmp/base_commit' or die $!;
chomp(my $base = <$base_file>);
close $base_file or die $!;
open my $original_file, '-|', 'git', 'show', "$base:$path" or die $!;
my $original = do { local $/; <$original_file> };
close $original_file or die 'git show failed';
open my $current_file, '<', $path or die $!;
my $current = do { local $/; <$current_file> };
close $current_file or die $!;
my @names = qw(
    TestRunReportsPreparationProgressAndCompletedCounts
    TestRunReportsPeriodicProgressWhileTargetIsRunning
    TestRunReportsPassiveSemanticCoverage
    TestRunFailsWhenRequiredSemanticProbeIsMissing
    TestRunRetainsOnlyProbeNovelSuccessesWithinExplicitBounds
    TestRunRetainsOnlyChoiceNovelSuccessesAndRecordsTheirFeatures
    TestRunMergesParallelCompletionsInSelectionOrdinalOrder
    TestRunFailsClosedWhenSuccessRetentionCountIsExhausted
    TestRunRejectsSuccessfulRetentionWithoutReplayTranscript
    TestRunCollectsBoundedQualificationEvidenceForOneSeed
);
my $assignment = "\tconfigDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)\n";
my $stripped = $current;
for my $name (@names) {
    my $matched = $stripped =~ s{(^func \Q$name\E\(t \*testing\.T\) \{\n)(.*?)(?=^func |\z)}{
        my ($header, $body) = ($1, $2);
        my $count = $body =~ s/\Q$assignment\E//g;
        die "$name: removed $count assignments, expected one\n" unless $count == 1;
        "$header$body";
    }mseg;
    die "$name: matched $matched functions, expected one\n" unless $matched == 1;
}
die "original file differs after selective removal\n" unless $stripped eq $original;
my $old_calls = () = $original =~ /scriptedPreparationDependencies\(/g;
my $new_calls = () = $current =~ /scriptedPreparationDependencies\(/g;
die "task65 baseline call count changed\n" unless $old_calls == 6 && $new_calls == 16;
my ($periodic) = $current =~ /(^func TestRunReportsPeriodicProgressWhileTargetIsRunning\(.*?)(?=^func )/ms;
die "periodic dependency attachment follows goroutine\n" unless index($periodic, $assignment) < index($periodic, "\tgo func() {");
print "base=$base\noriginal_sha256=" . sha256_hex($original) . "\ncurrent_sha256=" . sha256_hex($current) . "\n";
print "PASS: whole original file reproduced by removing exactly one assignment in each of ten admitted functions; six old calls intact; periodic assignment before goroutine\n";
