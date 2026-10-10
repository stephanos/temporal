use strict;
use warnings;
use Digest::SHA qw(sha256_hex);

my @observations;
for my $path (@ARGV) {
    open my $file, '<', $path or die $!;
    my $data = do { local $/; <$file> };
    close $file or die $!;
    my @blocks = $data =~ /(^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:.*?)(?=^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:|^\d+ issues:)/msg;
    die "$path: expected six complete diagnostic blocks, got " . scalar(@blocks) . "\n" unless @blocks == 6;
    push @observations, join('', @blocks);
}
die 'expected two lint observations' unless @observations == 2;
die 'introduced, resolved or changed Runner lint block' unless $observations[0] eq $observations[1];
print "PASS: six complete unfiltered Runner diagnostics unchanged; zero introduced or removed\n";
print 'full_blocks_sha256=' . sha256_hex($observations[0]) . "\n";
