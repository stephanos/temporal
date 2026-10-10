use strict;
use warnings;
use Digest::SHA;

my $prefix = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces';
my $paths_manifest = "$prefix/task-66/boundary-consumed-inputs.sha256";
my $current_manifest = "$prefix/task-67/busy-loops-20261010/final-sources.sha256";
open my $newer, '<', $current_manifest or die $!;
my %expected;
while (my $line = <$newer>) {
    chomp $line;
    my ($hash, $path) = $line =~ /^([a-f0-9]{64})  (.+)$/;
    die "invalid newer manifest line\n" unless defined $path;
    $expected{$path} = $hash;
}
close $newer or die $!;
open my $input, '<', $paths_manifest or die $!;
my ($count, $drift) = (0, 0);
while (my $line = <$input>) {
    chomp $line;
    my ($older, $path) = $line =~ /^([a-f0-9]{64})  (.+)$/;
    die "invalid consumed-input manifest line\n" unless defined $path;
    die "$path absent from newer validated source binding\n" unless exists $expected{$path};
    open my $file, '<', $path or die "$path: $!\n";
    binmode $file;
    my $actual = Digest::SHA->new(256)->addfile($file)->hexdigest;
    close $file or die $!;
    die "$path: newer consumed input differs\n" unless $actual eq $expected{$path};
    if ($older ne $expected{$path}) {
        $drift++;
        print "OLDER_TASK66_DRIFT $path $older -> $actual\n";
    }
    $count++;
}
close $input or die $!;
die "consumed input count changed\n" unless $count == 554;
die "unexpected task66-to-task67 drift count\n" unless $drift == 1;
print "PASS: all $count actual boundary implementation/schema/template/output inputs match task67's newer validated source manifest\n";
open my $diff, '-|', 'git', 'diff', '--name-only', '41ca15aefb849747a7fe73cb1821e0b7bf93bf58', '--', 'tests', 'tools/gomad3integration/qualification', 'go.mod', 'go.sum' or die $!;
my $changed = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die "qualification validator's tracked source/module inputs changed: $changed\n" if length $changed;
open my $local_diff, '-|', 'git', 'diff', '--name-only', '7727b062b0c263046f0409e8f9d6cf5e58e7c0ef', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration' or die $!;
my $local = do { local $/; <$local_diff> };
close $local_diff or die 'git diff failed';
die "unexpected product change: $local\n" unless $local eq "tools/gomad3/runner/runner_test.go\n";
print "PASS: qualification validator's tracked source/module inputs unchanged; only three statements in an existing Runner test body differ from task68 BASE\n";
print "Retained current generator/package-architecture receipts: task67/busy-loops-20261010/final-{validate,package-architecture}.json. Private API/public-consumer receipts: task65/final-boundaries-corrected.json, applied only to unchanged production inputs.\n";
print "Current supported-source-set Runner vet runs cover the changed test body separately. No generator, external-cache qualification, native pass or full source acceptance is inferred.\n";
print "Sparse checkout omits tests/mixedbrain/go.mod and tests/mixedbrain/go.sum; those files are not in this 554-path inventory. No whole-repository consumed-input equality is claimed.\n";
