package measure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	errutil "errors"
	"fmt"
	"os"
	"path/filepath"

	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func helperPath(runCtx context.Context) (measureValue0 string, measureErr error) {
	override := input(runCtx, "GOMAD_WASM_TEST_HELPER")
	if override != "" {
		if !filepath.IsAbs(override) {
			measureErr = errutil.Join(measureErr, errutil.New("GOMAD_WASM_TEST_HELPER must be absolute"))
			return
		}
		_, err := os.Stat(override)
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		return override, measureErr
	}
	name, err := absolute(runCtx, "../wasmhost/target/release/gomad-wasmhost")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	_, err = os.Stat(name)
	if err != nil {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New(fmt.Sprint("build pinned release wasmhost to run guest gates: ", err))))
		return
	}
	return name, measureErr
}
func fileHash(runCtx context.Context, name string) (measureValue0 string, measureErr error) {
	data, err := os.ReadFile(name)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), measureErr
}
func binaryInventory(runCtx context.Context, data []byte) (measureValue0 []wasi.Import, measureValue1 uint64, measureErr error) {
	if len(data) < 8 || string(data[:4]) != "\x00asm" {
		measureErr = errutil.Join(measureErr, errutil.New("not binary WASM"))
		return
	}
	readUint := func(runCtx context.Context, buffer *[]byte) (measureValue0 uint64, measureErr error) {
		var value uint64
		shift := uint(0)
		for ; shift < 64; shift += 7 {
			if len(*buffer) == 0 {
				measureErr = errutil.Join(measureErr, errutil.New("truncated WASM LEB"))
				return
			}
			b := (*buffer)[0]
			*buffer = (*buffer)[1:]
			value |= uint64(b&127) << shift
			if b < 128 {
				return value, measureErr
			}
		}
		measureErr = errutil.Join(measureErr, errutil.New("overflow WASM LEB"))
		return
	}
	readName := func(runCtx context.Context, buffer *[]byte) (measureValue0 string, measureErr error) {
		size, measureFailure547 := readUint(runCtx, buffer)
		if measureFailure547 != nil {
			measureErr = errutil.Join(measureErr, measureFailure547)
			return
		}
		if size > uint64(len(*buffer)) {
			measureErr = errutil.Join(measureErr, errutil.New("truncated WASM name"))
			return
		}
		name := string((*buffer)[:size])
		*buffer = (*buffer)[size:]
		return name, measureErr
	}
	types := []wasi.Import{}
	imports := []wasi.Import{}
	pages := uint64(0)
	remaining := data[8:]
	for len(remaining) > 0 {
		kind := remaining[0]
		remaining = remaining[1:]
		size, measureFailure548 := readUint(runCtx, &remaining)
		if measureFailure548 != nil {
			measureErr = errutil.Join(measureErr, measureFailure548)
			return
		}
		if size > uint64(len(remaining)) {
			measureErr = errutil.Join(measureErr, errutil.New("truncated WASM section"))
			return
		}
		section := remaining[:size]
		remaining = remaining[size:]
		switch kind {
		case 1:
			count, measureFailure549 := readUint(runCtx, &section)
			if measureFailure549 != nil {
				measureErr = errutil.Join(measureErr, measureFailure549)
				return
			}
			i := uint64(0)
			for ; i < count; i++ {
				if len(section) == 0 || section[0] != 0x60 {
					measureErr = errutil.Join(measureErr, errutil.New("fixture WASM function type"))
					return
				}
				section = section[1:]
				signature := wasi.Import{Params: []string{}, Results: []string{}}
				j := 0
				for ; j < 2; j++ {
					count, measureFailure550 := readUint(runCtx, &section)
					if measureFailure550 != nil {
						measureErr = errutil.Join(measureErr, measureFailure550)
						return
					}
					k := uint64(0)
					for ; k < count; k++ {
						if len(section) == 0 {
							measureErr = errutil.Join(measureErr, errutil.New("truncated value type"))
							return
						}
						name := map[byte]string{0x7f: "i32", 0x7e: "i64", 0x7d: "f32", 0x7c: "f64"}[section[0]]
						section = section[1:]
						if name == "" {
							measureErr = errutil.Join(measureErr, errutil.New("fixture WASM value type"))
							return
						}
						if j == 0 {
							signature.Params = append(signature.Params, name)
						} else {
							signature.Results = append(signature.Results, name)
						}
					}
				}
				types = append(types, signature)
			}
		case 2:
			count, measureFailure551 := readUint(runCtx, &section)
			if measureFailure551 != nil {
				measureErr = errutil.Join(measureErr, measureFailure551)
				return
			}
			i := uint64(0)
			for ; i < count; i++ {
				measureResult552_0, measureFailure552 := readName(runCtx, &section)
				if measureFailure552 != nil {
					measureErr = errutil.Join(measureErr, measureFailure552)
					return
				}
				measureResult553_0, measureFailure553 := readName(runCtx, &section)
				if measureFailure553 != nil {
					measureErr = errutil.Join(measureErr, measureFailure553)
					return
				}
				module, name := measureResult552_0, measureResult553_0
				if len(section) == 0 || section[0] != 0 {
					measureErr = errutil.Join(measureErr, errutil.New("fixture import is not function"))
					return
				}
				section = section[1:]
				index, measureFailure554 := readUint(runCtx, &section)
				if measureFailure554 != nil {
					measureErr = errutil.Join(measureErr, measureFailure554)
					return
				}
				if index >= uint64(len(types)) {
					measureErr = errutil.Join(measureErr, errutil.New("invalid fixture type index"))
					return
				}
				signature := types[index]
				signature.Module = module
				signature.Name = name
				imports = append(imports, signature)
			}
		case 5:
			measureResult555_0, measureFailure555 := readUint(runCtx, &section)
			if measureFailure555 != nil {
				measureErr = errutil.Join(measureErr, measureFailure555)
				return
			}
			if measureResult555_0 != 1 {
				measureErr = errutil.Join(measureErr, errutil.New("fixture memory count"))
				return
			}
			flags, measureFailure556 := readUint(runCtx, &section)
			if measureFailure556 != nil {
				measureErr = errutil.Join(measureErr, measureFailure556)
				return
			}
			measureResult557_0, measureFailure557 := readUint(runCtx, &section)
			if measureFailure557 != nil {
				measureErr = errutil.Join(measureErr, measureFailure557)
				return
			}
			pages = measureResult557_0
			if flags&1 != 0 {
				measureResult558_0, measureFailure558 := readUint(runCtx, &section)
				if measureFailure558 != nil {
					measureErr = errutil.Join(measureErr, measureFailure558)
					return
				}
				_ = measureResult558_0
			}
		}
	}
	if pages == 0 {
		measureErr = errutil.Join(measureErr, errutil.New("fixture memory missing"))
		return
	}
	return imports, pages, measureErr
}
