use strict;
use warnings;
use Digest::SHA;

my $prefix = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces';
open my $newer, '<', "$prefix/task-67/busy-loops-20261010/final-sources.sha256" or die $!;
my %expected;
while (my $line = <$newer>) {
    chomp $line;
    my ($hash, $path) = $line =~ /^([a-f0-9]{64})  (.+)$/;
    die "invalid newer manifest line\n" unless defined $path;
    $expected{$path} = $hash;
}
close $newer or die $!;
my %prior_only = map { $_ => 1 } qw(
    tools/gomad3integration/qualification/tests.json
    tools/gomad3integration/qualification/tests.generator.json
);
open my $input, '<', "$prefix/task-66/boundary-consumed-inputs.sha256" or die $!;
my ($count, $drift, $retained) = (0, 0, 0);
while (my $line = <$input>) {
    chomp $line;
    my ($older, $path) = $line =~ /^([a-f0-9]{64})  (.+)$/;
    die "invalid consumed-input manifest line\n" unless defined $path;
    my $hash = $expected{$path};
    if (!defined $hash) {
        die "$path absent from validated source binding\n" unless $prior_only{$path};
        $hash = $older;
        $retained++;
        print "RETAINED_TASK65_GENERATOR_INPUT $path $hash\n";
    }
    open my $file, '<', $path or die "$path: $!\n";
    binmode $file;
    my $actual = Digest::SHA->new(256)->addfile($file)->hexdigest;
    close $file or die $!;
    die "$path: actual consumed input differs\n" unless $actual eq $hash;
    if ($older ne $hash) {
        die "unexpected older drift $path\n" unless $path eq 'tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go';
        $drift++;
        print "OLDER_TASK66_DRIFT $path $older -> $actual\n";
    }
    $count++;
}
close $input or die $!;
die "consumed input inventory changed\n" unless $count == 554 && $drift == 1 && $retained == 2;
print "PASS: 552 actual consumed implementation/schema/template/output inputs equal newer task67 validated sources; two qualification generator inputs equal retained task65 validated inputs\n";
open my $diff, '-|', 'git', 'diff', '--name-only', '41ca15aefb849747a7fe73cb1821e0b7bf93bf58', '--', 'tests', 'tools/gomad3integration/qualification', 'go.mod', 'go.sum' or die $!;
my $changed = do { local $/; <$diff> };
close $diff or die 'git diff failed';
die "qualification validator's tracked source/module inputs changed: $changed\n" if length $changed;
print "PASS: qualification validator's tracked source/module inputs remain unchanged\n";
print "Generator/package-architecture receipts: task67/busy-loops-20261010/final-{validate,package-architecture}.json. Qualification generator and private API/public-consumer receipts: task65/final-{validate,boundaries}-corrected.json, limited to their unchanged actual consumed inputs.\n";
print "Current supported-source-set Runner vet covers the changed test body separately. No fresh generator, external-cache qualification, native pass or full source acceptance is inferred.\n";
print "The first fast-lint observation lacked tests/mixedbrain/go.mod and go.sum outside this 554-path inventory; root materialized them before the separately bound successor. Earlier receipts retain that absence limit; no whole-repository consumed-input equality is claimed.\n";
