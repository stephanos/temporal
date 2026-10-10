use strict;
use warnings;
use JSON::PP qw(decode_json);
use Digest::SHA qw(sha256_hex);
use File::Find qw(find);
sub raw {
    my ($path) = @_;
    open my $file, '<', $path or die "$path: $!";
    my $value = do { local $/; <$file> };
    close $file or die $!;
    return $value;
}
my ($verifier, $proof_path, $validate_path, $fast_path, $old_tools_path, $old_validate_path, $old_fast_path) = splice @ARGV, 0, 7;
die 'expected explicit evidence inputs' unless defined $old_fast_path;
{
    local @ARGV = ($proof_path, $validate_path, $fast_path);
    my $result = do $verifier;
    die $@ if $@;
    die $! unless defined $result;
}
my @receipts = map { decode_json(raw($_)) } ($proof_path, $validate_path, $fast_path);
$validate_path =~ m{^(.*)/[^/]+$} or die 'missing packet directory';
my $packet = $1;
my %tools;
for my $line (split /\n/, raw("$packet/tools_manifest-$receipts[0]{tools_manifest_sha256}")) {
    $line =~ /^([a-f0-9]{64})  (.+)$/ or die 'invalid tools entry';
    $tools{$2} = $1;
}
die 'expected 30 measured significant executable identities' unless keys(%tools) == 30;
my $prior_count = 0;
for my $line (split /\n/, raw($old_tools_path)) {
    $line =~ /^([a-f0-9]{64})  (.+)$/ or die 'invalid original tools entry';
    die "original tool changed $2" unless $tools{$2} eq $1;
    $prior_count++;
}
die 'expected 26 preserved prior tools' unless $prior_count == 26;
die 'shell target bytes differ' unless $tools{'/bin/sh'} eq $tools{'/usr/bin/dash'};
for my $tool (qw(/usr/bin/rm /usr/bin/grep)) { die "missing $tool" unless exists $tools{$tool}; }
my $routing = raw("$packet/routing_manifest-$receipts[0]{routing_manifest_sha256}");
die 'Make shell route unexpected' unless $routing =~ /^Make SHELL literal=\/bin\/sh link=dash resolved=\/usr\/bin\/dash$/m;
for my $pair ([$receipts[1], $old_validate_path], [$receipts[2], $old_fast_path]) {
    my @current = @{$pair->[0]{argv}};
    die 'SHELL override missing from actual Make argv' unless pop(@current) eq 'SHELL=/bin/sh';
    my $original = decode_json(raw($pair->[1]));
    die 'Make argv changed beyond SHELL addition' unless JSON::PP->new->canonical->encode(\@current) eq JSON::PP->new->canonical->encode($original->{argv});
}
my %tests;
for my $line (split /\n/, raw("$packet/source_manifest-$receipts[1]{source_manifest_sha256}")) {
    if ($line =~ /^([a-f0-9]{64})  (tests\/.+)$/) { $tests{$2} = $1; }
}
my @actual;
find({wanted => sub { push @actual, $File::Find::name if -f $_; }, no_chdir => 1}, 'tests');
die 'incomplete first validation materialized tests inventory' unless @actual == 132 && keys(%tests) == 132;
my $top = 0;
for my $path (@actual) {
    die "test input absent or changed $path" unless exists $tests{$path} && sha256_hex(raw($path)) eq $tests{$path};
    $top++ if $path =~ m{^tests/[^/]+_test\.go$};
}
die 'top-level test inventory changed' unless $top == 113;
my $fast = raw("$packet/current-fast-lint.log");
my $warnings = () = $fast =~ /^level=warning /mg;
my $missing = () = $fast =~ /^find: /mg;
die 'fast lint observed wrong surface' unless $fast =~ /Lint module tools\/gomad3: 55 host packages/ && $fast =~ /diff: 50\/0/ && $fast =~ /^0 issues\.$/m;
my $proof = raw("$packet/retained-environment-proof.log");
my $pairs = () = $proof =~ /^PASS .* every remaining byte and GOGCCFLAGS option equal$/mg;
my $captures = () = $proof =~ / replacements=1$/mg;
die 'retained proof collected wrong inputs' unless $pairs == 19 && $captures == 38;
for my $receipt (@receipts) { die 'principal gate failed' unless $receipt->{exit} == 0; }
printf "PASS original tools=%d successor tools=%d; /bin/sh=dash and rm/grep measured\n", $prior_count, scalar keys %tools;
printf "PASS first validation materialized tests=%d top-level tests=%d all hashes current\n", scalar @actual, $top;
printf "PASS exact Make argv changed only by explicit SHELL=/bin/sh; default lint-cache warnings=%d sparse-find warnings=%d; diff-filtered 55 packages, not aggregate GREEN\n", $warnings, $missing;
printf "PASS retained environment proof: actual pairs=%d captures=%d; historical limitations remain\n", $pairs, $captures;
