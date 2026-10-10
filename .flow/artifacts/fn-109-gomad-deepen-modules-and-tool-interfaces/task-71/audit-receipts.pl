use strict;
use warnings;
use JSON::PP qw(decode_json);
use Digest::SHA qw(sha256_hex);
my $packet = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-71';
my %inputs = map { $_ => 1 } @ARGV;
sub read_bound {
    my ($path) = @_;
    die "unbound audit input $path" unless $inputs{$path};
    open my $file, '<', $path or die "$path: $!";
    binmode $file;
    my $bytes = do { local $/; <$file> };
    close $file or die $!;
    return $bytes;
}
my @names = qw(baseline-three baseline-controls baseline-runner-lint baseline-runner-lint-retained-environment-proof baseline-controls-current-environment-proof baseline-runner-lint-current-environment-proof final-three final-focused final-preservation final-format final-diff-check final-boundaries final-host-vet final-errortype final-supported-darwin-vet final-supported-linux-vet final-runner-lint final-fast-lint final-validate final-outcome-comparison final-lint-comparison);
my %red = map { $_ => 1 } qw(baseline-three baseline-runner-lint final-runner-lint);
my %historical_gap = map { $_ => 1 } qw(baseline-three baseline-controls baseline-runner-lint);
my @rows;
for my $name (@names) {
    my $receipt = decode_json(read_bound("$packet/$name.json"));
    die "$name cwd/head mismatch" unless $receipt->{cwd} eq '/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/completion' && $receipt->{head} eq '30a8847362779670311486706701233d0043be07';
    die "$name unexpected exit" unless $receipt->{exit} == ($red{$name} ? 1 : 0);
    die "$name stability/proof failed" unless $receipt->{pre_post_match_exit} == 0 && $receipt->{environment_proof_exit} == 0;
    die "$name timeout identity" unless $receipt->{external_timeout_seconds} == 900 && $receipt->{kill_after_seconds} == 15;
    for my $key (qw(source tools routing)) {
        my $value = $receipt->{$key . '_manifest_sha256'};
        for my $prefix (qw(post_ after_environment_ post_environment_ post_helper_)) {
            die "$name $prefix$key drift" unless $receipt->{$prefix . $key . '_manifest_sha256'} eq $value;
        }
        die "$name $key manifest hash" unless sha256_hex(read_bound("$packet/" . $key . "_manifest-$value")) eq $value;
    }
    die "$name wrapper drift" unless $receipt->{wrapper_sha256} eq $receipt->{post_wrapper_sha256};
    my $wrapper = $historical_gap{$name} ? 'run-control.sh' : ($name =~ /^baseline-/ ? 'run-control-v2.sh' : 'run-control-v3.sh');
    die "$name actual wrapper hash" unless sha256_hex(read_bound("$packet/$wrapper")) eq $receipt->{wrapper_sha256};
    die "$name raw log hash" unless sha256_hex(read_bound("$packet/$name.log")) eq $receipt->{raw_log_sha256};
    die "$name environment proof hash" unless sha256_hex(read_bound("$packet/$name-environment-proof.log")) eq $receipt->{environment_proof_sha256};
    my @normalized;
    for my $key (qw(environment_sha256 post_environment_sha256)) {
        my $hash = $receipt->{$key};
        my $bytes = read_bound("$packet/environment_manifest-$hash");
        die "$name raw environment hash" unless sha256_hex($bytes) eq $hash;
        my $count = $bytes =~ s{^(\s*"GOGCCFLAGS": "[^"\n]*)/go-build[0-9]+=/tmp/go-build}{$1/go-build<ephemeral>=/tmp/go-build}mg;
        die "$name normalization count" unless $count == 1;
        push @normalized, $bytes;
    }
    die "$name environment changes outside ephemeral path" unless $normalized[0] eq $normalized[1];
    unless ($historical_gap{$name}) {
        die "$name proof-input stability failed" unless $receipt->{environment_proof_inputs_match_exit} == 0;
        for my $phase (qw(before after)) {
            my $hash = $receipt->{'environment_proof_inputs_' . $phase . '_sha256'};
            die "$name proof input manifest hash" unless sha256_hex(read_bound("$packet/$name-environment-inputs-$phase.sha256")) eq $hash;
        }
        die "$name proof input manifest changed" unless $receipt->{environment_proof_inputs_before_sha256} eq $receipt->{environment_proof_inputs_after_sha256};
    }
    my $inventory = $receipt->{test_inventory_sha256};
    die "$name inventory drift" unless $inventory eq $receipt->{post_test_inventory_sha256};
    my $inventory_bytes = read_bound("$packet/test_inventory-$inventory");
    die "$name inventory hash" unless sha256_hex($inventory_bytes) eq $inventory;
    my @tests = $inventory_bytes =~ /^[a-f0-9]{64}  (tests\/.+)$/mg;
    die "$name incomplete tests materialization" unless @tests == 132 && $inventory_bytes =~ /^top-level-test-files=113$/m;
    push @rows, {name=>$name, exit=>0+$receipt->{exit}, elapsed_seconds=>0+$receipt->{elapsed_seconds}, original_helper_raw_input_window_missing=>($historical_gap{$name} ? JSON::PP::true : JSON::PP::false)};
}
my $tools = read_bound("$packet/tools_manifest-38051db81a79ef0a7b10d46eb6c5398d704f69ffb4827b088008a840bfde4670");
my %tools = map { $_ => 1 } $tools =~ /^[a-f0-9]{64}  (.+)$/mg;
my $source = read_bound("$packet/source_manifest-" . decode_json(read_bound("$packet/final-validate.json"))->{source_manifest_sha256});
my $present = () = $source =~ /^[a-f0-9]{64}  /mg;
my $absent = () = $source =~ /^ABSENT /mg;
my $fast = read_bound("$packet/final-fast-lint.log");
my %warnings;
for my $name (qw(baseline-runner-lint final-runner-lint final-fast-lint)) {
    my $raw = read_bound("$packet/$name.log");
    $warnings{$name} = {cache_enospc_lines=>scalar(() = $raw =~ /^.*no space left on device.*$/mg), missing_sparse_find_lines=>scalar(() = $raw =~ /^find: .*No such file or directory$/mg)};
}
die 'fast lint package/diff observations missing' unless $fast =~ /Lint module tools\/gomad3: 55 host packages/ && $fast =~ /diff: 50\/0/ && $fast =~ /^0 issues\.$/m;
die 'gofmt output is nonempty' unless read_bound("$packet/final-format.log") eq '';
my $output = {result=>'PASS', receipts=>\@rows, receipt_count=>scalar @names, raw_environment_pair_count=>scalar @names, raw_environment_observation_count=>2*scalar @names, environment_normalization=>'exactly one GOGCCFLAGS go-build numeric ephemeral mapping; all other bytes identical', baseline_historical_helper_gap_count=>3, current_verification_not_retroactive=>JSON::PP::true, tools_distinct_literal_paths=>scalar keys %tools, tests_materialized=>132, top_level_tests=>113, final_validate_source_present_lines=>$present, final_validate_source_absent_lines=>$absent, warnings=>\%warnings, fast_lint_diff_filter=>'50 unchanged diagnostics filtered to zero; aggregate remains RED'};
print JSON::PP->new->canonical->pretty->encode($output);
