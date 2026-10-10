use strict;
use warnings;
use JSON::PP qw(decode_json);

die 'expected four raw observations' unless @ARGV == 4;
my @observations;
for my $path (@ARGV) {
    open my $file, '<', $path or die $!;
    my (%outcomes, %output);
    while (my $line = <$file>) {
        my $event = decode_json($line);
        next unless defined $event->{Test};
        my $name = $event->{Test};
        $output{$name} .= $event->{Output} if $event->{Action} eq 'output';
        next unless $event->{Action} =~ /^(pass|fail|skip)$/;
        die "duplicate terminal name $name" if exists $outcomes{$name};
        $outcomes{$name} = $event->{Action};
    }
    close $file or die $!;
    push @observations, [\%outcomes, \%output];
}
my @originals = qw(TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts);
my ($before, $controls, $final, $focused) = map { $_->[0] } @observations;
die 'selected-original count differs' unless keys(%$before) == 2 && keys(%$final) == 2;
for my $name (@originals) {
    die "$name baseline not meaningful preparation RED" unless $before->{$name} eq 'fail' && $observations[0][1]{$name} =~ /deterministic I\/O requires one of darwin\/arm64, linux\/amd64; host is linux\/arm64/;
    die "$name final did not pass" unless $final->{$name} eq 'pass' && $focused->{$name} eq 'pass';
}
for my $name (keys %$controls) {
    die "$name control outcome changed" unless exists $focused->{$name} && $controls->{$name} eq $focused->{$name} && $controls->{$name} eq 'pass';
}
my %allowed = map { $_ => 1 } (keys(%$controls), @originals);
die 'unadmitted focused name' if grep { !$allowed{$_} } keys %$focused;
die 'focused count differs' unless keys(%$focused) == keys(%$controls) + 2;
printf "PASS: both original preparation failures become passes; %d actual terminal control names unchanged; %d final actual terminal named passes; zero fail/skip; no invented slash names\n", scalar(keys %$controls), scalar(keys %$focused);
