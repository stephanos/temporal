package deterministicio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type RegeneratedSource struct {
	Path                 string `json:"path"`
	OldSourceSHA256      string `json:"old_source_sha256"`
	SourceSHA256         string `json:"source_sha256"`
	OldReplacementSHA256 string `json:"old_replacement_sha256"`
	ReplacementSHA256    string `json:"replacement_sha256"`
	Source               []byte `json:"source"`
	Replacement          []byte `json:"replacement"`
}

type AdapterRegeneration struct {
	Module                        string              `json:"module"`
	SourceFile                    string              `json:"source_file"`
	Version                       string              `json:"version"`
	Sum                           string              `json:"sum"`
	OldOriginalInventorySHA256    string              `json:"old_original_inventory_sha256"`
	OriginalInventorySHA256       string              `json:"original_inventory_sha256"`
	OldReplacementInventorySHA256 string              `json:"old_replacement_inventory_sha256"`
	ReplacementInventorySHA256    string              `json:"replacement_inventory_sha256"`
	PreparedSourceSets            map[string]string   `json:"prepared_source_sets"`
	Sources                       []RegeneratedSource `json:"sources"`
	ReplacementRoot               string              `json:"-"`
	PreparedPackage               string              `json:"-"`
}

func RegenerateAdapter(module, version, sum, moduleRoot, workRoot string) (AdapterRegeneration, error) {
	if version == "" || sum == "" || moduleRoot == "" || workRoot == "" {
		return AdapterRegeneration{}, errors.New("adapter regeneration requires an exact version, sum, module source, and work root")
	}
	var spec rewrittenModule
	switch module {
	case sprigModulePath:
		spec = rewrittenModule{module: module, version: sprigVersion, sum: sprigSum, replacementDirectory: "sprig", originalInventorySHA256: sprigOriginalSourceInventorySHA256, replacementInventorySHA256: sprigReplacementSourceInventorySHA256, preparedPackage: sprigModulePath, rewrites: sprigRewrites}
	case validatorModulePath:
		spec = rewrittenModule{module: module, version: validatorVersion, sum: validatorSum, replacementDirectory: "validator", originalInventorySHA256: validatorOriginalSourceInventorySHA256, replacementInventorySHA256: validatorReplacementSourceInventorySHA256, preparedPackage: validatorModulePath, rewrites: validatorRewrites}
	case pebbleModulePath:
		spec = rewrittenModule{module: module, version: pebbleVersion, sum: pebbleSum, replacementDirectory: "pebble", originalInventorySHA256: pebbleOriginalSourceInventorySHA256, replacementInventorySHA256: pebbleReplacementSourceInventorySHA256, preparedPackage: pebbleModulePath + "/vfs", rewrites: pebbleRewrites}
	case cactusStatsDModulePath:
		spec = rewrittenModule{module: module, version: cactusStatsDVersion, sum: cactusStatsDSum, replacementDirectory: "cactus-statsd", originalInventorySHA256: cactusStatsDOriginalSourceInventorySHA256, replacementInventorySHA256: cactusStatsDReplacementSourceInventorySHA256, preparedPackage: cactusStatsDModulePath + "/statsd", rewrites: cactusStatsDRewrites}
	case memberlistModulePath:
		spec = rewrittenModule{module: module, version: memberlistVersion, sum: memberlistSum, replacementDirectory: "memberlist", originalInventorySHA256: memberlistOriginalSourceInventorySHA256, replacementInventorySHA256: memberlistReplacementSourceInventorySHA256, preparedPackage: memberlistModulePath, rewrites: memberlistRewrites}
	case sentryModulePath:
		spec = rewrittenModule{module: module, version: sentryVersion, sum: sentrySum, replacementDirectory: "sentry-go", originalInventorySHA256: sentryOriginalSourceInventorySHA256, replacementInventorySHA256: sentryReplacementSourceInventorySHA256, preparedPackage: sentryModulePath, rewrites: sentryRewrites}
	case hashicorpMetricsModulePath:
		spec = rewrittenModule{module: module, version: hashicorpMetricsVersion, sum: hashicorpMetricsSum, replacementDirectory: "hashicorp-go-metrics", originalInventorySHA256: hashicorpMetricsOriginalSourceInventorySHA256, replacementInventorySHA256: hashicorpMetricsReplacementSourceInventorySHA256, preparedPackage: hashicorpMetricsModulePath, rewrites: hashicorpMetricsRewrites}
	case fxModulePath:
		spec = rewrittenModule{module: module, version: fxVersion, sum: fxSum, replacementDirectory: "uber-fx", originalInventorySHA256: fxOriginalSourceInventorySHA256, replacementInventorySHA256: fxReplacementSourceInventorySHA256, preparedPackage: fxModulePath, rewrites: fxRewrites}
	case temporalSDKModulePath:
		spec = rewrittenModule{module: module, version: temporalSDKVersion, sum: temporalSDKSum, replacementDirectory: "temporal-sdk", originalInventorySHA256: temporalSDKOriginalSourceInventorySHA256, replacementInventorySHA256: temporalSDKReplacementSourceInventorySHA256, preparedPackage: temporalSDKModulePath + "/internal", rewrites: temporalSDKRewrites}
	case otelSDKModulePath:
		spec = rewrittenModule{module: module, version: otelSDKVersion, sum: otelSDKSum, replacementDirectory: "otel-sdk", originalInventorySHA256: otelSDKOriginalSourceInventorySHA256, replacementInventorySHA256: otelSDKReplacementSourceInventorySHA256, preparedPackage: otelSDKModulePath + "/resource", rewrites: otelSDKRewrites}
	case sockaddrModulePath:
		spec = rewrittenModule{module: module, version: sockaddrVersion, sum: sockaddrSum, replacementDirectory: "go-sockaddr", originalInventorySHA256: sockaddrOriginalSourceInventorySHA256, replacementInventorySHA256: sockaddrReplacementSourceInventorySHA256, preparedPackage: sockaddrModulePath, rewrites: sockaddrRewrites}
	case memoryModulePath:
		spec = rewrittenModule{module: module, version: memoryVersion, sum: memorySum, replacementDirectory: "modernc-memory", originalInventorySHA256: memoryOriginalSourceInventorySHA256, replacementInventorySHA256: memoryReplacementSourceInventorySHA256, preparedPackage: memoryModulePath, rewrites: memoryRewrites}
	case grpcModulePath:
		return regenerateGRPC(moduleRoot, workRoot, version, sum)
	case xnetModulePath:
		return regenerateXNet(moduleRoot, workRoot, version, sum)
	case libcModulePath:
		return regenerateLibc(moduleRoot, workRoot, version, sum)
	default:
		return AdapterRegeneration{}, fmt.Errorf("adapter %s is not registered", module)
	}
	result, err := regenerateRewrittenModule(moduleRoot, workRoot, spec)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	result.Version, result.Sum = version, sum
	return result, nil
}

func regenerateRewrittenModule(moduleRoot, workRoot string, spec rewrittenModule) (AdapterRegeneration, error) {
	result := AdapterRegeneration{Module: spec.module, SourceFile: adapterSourceFile(spec.module), OldOriginalInventorySHA256: spec.originalInventorySHA256, OldReplacementInventorySHA256: spec.replacementInventorySHA256}
	var err error
	result.OriginalInventorySHA256, err = digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	replacements := make(map[string][]byte, len(spec.rewrites))
	for _, rewrite := range spec.rewrites {
		contents, err := readAdapterSource(spec.module, moduleRoot, rewrite.path)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		rewritten, err := regenerateSourceRewrite(spec.module, rewrite, contents)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		replacements[rewrite.path] = rewritten
		result.Sources = append(result.Sources, RegeneratedSource{Path: rewrite.path, OldSourceSHA256: rewrite.sourceSHA256, SourceSHA256: digestBytes(contents), OldReplacementSHA256: rewrite.replacementSHA256, ReplacementSHA256: digestBytes(rewritten), Source: contents, Replacement: rewritten})
	}
	return finishAdapterRegeneration(moduleRoot, workRoot, spec, result, replacements)
}

func finishAdapterRegeneration(moduleRoot, workRoot string, spec rewrittenModule, result AdapterRegeneration, replacements map[string][]byte) (AdapterRegeneration, error) {
	result.ReplacementRoot = filepath.Join(workRoot, spec.replacementDirectory)
	if err := os.MkdirAll(workRoot, 0o700); err != nil {
		return AdapterRegeneration{}, err
	}
	if err := copyAdapterModule(moduleRoot, result.ReplacementRoot, replacements, defaultAdapterCopyLimits); err != nil {
		return AdapterRegeneration{}, err
	}
	var err error
	result.ReplacementInventorySHA256, err = digestAdapterSourceInventory(result.ReplacementRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	result.PreparedPackage = spec.preparedPackage
	return result, nil
}

func adapterSourceFile(module string) string {
	switch module {
	case sprigModulePath:
		return "sprig_adapter.go"
	case validatorModulePath:
		return "validator_adapter.go"
	case pebbleModulePath:
		return "pebble_adapter.go"
	case cactusStatsDModulePath:
		return "cactusstatsd_adapter.go"
	case memberlistModulePath:
		return "memberlist_adapter.go"
	case sentryModulePath:
		return "sentry_adapter.go"
	case hashicorpMetricsModulePath:
		return "hashicorpmetrics_adapter.go"
	case fxModulePath:
		return "fx_adapter.go"
	case temporalSDKModulePath:
		return "temporalsdk_adapter.go"
	case otelSDKModulePath:
		return "otelsdk_adapter.go"
	case sockaddrModulePath:
		return "sockaddr_adapter.go"
	case memoryModulePath:
		return "memory_adapter.go"
	case grpcModulePath:
		return "grpc_adapter.go"
	case xnetModulePath:
		return "xnet_adapter.go"
	case libcModulePath:
		return "libc_adapter.go"
	default:
		return ""
	}
}

func addRegeneratedSource(result *AdapterRegeneration, path, oldSource, oldReplacement string, source, replacement []byte) {
	result.Sources = append(result.Sources, RegeneratedSource{Path: path, OldSourceSHA256: oldSource, SourceSHA256: digestBytes(source), OldReplacementSHA256: oldReplacement, ReplacementSHA256: digestBytes(replacement), Source: source, Replacement: replacement})
}

func regenerateGRPC(moduleRoot, workRoot, version, sum string) (AdapterRegeneration, error) {
	result := AdapterRegeneration{Module: grpcModulePath, SourceFile: adapterSourceFile(grpcModulePath), Version: version, Sum: sum, OldOriginalInventorySHA256: grpcOriginalSourceInventorySHA256, OldReplacementInventorySHA256: grpcReplacementSourceInventorySHA256}
	var err error
	result.OriginalInventorySHA256, err = digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	replacements := map[string][]byte{}
	keepalive, err := readAdapterSource(grpcModulePath, moduleRoot, grpcKeepalivePath)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	replacements[grpcKeepalivePath], err = rewriteGRPCKeepaliveSource(keepalive)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	addRegeneratedSource(&result, grpcKeepalivePath, grpcKeepaliveSourceSHA256, grpcKeepaliveReplacementSHA256, keepalive, replacements[grpcKeepalivePath])
	for _, rewrite := range grpcLinuxRewrites {
		linux, err := readAdapterSource(grpcModulePath, moduleRoot, rewrite.linuxPath)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		nonLinux, err := readAdapterSource(grpcModulePath, moduleRoot, rewrite.nonLinuxPath)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		replacements[rewrite.linuxPath], err = regenerateGRPCLinuxSource(rewrite, nonLinux)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		addRegeneratedSource(&result, rewrite.linuxPath, rewrite.linuxSHA256, rewrite.replacementSHA256, linux, replacements[rewrite.linuxPath])
		addRegeneratedSource(&result, rewrite.nonLinuxPath, rewrite.nonLinuxSHA256, "", nonLinux, nonLinux)
	}
	for _, rewrites := range [][]sourceRewrite{grpcSyscallRewrites, grpcDNSRewrites} {
		for _, rewrite := range rewrites {
			contents, err := readAdapterSource(grpcModulePath, moduleRoot, rewrite.path)
			if err != nil {
				return AdapterRegeneration{}, err
			}
			replacements[rewrite.path], err = regenerateSourceRewrite(grpcModulePath, rewrite, contents)
			if err != nil {
				return AdapterRegeneration{}, err
			}
			addRegeneratedSource(&result, rewrite.path, rewrite.sourceSHA256, rewrite.replacementSHA256, contents, replacements[rewrite.path])
		}
	}
	return finishAdapterRegeneration(moduleRoot, workRoot, rewrittenModule{module: grpcModulePath, replacementDirectory: "google-grpc", preparedPackage: grpcModulePath + "/internal"}, result, replacements)
}

func regenerateXNet(moduleRoot, workRoot, version, sum string) (AdapterRegeneration, error) {
	result := AdapterRegeneration{Module: xnetModulePath, SourceFile: adapterSourceFile(xnetModulePath), Version: version, Sum: sum, OldOriginalInventorySHA256: xnetOriginalSourceInventorySHA256, OldReplacementInventorySHA256: xnetReplacementSourceInventorySHA256}
	var err error
	result.OriginalInventorySHA256, err = digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	socket, err := readAdapterSource(xnetModulePath, moduleRoot, xnetSocketPath)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	empty, err := readAdapterSource(xnetModulePath, moduleRoot, xnetEmptyPath)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	rewrittenSocket, err := rewriteXNetSocketSource(socket)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	rewrittenEmpty, err := replaceXNetAnchor(empty, []byte("//go:build darwin"), []byte("//go:build ignore"))
	if err != nil {
		return AdapterRegeneration{}, err
	}
	addRegeneratedSource(&result, xnetSocketPath, xnetSocketSourceSHA256, xnetSocketReplacementSHA256, socket, rewrittenSocket)
	addRegeneratedSource(&result, xnetEmptyPath, xnetEmptySourceSHA256, xnetEmptyReplacementSHA256, empty, rewrittenEmpty)
	return finishAdapterRegeneration(moduleRoot, workRoot, rewrittenModule{module: xnetModulePath, replacementDirectory: "golang-x-net", preparedPackage: xnetModulePath + "/internal/socket"}, result, map[string][]byte{xnetSocketPath: rewrittenSocket, xnetEmptyPath: rewrittenEmpty})
}

func regenerateLibc(moduleRoot, workRoot, version, sum string) (AdapterRegeneration, error) {
	result := AdapterRegeneration{Module: libcModulePath, SourceFile: adapterSourceFile(libcModulePath), Version: version, Sum: sum}
	var err error
	result.OriginalInventorySHA256, err = digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	replacements, _, err := rewriteLibcModuleUnpinned(moduleRoot)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	for _, pin := range []struct{ path, old string }{
		{"libc_darwin.go", libcDarwinSHA256}, {"libc_darwin_arm64.go", libcDarwinArm64SHA256}, {"libc_unix.go", libcUnixSHA256},
		{"syscall_musl.go", libcSyscallMuslSHA256}, {"libc_musl.go", libcMuslSHA256}, {"libc_musl_linux_amd64.go", libcMuslLinuxAmd64SHA256},
	} {
		contents, err := readAdapterSource(libcModulePath, moduleRoot, pin.path)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		addRegeneratedSource(&result, pin.path, pin.old, "", contents, replacements[pin.path])
	}
	return finishAdapterRegeneration(moduleRoot, workRoot, rewrittenModule{module: libcModulePath, replacementDirectory: "modernc-libc", preparedPackage: libcModulePath}, result, replacements)
}
