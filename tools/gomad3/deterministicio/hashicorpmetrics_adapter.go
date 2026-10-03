package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	hashicorpMetricsModulePath                       = "github.com/hashicorp/go-metrics"
	hashicorpMetricsVersion                          = "v0.5.4"
	hashicorpMetricsSum                              = "h1:8mmPiIJkTPPEbAiV97IxdAGNdRdaWwVap1BU6elejKY="
	hashicorpMetricsOriginalSourceInventorySHA256    = "sha256:0ec676c8563b5c5e59c090a2bbe8c59cdf4747af300271f2c7c7f8772925beb6"
	hashicorpMetricsReplacementSourceInventorySHA256 = "sha256:3514f7ccde2c9fbc0b888b359931c3be62498a97f238011ce2dbc8913ea57ef2"
	hashicorpMetricsSignalPath                       = "inmem_signal.go"
	hashicorpMetricsSignalSourceSHA256               = "sha256:49eaf463a89e14809024f7ae01ce9fac1203cbf7c2cbffb47d146321197c59b7"
	hashicorpMetricsSignalReplacementSHA256          = "sha256:8526e7c6ec09e38d694cca294d7cba194a0fe25491aa57804184fa9184a96173"
)

var hashicorpMetricsPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:040cd95bb611568b5a7c0110726ca135f16e6a900175cb8f38b49f9248d70217",
	"linux/amd64":  "sha256:040cd95bb611568b5a7c0110726ca135f16e6a900175cb8f38b49f9248d70217",
}

var hashicorpMetricsPreparedSourceSetSHA256 = hostPin(hashicorpMetricsPreparedSourceSetSHA256ByHost)

// hashicorpMetricsRewrites refuse signal services without changing their
// public syscall.Signal signatures, metrics emission, or dump formatting.
// The pointer-only constructors cannot return an unsupported-service error.
var hashicorpMetricsRewrites = []sourceRewrite{
	{
		path:              hashicorpMetricsSignalPath,
		sourceSHA256:      hashicorpMetricsSignalSourceSHA256,
		replacementSHA256: hashicorpMetricsSignalReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("\t\"os/signal\"\n"), replacement: nil},
			{
				anchor:      []byte("// and dumps the current metrics out to a writer\n"),
				replacement: []byte("// and dumps the current metrics out to a writer\n// Deterministic builds refuse this service because host signals are unsupported.\n"),
			},
			{
				anchor: []byte("func NewInmemSignal(inmem *InmemSink, sig syscall.Signal, w io.Writer) *InmemSignal {\n" +
					"\ti := &InmemSignal{\n\t\tsignal: sig,\n\t\tinm:    inmem,\n\t\tw:      w,\n" +
					"\t\tsigCh:  make(chan os.Signal, 1),\n\t\tstopCh: make(chan struct{}),\n\t}\n" +
					"\tsignal.Notify(i.sigCh, sig)\n\tgo i.run()\n\treturn i\n}\n"),
				replacement: []byte("func NewInmemSignal(inmem *InmemSink, sig syscall.Signal, w io.Writer) *InmemSignal {\n" +
					"\tpanic(\"gomad: in-memory metrics signal service is unsupported\")\n}\n"),
			},
			{anchor: []byte("\tsignal.Stop(i.sigCh)\n"), replacement: nil},
		},
	},
}

var hashicorpMetricsAdapter = rewrittenModule{
	module: hashicorpMetricsModulePath, version: hashicorpMetricsVersion, sum: hashicorpMetricsSum,
	cacheElements:                 []string{"github.com", "hashicorp", "go-metrics@" + hashicorpMetricsVersion},
	replacementDirectory:          "hashicorp-go-metrics",
	originalInventorySHA256:       hashicorpMetricsOriginalSourceInventorySHA256,
	replacementInventorySHA256:    hashicorpMetricsReplacementSourceInventorySHA256,
	preparedPackage:               hashicorpMetricsModulePath,
	preparedSourceSetSHA256ByHost: hashicorpMetricsPreparedSourceSetSHA256ByHost,
	rewrites:                      hashicorpMetricsRewrites,
}

func prepareHashicorpMetrics(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, hashicorpMetricsAdapter)
}
