package hooks

import "unsafe"

func CryptoInternalFips140Subtle_xorBytes(dstb, xb, yb *byte, n int) {
	dst := unsafe.Slice(dstb, n)
	x := unsafe.Slice(xb, n)
	y := unsafe.Slice(yb, n)
	for i := range dst {
		dst[i] = x[i] ^ y[i]
	}
}
