use strict;
use warnings;
use Digest::SHA qw(sha256_hex);

my $packet = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68';
my @observations;
for my $name (qw(baseline-runner-lint final-runner-lint)) {
    open my $file, '<', "$packet/$name.log" or die $!;
    my $data = do { local $/; <$file> };
    close $file or die $!;
    my @blocks = $data =~ /(^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:.*?)(?=^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:|^\d+ issues:)/msg;
    die "$name: expected six complete diagnostic blocks, got " . scalar(@blocks) . "\n" unless @blocks == 6;
    push @observations, join('', @blocks);
}
die "introduced, resolved or changed Runner lint block\n" unless $observations[0] eq $observations[1];
print "PASS: six unfiltered complete Runner diagnostic blocks unchanged; zero introduced or removed findings\n";
print "full_blocks_sha256=" . sha256_hex($observations[0]) . "\n";
print "Both unfiltered lint exits remain 1. Standalone errortype's exit 0 does not make aggregate lint green.\n";
