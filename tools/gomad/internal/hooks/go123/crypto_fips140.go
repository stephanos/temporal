package go123

import "github.com/jellevandenhooff/gosim/gosimruntime"

var cryptoInternalFips140Indicators = map[int]uint8{}

func CryptoInternalFips140_fatal(message string) {
	panic(message)
}

func CryptoInternalFips140_getIndicator() uint8 {
	return cryptoInternalFips140Indicators[gosimruntime.GetGoroutine()]
}

func CryptoInternalFips140_setIndicator(indicator uint8) {
	cryptoInternalFips140Indicators[gosimruntime.GetGoroutine()] = indicator
}

var cryptoFips140Bypass = map[int]bool{}

func CryptoFips140_setBypass() {
	cryptoFips140Bypass[gosimruntime.GetGoroutine()] = true
}

func CryptoFips140_isBypassed() bool {
	return cryptoFips140Bypass[gosimruntime.GetGoroutine()]
}

func CryptoFips140_unsetBypass() {
	delete(cryptoFips140Bypass, gosimruntime.GetGoroutine())
}
