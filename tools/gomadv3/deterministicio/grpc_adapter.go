package deterministicio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

const (
	grpcModulePath                       = "google.golang.org/grpc"
	grpcVersion                          = "v1.80.0"
	grpcSum                              = "h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM="
	grpcOriginalSourceInventorySHA256    = "sha256:8b2fee8f36a0554c6652dccb95641983d2bf57c2efaa7af584c0ecc08ce6c1aa"
	grpcKeepaliveSourceSHA256            = "sha256:e8bfe03234b391d24006a3a274590111f0f8705fc5b25d9a78391bfdde3df32c"
	grpcKeepaliveReplacementSHA256       = "sha256:8705566fa6ba58f69d8c8215227ddadad46794c333bca38fe6d5399d6be24e8c"
	grpcReplacementSourceInventorySHA256 = "sha256:6bfaf02259a872caad5349a60dff7a2efa2e4b61eae51ea463d20398a078767b"
	grpcPreparedInternalSourceSetSHA256  = "sha256:348f37231e8391fd9361eb84ed9d5a39b9cacc4136461ce103ccc828be7db250"
	grpcKeepalivePath                    = "internal/tcp_keepalive_unix.go"
)

func prepareGRPC(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	if identity.Module != grpcModulePath || identity.Version != grpcVersion || identity.Sum != grpcSum {
		return adapterPreparation{}, errors.New("gRPC adapter identity mismatch")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "google.golang.org", "grpc@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned gRPC module: %w", err)
	}
	if err := verifyGRPCModule(moduleSource); err != nil {
		return adapterPreparation{}, err
	}
	source := filepath.Join(moduleSource, filepath.FromSlash(grpcKeepalivePath))
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() {
		return adapterPreparation{}, errors.New("pinned gRPC keepalive source is not a regular file")
	}
	contents, err := os.ReadFile(source)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("read pinned gRPC keepalive source: %w", err)
	}
	rewritten, err := rewriteGRPCKeepalive(contents)
	if err != nil {
		return adapterPreparation{}, err
	}
	moduleReplacement := filepath.Join(root, "google-grpc")
	if err := copyAdapterModule(moduleSource, moduleReplacement, map[string][]byte{grpcKeepalivePath: rewritten}, defaultAdapterCopyLimits); err != nil {
		return adapterPreparation{}, fmt.Errorf("copy gRPC adapter module: %w", err)
	}
	replacementInventory, err := digestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash gRPC replacement inventory: %w", err)
	}
	if replacementInventory != grpcReplacementSourceInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("gRPC replacement inventory identity mismatch: got %s, want %s", replacementInventory, grpcReplacementSourceInventorySHA256)
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: source, ReplacementRoot: moduleReplacement, Replacement: filepath.Join(moduleReplacement, filepath.FromSlash(grpcKeepalivePath)),
			PreparedPackage:                  grpcModulePath + "/internal",
			SourceSHA256:                     grpcKeepaliveSourceSHA256,
			ReplacementSHA256:                grpcKeepaliveReplacementSHA256,
			OriginalSourceInventorySHA256:    grpcOriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          grpcPreparedInternalSourceSetSHA256,
		},
	}, nil
}

func verifyGRPCModule(moduleRoot string) error {
	inventory, err := digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return fmt.Errorf("hash pinned gRPC source inventory: %w", err)
	}
	if inventory != grpcOriginalSourceInventorySHA256 {
		return fmt.Errorf("pinned gRPC source inventory identity mismatch: got %s, want %s", inventory, grpcOriginalSourceInventorySHA256)
	}
	return nil
}

func rewriteGRPCKeepalive(contents []byte) ([]byte, error) {
	if digestBytes(contents) != grpcKeepaliveSourceSHA256 {
		return nil, errors.New("pinned gRPC keepalive source identity mismatch")
	}
	rewritten, err := rewriteGRPCKeepaliveSource(contents)
	if err != nil {
		return nil, err
	}
	if digestBytes(rewritten) != grpcKeepaliveReplacementSHA256 {
		return nil, errors.New("gRPC keepalive replacement identity mismatch")
	}
	return rewritten, nil
}

func rewriteGRPCKeepaliveSource(contents []byte) ([]byte, error) {
	rewrites := []struct {
		anchor      []byte
		replacement []byte
	}{
		{anchor: []byte("\t\"syscall\"\n")},
		{anchor: []byte("\n\t\"golang.org/x/sys/unix\"\n"), replacement: []byte("\n")},
		{anchor: []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")},
	}
	result := append([]byte(nil), contents...)
	for _, rewrite := range rewrites {
		if bytes.Count(result, rewrite.anchor) != 1 {
			return nil, fmt.Errorf("pinned gRPC keepalive rewrite anchor mismatch for %q", rewrite.anchor)
		}
		result = bytes.Replace(result, rewrite.anchor, rewrite.replacement, 1)
	}
	return result, nil
}
