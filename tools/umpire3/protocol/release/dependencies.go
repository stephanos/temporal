package release

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"go.temporal.io/server/tools/umpire3/protocol/catalog"
	"go.temporal.io/server/tools/umpire3/protocol/checker"
	"go.temporal.io/server/tools/umpire3/protocol/experiment"
	"go.temporal.io/server/tools/umpire3/protocol/internal/codec"
	"go.temporal.io/server/tools/umpire3/protocol/monitor"
)

const (
	DefaultDecodeLimit = codec.DefaultDecodeLimit
	FormatVersion      = catalog.FormatVersion
)

type ResultClass = catalog.ResultClass
type TrustBadge = catalog.TrustBadge
type Experiment = experiment.Experiment

func DefaultCatalog() (catalog.Catalog, error) {
	return catalog.DefaultCatalog()
}

func DefaultComposition() (catalog.Composition, error) {
	return catalog.DefaultComposition()
}

func DefaultParityLedger() (catalog.ParityLedger, error) {
	return catalog.DefaultParityLedger()
}

func DefaultMonitorCatalog() (monitor.MonitorCatalog, error) {
	return monitor.DefaultMonitorCatalog()
}

func DefaultCheckerCoverage() (checker.CheckerCoverageManifest, error) {
	return checker.DefaultCheckerCoverage()
}

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	return codec.DecodeStrictJSON(reader, limit, kind, destination)
}

func validHash(value string) bool {
	return codec.ValidHash(value)
}

func stringCompare(left, right string) int {
	return strings.Compare(left, right)
}

func digestBytes(value []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(value))
}
