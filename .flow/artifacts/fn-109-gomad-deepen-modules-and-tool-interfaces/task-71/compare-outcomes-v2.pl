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
        my $name = "$event->{Package}:$event->{Test}";
        $output{$name} .= $event->{Output} if $event->{Action} eq 'output';
        next unless $event->{Action} =~ /^(pass|fail|skip)$/;
        die "duplicate terminal $name" if exists $outcomes{$name};
        $outcomes{$name} = $event->{Action};
    }
    close $file or die $!;
    push @observations, [\%outcomes, \%output];
}
my ($before, $controls, $final, $focused) = map { $_->[0] } @observations;
die 'selected original emitted count differs' unless keys(%$before) == 60 && keys(%$final) == 60;
for my $name (sort keys %$before) {
    die "$name baseline not fail" unless $before->{$name} eq 'fail';
    if ($name =~ m{:[^/]+/}) {
        die "$name preparation refusal missing" unless $observations[0][1]{$name} =~ /preparation\.stageError\{stage:"validation"|deterministic I\/O requires one of darwin\/arm64, linux\/amd64; host is linux\/arm64/;
    }
    die "$name final not pass" unless $final->{$name} eq 'pass' && $focused->{$name} eq 'pass';
}
for my $name (sort keys %$controls) {
    die "$name control changed" unless exists $focused->{$name} && $controls->{$name} eq 'pass' && $focused->{$name} eq 'pass';
}
my %allowed = map { $_ => 1 } (keys(%$controls), keys(%$before));
die 'focused emitted domain differs' unless keys(%$focused) == keys(%allowed) && !grep { !$allowed{$_} } keys %$focused;
printf "PASS: 60 actual original terminal outcomes RED to GREEN; %d control terminal outcomes unchanged; %d final terminal passes; zero fail/skip\n", scalar(keys %$controls), scalar(keys %$focused);
for my $name (sort keys %$focused) { printf "%s %s\n", $focused->{$name}, $name; }
