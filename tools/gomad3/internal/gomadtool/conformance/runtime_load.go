package conformance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func (campaign *runtimeCampaign) requireAddressPerturbation(packageName string) error {
	var expectedPayload string
	markers := make(map[string]struct{})
	for _, padding := range []string{"0", "1048576", "4194304"} {
		result, err := campaign.command(
			fmt.Sprintf("%s-address-padding-%s", strings.TrimPrefix(packageName, "./"), padding),
			[]string{campaign.config.Go, "run", "-exec", campaign.execWrapper, packageName, "-gomad-address-padding=" + padding}, campaign.testdata, time.Minute,
			[]string{"GOMADSEED", "CGO_ENABLED", "GOGC", "TZ", "GOMAD3_CHILD_SEED"},
			"CGO_ENABLED=0", "GOGC=off", "TZ=UTC", "GOMAD3_CHILD_SEED=1",
		)
		if err != nil {
			return err
		}
		lines := strings.Split(commandOutput(result), "\n")
		var marker string
		var payload []string
		for _, line := range lines {
			if strings.HasPrefix(line, "GOMAD3_ADDRESS ") {
				if marker != "" {
					return fmt.Errorf("%s address perturbation emitted no unique marker for padding %s", packageName, padding)
				}
				marker = strings.TrimPrefix(line, "GOMAD3_ADDRESS ")
				continue
			}
			payload = append(payload, line)
		}
		if marker == "" || lines[len(lines)-1] != "GOMAD3_ADDRESS "+marker {
			return fmt.Errorf("%s address perturbation emitted its marker before completing output for padding %s", packageName, padding)
		}
		if _, err := strconv.ParseUint(strings.TrimPrefix(marker, "0x"), 16, 64); err != nil || !strings.HasPrefix(marker, "0x") {
			return fmt.Errorf("%s address perturbation emitted invalid marker for padding %s: %s", packageName, padding, marker)
		}
		markers[marker] = struct{}{}
		joined := strings.Join(payload, "\n")
		if expectedPayload == "" {
			expectedPayload = joined
		} else if joined != expectedPayload {
			return fmt.Errorf("%s output changed under address perturbation for padding %s", packageName, padding)
		}
	}
	if len(markers) <= 1 {
		return fmt.Errorf("%s address perturbation did not produce distinct layouts", packageName)
	}
	return nil
}

func (campaign *runtimeCampaign) requireHostLoad(binaries map[string]string) (returnedErr error) {
	fixtures := []struct{ label, binary string }{
		{label: "scheduler", binary: binaries["scheduler"]}, {label: "maps", binary: binaries["maps"]},
		{label: "select", binary: binaries["select"]}, {label: "channels", binary: binaries["channels"]},
		{label: "sync", binary: binaries["sync"]}, {label: "runqueue", binary: binaries["runqueue"]},
		{label: "automatic-gc", binary: binaries["automatic-gc"]},
	}
	expected := make([]string, len(fixtures))
	for index, fixture := range fixtures {
		result, err := campaign.hostFixture(fixture.label+"-baseline", fixture.binary, 0)
		if err != nil {
			return err
		}
		expected[index] = commandOutput(result)
	}
	stop, err := startCPULoadWorkers(2)
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, stop()) }()
	for iteration := 1; iteration <= 8; iteration++ {
		for index, fixture := range fixtures {
			result, err := campaign.hostFixture(fixture.label+"-host-load", fixture.binary, iteration)
			if err != nil {
				return err
			}
			if commandOutput(result) != expected[index] {
				return fmt.Errorf("%s output changed under unrelated host CPU load on run %d", fixture.label, iteration)
			}
		}
	}
	return stop()
}

func (campaign *runtimeCampaign) hostFixture(label, binary string, iteration int) (hostexec.Result, error) {
	return campaign.command(
		fmt.Sprintf("%s-%d", label, iteration), []string{binary}, campaign.testdata, 10*time.Second,
		[]string{"GODEBUG", "GOMAXPROCS", "GOMADSEED"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1", "GOMADSEED=1",
	)
}

var mapFamilyLabels = []string{
	"create", "string", "clone", "delete-reinsert", "clear", "uint8", "uint16", "uint32", "uint64",
	"float32", "float64", "complex64", "complex128", "empty-interface", "non-empty-interface", "array", "struct",
	"growth", "small", "nan",
}

func (campaign *runtimeCampaign) requireMapFamilies() error {
	diversity := make(map[string]map[string]struct{}, len(mapFamilyLabels))
	for _, family := range mapFamilyLabels {
		diversity[family] = make(map[string]struct{})
	}
	for seed := range 32 {
		output, err := campaign.runEnabled(strconv.Itoa(seed), "./maps", "map-families", seed)
		if err != nil {
			return err
		}
		for _, family := range mapFamilyLabels {
			prefix := family + ":"
			line, count := prefixedLine(output, prefix)
			if count == 0 {
				return fmt.Errorf("map hashing audit is missing family %s", family)
			}
			if count != 1 {
				return fmt.Errorf("map hashing audit emitted family %s more than once", family)
			}
			diversity[family][line] = struct{}{}
		}
	}
	for _, family := range mapFamilyLabels {
		if len(diversity[family]) <= 1 {
			return fmt.Errorf("different seeds produced no map diversity for family %s", family)
		}
	}
	return nil
}
