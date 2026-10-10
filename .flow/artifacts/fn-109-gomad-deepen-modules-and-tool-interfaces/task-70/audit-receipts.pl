use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
use JSON::PP qw(decode_json);
my $packet = shift @ARGV;
sub bytes {
    my ($path) = @_;
    open my $file, '<', $path or die "$path: $!";
    binmode $file;
    my $data = do { local $/; <$file> };
    close $file or die $!;
    return $data;
}
my %expected = (
    'baseline-three' => 1, 'baseline-controls' => 0, 'baseline-runner-lint' => 1,
    'final-three' => 0, 'final-focused' => 0, 'final-preservation' => 0,
    'final-outcome-comparison' => 0, 'final-format' => 0, 'final-diff-check' => 0,
    'final-host-vet' => 0, 'final-errortype' => 0,
    'final-supported-darwin-vet' => 0, 'final-supported-linux-vet' => 0,
    'final-boundaries' => 0, 'final-validate' => 0, 'final-fast-lint' => 0,
    'final-runner-lint' => 1, 'final-lint-comparison' => 0,
);
my %seen;
for my $path (grep { /\.json\z/ } @ARGV) {
    my ($name) = $path =~ m{/([^/]+)\.json\z};
    die "unexpected receipt $path" unless exists $expected{$name};
    die "duplicate receipt $name" if $seen{$name}++;
    my $receipt = decode_json(bytes($path));
    die "$name wrong exit/stability" unless $receipt->{exit} == $expected{$name} && $receipt->{pre_post_match_exit} == 0;
    die "$name wrong workspace/head" unless $receipt->{cwd} eq '/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence' && $receipt->{head} eq 'fd9cfc4026db966a877597a9658a0af589f18aea';
    for my $pair ([source_manifest_sha256 => 'manifest'], [post_source_manifest_sha256 => 'manifest'], [tools_manifest_sha256 => 'tools_manifest'], [post_tools_manifest_sha256 => 'tools_manifest'], [environment_sha256 => 'environment_manifest'], [post_environment_sha256 => 'environment_manifest']) {
        my ($field, $prefix) = @$pair;
        die "$name $field digest differs" unless sha256_hex(bytes("$packet/$prefix-$receipt->{$field}")) eq $receipt->{$field};
    }
    die "$name raw digest differs" unless sha256_hex(bytes("$packet/$name.log")) eq $receipt->{raw_log_sha256};
    die "$name proof digest differs" unless sha256_hex(bytes("$packet/$name-environment-proof.log")) eq $receipt->{environment_proof_sha256};
    die "$name wrapper changed" unless sha256_hex(bytes("$packet/run-control.sh")) eq $receipt->{wrapper_sha256} && $receipt->{wrapper_sha256} eq $receipt->{post_wrapper_sha256};
    printf "%s exit=%d elapsed_seconds=%d stability=0 receipt_sha256=%s raw_sha256=%s\n", $name, $receipt->{exit}, $receipt->{elapsed_seconds}, sha256_hex(bytes($path)), $receipt->{raw_log_sha256};
}
die 'receipt set differs' unless keys(%seen) == keys(%expected);
my $validation = decode_json(bytes("$packet/final-validate.json"));
my $manifest = bytes("$packet/manifest-$validation->{source_manifest_sha256}");
my @tests = $manifest =~ /^[0-9a-f]{64}  (tests\/[^\n]+)$/mg;
my @top_tests = grep { m{^tests/[^/]+_test\.go\z} } @tests;
die 'full materialized tests inventory differs' unless @tests == 132 && @top_tests == 113;
for my $path (@tests) {
    my $hash = sha256_hex(bytes($path));
    die "materialized test changed: $path" unless $manifest =~ /^\Q$hash  $path\E$/m;
}
printf "PASS: first validation captured all %d materialized test files including %d top-level test files; every file unchanged\n", scalar(@tests), scalar(@top_tests);
for my $name (qw(baseline-runner-lint final-runner-lint final-fast-lint)) {
    my $log = bytes("$packet/$name.log");
    my $warnings = () = $log =~ /^level=warning /mg;
    my $missing = () = $log =~ /^find: .*No such file or directory/mg;
    printf "%s cache_warning_count=%d missing_sparse_directory_warning_count=%d\n", $name, $warnings, $missing;
}
die 'gofmt reported a diff' unless bytes("$packet/final-format.log") eq '';
print "PASS: 18 numerical observations and immutable raw/source/tool/environment/wrapper bindings validated; no receipt overwritten\n";
