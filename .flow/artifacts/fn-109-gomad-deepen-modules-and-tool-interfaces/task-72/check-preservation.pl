use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
my $base = 'effaf6a00ab79332c9b85541955d21eed28779e2';
my $path = 'tools/gomad3/runner/seed_completion_characterization_test.go';
open my $original, '-|', 'git', 'show', "$base:$path" or die $!;
my $before = do { local $/; <$original> };
close $original or die 'git show failed';
die 'BASE hash differs' unless sha256_hex($before) eq 'b35b3ec2d623c5746cc58e8996ff033b8e584813e6308f5fe7e0a03fde76e2b6';
open my $candidate, '<', $path or die $!;
binmode $candidate;
my $after = do { local $/; <$candidate> };
close $candidate or die $!;
my $reconstructed = $after;
for my $name (qw(TestSeedCompletionKeepsCampaignStatistics TestSeedCompletionFaultsKeepCampaignStatistics)) {
    my $count = $reconstructed =~ s{(func \Q$name\E\(t \*testing\.T\) \{.*?\n)(?=func |\z)}{
        my $body = $1;
        my $removed = $body =~ s{\t+config\.dependencies = scriptedPreparationDependencies\(t, config\.Preparer, config\.dependencies\.executor\)\n(?=\t+summary, err := exploreWith)}{}g;
        die "$name assignment/location differs" unless $removed == 1;
        $body;
    }gse;
    die "$name function count differs" unless $count == 1;
}
die 'whole file reconstruction differs' unless $reconstructed eq $before;
open my $diff, '-|', 'git', 'diff', '--numstat', $base, '--', $path or die $!;
my $numstat = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die 'not exactly two insertions and zero deletions' unless $numstat eq "2\t0\t$path\n";
open my $paths, '-|', 'git', 'diff', '--name-only', $base, '--', 'tools', 'tests', 'cmd/tools/lintcode', '.github/.golangci.yml', 'Makefile', 'go.mod', 'go.sum' or die $!;
my $changed = do { local $/; <$paths> };
close $paths or die 'git diff failed';
die "excluded product changed: $changed" unless $changed eq "$path\n";
printf "PASS: two exact assignments; original=%s candidate=%s reconstructed=%s; whole BASE restored; all excluded product paths unchanged\n", sha256_hex($before), sha256_hex($after), sha256_hex($reconstructed);
