use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
die 'expected two lint observations' unless @ARGV == 2;
my @observations;
for my $path (@ARGV) {
    open my $file, '<', $path or die $!;
    my $data = do { local $/; <$file> };
    close $file or die $!;
    my @blocks = $data =~ /(^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:.*?)(?=^tools\/gomad3\/runner\/[^:\n]+:\d+:\d+:|^\d+ issues:)/msg;
    die "$path expected six complete blocks" unless @blocks == 6;
    my $blocks = join('', @blocks);
    die 'inherited complete block digest differs' unless sha256_hex($blocks) eq '0343dd7822baa18fb03d36e7917bd5c69ff5bd5b6e483f7a0f3c592ae52cb31b';
    push @observations, $blocks;
}
die 'Runner lint block changed' unless $observations[0] eq $observations[1];
print "PASS: six complete header/source/caret blocks unchanged; zero introduced or removed; SHA256=0343dd7822baa18fb03d36e7917bd5c69ff5bd5b6e483f7a0f3c592ae52cb31b\n";
