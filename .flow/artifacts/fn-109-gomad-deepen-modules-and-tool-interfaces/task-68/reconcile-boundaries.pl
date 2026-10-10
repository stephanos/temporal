use strict;
use warnings;
use Digest::SHA qw(sha256_hex);

my $manifest = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-66/boundary-consumed-inputs.sha256';
open my $input, '<', $manifest or die $!;
my $count = 0;
while (my $line = <$input>) {
    chomp $line;
    my ($expected, $path) = $line =~ /^([a-f0-9]{64})  (.+)$/;
    die "invalid manifest line\n" unless defined $path;
    open my $file, '<', $path or die "$path: $!\n";
    binmode $file;
    my $actual = Digest::SHA->new(256)->addfile($file)->hexdigest;
    close $file or die $!;
    die "$path: consumed input changed\n" unless $actual eq $expected;
    $count++;
}
close $input or die $!;
die "consumed input count changed\n" unless $count == 554;
print "PASS: all $count retained boundary implementation/schema/template/output inputs equal current consumed files\n";
open my $diff, '-|', 'git', 'diff', '--name-only', '41ca15aefb849747a7fe73cb1821e0b7bf93bf58', '--', 'tests', 'tools/gomad3integration/qualification', 'go.mod', 'go.sum' or die $!;
my $changed = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die "qualification validator consumed source/module inputs changed: $changed\n" if length $changed;
open my $local_diff, '-|', 'git', 'diff', '--name-only', '7727b062b0c263046f0409e8f9d6cf5e58e7c0ef', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration' or die $!;
my $local = do { local $/; <$local_diff> };
close $local_diff or die 'git diff failed';
die "unexpected product change: $local\n" unless $local eq "tools/gomad3/runner/runner_test.go\n";
print "PASS: qualification validator's source/module inputs have no tracked changes since retained execution HEAD\n";
print "PASS: the only current product change is three statements in an existing test body; production declarations, package/source-set inventory, schemas, templates and generated outputs are unchanged\n";
print "Retained receipts: task-65/final-boundaries-corrected.json and task-65/final-validate-corrected.json; task-67/busy-loops-20261010/final-validate.json and combined-66-67/vet-{darwin-arm64,linux-amd64}.json\n";
print "This reconciles unchanged consumed source inputs only. It supplies no fresh generator or static execution, external-cache qualification, native pass or closure of earlier acceptance gaps.\n";
