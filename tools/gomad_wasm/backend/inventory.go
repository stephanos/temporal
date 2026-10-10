package backend

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

type moduleReader struct {
	data []byte
	err  error
}

func (r *moduleReader) byte() byte {
	if len(r.data) == 0 {
		r.err = io.ErrUnexpectedEOF
		return 0
	}
	b := r.data[0]
	r.data = r.data[1:]
	return b
}
func (r *moduleReader) uint() uint64 {
	var n uint64
	for shift := uint(0); shift < 35; shift += 7 {
		b := r.byte()
		if r.err != nil {
			return 0
		}
		if shift == 28 && b > 15 {
			r.err = errors.New("overflow module integer")
			return 0
		}
		n |= uint64(b&127) << shift
		if b < 128 {
			return n
		}
	}
	r.err = errors.New("invalid module integer")
	return 0
}
func (r *moduleReader) take(n uint64) []byte {
	if n > uint64(len(r.data)) {
		r.err = io.ErrUnexpectedEOF
		return nil
	}
	v := r.data[:n]
	r.data = r.data[n:]
	return v
}
func (r *moduleReader) name() string {
	v := r.take(r.uint())
	if !utf8.Valid(v) {
		r.err = errors.New("invalid module name")
	}
	return string(v)
}
func inventory(data []byte) ([]wasi.Import, uint64, error) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0, 97, 115, 109, 1, 0, 0, 0}) {
		return nil, 0, errors.New("WASM binary header is required")
	}
	r := moduleReader{data: data[8:]}
	types := []wasi.Import{}
	imports := []wasi.Import{}
	var pages uint64
	seen := map[byte]bool{}
	for len(r.data) > 0 && r.err == nil {
		kind := r.byte()
		body := r.take(r.uint())
		if r.err != nil {
			break
		}
		s := moduleReader{data: body}
		if kind != 0 && seen[kind] {
			return nil, 0, errors.New("duplicate module section")
		}
		seen[kind] = true
		switch kind {
		case 1:
			for count := s.uint(); count > 0 && s.err == nil; count-- {
				if s.byte() != 0x60 {
					return nil, 0, errors.New("unsupported module type")
				}
				signature := wasi.Import{Params: []string{}, Results: []string{}}
				for j := 0; j < 2; j++ {
					for count := s.uint(); count > 0 && s.err == nil; count-- {
						name := map[byte]string{0x7f: "i32", 0x7e: "i64", 0x7d: "f32", 0x7c: "f64"}[s.byte()]
						if name == "" {
							return nil, 0, errors.New("unsupported module value type")
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
			for count := s.uint(); count > 0 && s.err == nil; count-- {
				module, name := s.name(), s.name()
				if s.byte() != 0 {
					return nil, 0, errors.New("nonfunction module import is unsupported")
				}
				index := s.uint()
				if index >= uint64(len(types)) {
					return nil, 0, errors.New("invalid import type")
				}
				signature := types[index]
				signature.Module = module
				signature.Name = name
				imports = append(imports, signature)
			}
		case 5:
			if s.uint() != 1 {
				return nil, 0, errors.New("exactly one module memory is required")
			}
			flags := s.uint()
			if flags > 1 {
				return nil, 0, errors.New("shared or memory64 module is unsupported")
			}
			pages = s.uint()
			if flags == 1 {
				maximum := s.uint()
				if maximum < pages {
					return nil, 0, errors.New("invalid module memory maximum")
				}
			}
		default:
			continue
		}
		if s.err != nil {
			return nil, 0, fmt.Errorf("module section: %w", s.err)
		}
		if len(s.data) != 0 {
			return nil, 0, errors.New("trailing module section data")
		}
	}
	if r.err != nil {
		return nil, 0, r.err
	}
	if pages == 0 || pages > 65536 {
		return nil, 0, errors.New("invalid initial module memory")
	}
	return imports, pages, nil
}
