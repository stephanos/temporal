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
die 'expected successor receipts' unless @ARGV;
for my $path (@ARGV) {
    my $receipt = decode_json(bytes($path));
    for my $kind (qw(source tools routing)) {
        my $initial = $receipt->{"${kind}_manifest_sha256"};
        my $post_child = $receipt->{"post_${kind}_manifest_sha256"};
        my $post_helper = $receipt->{"post_helper_${kind}_manifest_sha256"};
        die "missing post_helper_${kind}_manifest_sha256 in $path\n" unless defined $post_helper;
        die "changed $kind in $path\n" unless $initial eq $post_child && $initial eq $post_helper;
    }
    die "unstable receipt $path\n" unless $receipt->{pre_post_match_exit} == 0;
    die "helper failed $path\n" unless $receipt->{environment_proof_exit} == 0;
    die "wrapper changed $path\n" unless $receipt->{wrapper_sha256} eq $receipt->{post_wrapper_sha256};
    if ($receipt->{argv}[0] eq 'make') {
        die "Make shell not explicit $path\n" unless grep { $_ eq 'SHELL=/bin/sh' } @{$receipt->{argv}};
    }
    $path =~ m{^(.*)/[^/]+$} or die 'receipt path must name a directory';
    my $directory = $1;
    for my $kind (qw(source tools routing)) {
        my $digest = $receipt->{"${kind}_manifest_sha256"};
        die "manifest digest mismatch $path\n" unless sha256_hex(bytes("$directory/${kind}_manifest-$digest")) eq $digest;
    }
    for my $kind (qw(environment post_environment)) {
        my $digest = $receipt->{"${kind}_sha256"};
        die "environment digest mismatch $path\n" unless sha256_hex(bytes("$directory/environment_manifest-$digest")) eq $digest;
    }
    $path =~ m{/([^/]+)\.json$} or die 'invalid receipt basename';
    my $name = $1;
    die "raw digest mismatch $path\n" unless sha256_hex(bytes("$directory/$name.log")) eq $receipt->{raw_log_sha256};
    die "proof digest mismatch $path\n" unless sha256_hex(bytes("$directory/$name-environment-proof.log")) eq $receipt->{environment_proof_sha256};
    printf "PASS %s principal_exit=%d post-helper source/tools/routing stable\n", $path, $receipt->{exit};
}
