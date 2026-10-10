use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
die 'expected two environment observations' unless @ARGV == 2;
my @normalized;
for my $path (@ARGV) {
    open my $file, '<', $path or die $!;
    my $raw = do { local $/; <$file> };
    close $file or die $!;
    my $value = $raw;
    my $count = $value =~ s{^(\s*"GOGCCFLAGS": "[^"\n]*)/go-build[0-9]+=/tmp/go-build}{$1/go-build<ephemeral>=/tmp/go-build}mg;
    die 'GOGCCFLAGS ephemeral mapping missing or repeated' unless $count == 1;
    push @normalized, $value;
    printf "%s raw=%s normalized=%s replacements=%d\n", $path, sha256_hex($raw), sha256_hex($value), $count;
}
die 'environment changed outside exact GOGCCFLAGS ephemeral path' unless $normalized[0] eq $normalized[1];
print "PASS: every other environment byte and GOGCCFLAGS option unchanged\n";
