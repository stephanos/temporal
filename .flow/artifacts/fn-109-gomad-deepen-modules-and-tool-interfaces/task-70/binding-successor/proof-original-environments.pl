use strict;
use warnings;
use JSON::PP qw(decode_json);
use Digest::SHA qw(sha256_hex);
sub bytes {
    my ($path) = @_;
    open my $file, '<', $path or die "$path: $!";
    my $value = do { local $/; <$file> };
    close $file or die $!;
    return $value;
}
die 'expected exactly 19 original receipts' unless @ARGV == 19;
my %seen;
for my $path (@ARGV) {
    die "duplicate receipt $path" if $seen{$path}++;
    my $receipt = decode_json(bytes($path));
    $path =~ m{^(.*)/[^/]+$} or die 'receipt path must name directory';
    my $directory = $1;
    my @normalized;
    for my $field (qw(environment_sha256 post_environment_sha256)) {
        my $digest = $receipt->{$field};
        die "invalid environment digest $path" unless $digest =~ /^[0-9a-f]{64}$/;
        my $input = "$directory/environment_manifest-$digest";
        my $raw = bytes($input);
        die "input digest mismatch $input" unless sha256_hex($raw) eq $digest;
        my $value = $raw;
        my $count = $value =~ s{^(\s*"GOGCCFLAGS": "[^"\n]*)/go-build[0-9]+=/tmp/go-build}{$1/go-build<ephemeral>=/tmp/go-build}mg;
        die "GOGCCFLAGS ephemeral mapping missing or repeated $input" unless $count == 1;
        push @normalized, $value;
        printf "%s %s raw=%s normalized=%s replacements=%d\n", $path, $field, $digest, sha256_hex($value), $count;
    }
    die "environment differs outside exact single GOGCCFLAGS mapping $path" unless $normalized[0] eq $normalized[1];
    print "PASS $path every remaining byte and GOGCCFLAGS option equal\n";
}
print "PASS current verification of retained bytes: 19 pairs, 38 raw captures; no historical post-helper stability claim\n";
