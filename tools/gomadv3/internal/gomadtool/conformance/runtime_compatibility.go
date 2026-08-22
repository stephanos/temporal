package conformance

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

func (campaign *runtimeCampaign) requireStockCompatibility() error {
	wantVersion := gomadversion.GoVersion
	stockGo := os.Getenv("GOMADV3_STOCK_GO")
	if stockGo == "" {
		launcher, err := exec.LookPath("go")
		if err == nil {
			root, commandErr := campaign.command(
				"stock-go-resolve", []string{launcher, "env", "GOROOT"}, campaign.testdata, 10*time.Second,
				[]string{"GOMADSEED", "GONOPROXY", "GONOSUMDB", "GOPRIVATE", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN"}, "GOTOOLCHAIN="+wantVersion,
			)
			if commandErr == nil {
				stockGo = filepath.Join(commandOutput(root), "bin", "go")
			}
		}
	}
	info, err := os.Stat(stockGo)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.Join(fmt.Errorf("stock Go is missing; set GOMADV3_STOCK_GO to a %s executable", wantVersion), err)
	}
	versionResult, err := campaign.command(
		"stock-go-version", []string{stockGo, "version"}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOTOOLCHAIN"}, "GOTOOLCHAIN=local",
	)
	if err != nil {
		return err
	}
	if actual := commandOutput(versionResult); !strings.HasPrefix(actual, "go version "+wantVersion+" ") {
		return fmt.Errorf("stock Go must report %s; %s reported: %s", wantVersion, stockGo, actual)
	}
	rootResult, err := campaign.command(
		"stock-go-goroot", []string{stockGo, "env", "GOROOT"}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOTOOLCHAIN"}, "GOTOOLCHAIN=local",
	)
	if err != nil {
		return err
	}
	stockRoot := commandOutput(rootResult)
	if info, err := os.Stat(stockRoot); err != nil || !info.IsDir() {
		return errors.Join(fmt.Errorf("stock Go reported an invalid GOROOT: %s", stockRoot), err)
	}
	canonicalStock, err := filepath.EvalSymlinks(stockRoot)
	if err != nil {
		return err
	}
	canonicalCustom, err := filepath.EvalSymlinks(campaign.goRoot)
	if err != nil {
		return err
	}
	if canonicalStock == canonicalCustom {
		return fmt.Errorf("stock Go resolves to the gomadv3 custom GOROOT: %s", canonicalStock)
	}
	customRun, err := campaign.command(
		"activation-custom-compatibility", []string{campaign.config.Go, "run", "./activation"}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	if err != nil {
		return err
	}
	stockRun, err := campaign.command(
		"activation-stock-compatibility", []string{stockGo, "run", "./activation"}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	if err != nil {
		return err
	}
	if commandOutput(customRun) != commandOutput(stockRun) {
		return fmt.Errorf("disabled go run output differs from stock %s", wantVersion)
	}
	customTest, err := campaign.compatibilityGoTest(campaign.config.Go, "custom")
	if err != nil {
		return err
	}
	stockTest, err := campaign.compatibilityGoTest(stockGo, "stock")
	if err != nil {
		return err
	}
	customLine, customCount := prefixedLine(customTest, "GOMADV3_COMPAT ")
	stockLine, stockCount := prefixedLine(stockTest, "GOMADV3_COMPAT ")
	if customCount != 1 || stockCount != 1 {
		return fmt.Errorf("disabled go test compatibility output must appear exactly once: custom=%q stock=%q", customLine, stockLine)
	}
	if customLine != stockLine {
		return fmt.Errorf("disabled go test output differs from stock %s", wantVersion)
	}
	customBenchmark, err := campaign.disabledClockBenchmark(campaign.config.Go, "custom")
	if err != nil {
		return err
	}
	stockBenchmark, err := campaign.disabledClockBenchmark(stockGo, "stock")
	if err != nil {
		return err
	}
	customNS, err := benchmarkMedianNS(customBenchmark)
	if err != nil {
		return err
	}
	stockNS, err := benchmarkMedianNS(stockBenchmark)
	if err != nil {
		return err
	}
	limit := max(stockNS*2, stockNS+10)
	if customNS > limit {
		return fmt.Errorf("disabled clock read regression: custom median %v ns/op, stock median %v ns/op", customNS, stockNS)
	}
	return nil
}

func (campaign *runtimeCampaign) compatibilityGoTest(goCommand, implementation string) (string, error) {
	result, err := campaign.command(
		"gotest-"+implementation+"-compatibility",
		[]string{goCommand, "test", "-count=1", "-tags=test_dep", "-run", "^TestDisabledCompatibility$", "-v", "./gotest"},
		campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	return commandOutput(result), err
}

func prefixedLine(output, prefix string) (string, int) {
	var found string
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			found = line
			count++
		}
	}
	return found, count
}

func (campaign *runtimeCampaign) disabledClockBenchmark(goCommand, implementation string) (string, error) {
	var combined strings.Builder
	for iteration := range 2 {
		result, err := campaign.command(
			fmt.Sprintf("clock-bench-%s-%d", implementation, iteration),
			[]string{goCommand, "test", "-run", "^$", "-bench", "^BenchmarkDisabledClockNow$", "-benchtime=250ms", "-count=7", "-cpu=1", "./clock_bench"},
			campaign.testdata, time.Minute,
			[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
			"GODEBUG=", "GOMAXPROCS=1", "GOTOOLCHAIN=local", "GOWORK=off",
		)
		if err != nil {
			return "", err
		}
		combined.Write(result.Stdout.RawBytes)
	}
	return combined.String(), nil
}

func (campaign *runtimeCampaign) runClockDirect(seed, binary, mode string, iteration int) (string, error) {
	result, err := campaign.command(
		fmt.Sprintf("clock-%s-seed-%s-%d", mode, seed, iteration), []string{binary, mode}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED="+seed, "TZ=UTC",
	)
	return commandOutput(result), err
}
