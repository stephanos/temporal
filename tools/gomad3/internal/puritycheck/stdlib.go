package puritycheck

import (
	"fmt"
	"regexp"
	"strings"
)

// Reviewed against go1.27; a new minor requires review before admitting symbols.
const reviewedGoMinor = "go1.27"

var pureSymbols = map[string][]string{
	"bytes":           {"*"},
	"strings":         {"*"},
	"strconv":         {"*"},
	"encoding/json":   {"*"},
	"encoding/hex":    {"*"},
	"encoding/binary": {"*"},
	"encoding/base64": {"*"},
	"compress/gzip":   {"*"},
	"errors":          {"*"},
	"math":            {"*"},
	"math/bits":       {"*"},
	"unicode":         {"*"},
	"unicode/utf8":    {"*"},
	"slices":          {"*"},
	"sort":            {"*"},
	"regexp":          {"*"},
	"path":            {"*"},
	"path/filepath":   {"Base", "Clean", "Dir", "Join", "IsAbs", "Ext", "ToSlash", "FromSlash", "VolumeName"},
	"fmt":             {"Errorf", "Sprintf", "Sprint", "Sprintln", "Append", "Appendf", "Appendln", "Fprint", "Fprintf", "Fprintln"},
	"io":              {"Copy", "CopyN", "ReadAll", "ReadFull", "ReadAtLeast", "WriteString", "LimitReader", "MultiReader", "MultiWriter", "NopCloser"},
	"crypto/sha256":   {"New", "Sum256", "Sum224"},
	"crypto/hmac":     {"New"},
	"embed":           {"FS.ReadFile"},
	"bufio":           {"NewWriterSize", "Writer.Write", "Writer.WriteString", "Writer.Flush"},
	"container/heap":  {"Push", "Remove"},
	"sync":            {"Mutex.Lock", "Mutex.Unlock", "Mutex.TryLock", "RWMutex.Lock", "RWMutex.Unlock", "RWMutex.RLock", "RWMutex.RUnlock", "RWMutex.TryLock", "RWMutex.TryRLock"},
	"sync/atomic":     {"LoadUint32", "StoreUint32"},
	"reflect":         {"TypeOf", "TypeFor", "PointerTo", "ValueOf", "Value.Type", "Value.IsValid", "Value.Kind", "Value.IsNil", "Value.Elem", "Value.Pointer", "Value.MapRange", "Value.MapIndex", "Value.Len", "Value.Index", "Value.NumField", "Value.Field", "Value.String", "Value.Bytes", "Value.IsZero", "Value.CanInterface", "Value.Interface", "Value.Bool", "Value.Int", "Value.Uint", "Value.CanAddr", "Value.Addr", "StructTag.Get", "StructField.IsExported", "MapIter.Next", "MapIter.Key", "MapIter.Value"},
	"time":            {"Duration", "Duration.*", "Date", "Unix", "UnixMilli", "UnixMicro", "UTC", "ParseInLocation", "Time.Add", "Time.Before", "Time.After", "Time.Equal", "Time.Compare", "Time.Sub", "Time.IsZero", "Time.Unix", "Time.UnixNano", "Time.UnixMilli", "Time.UnixMicro", "Time.UTC", "Time.Location", "!Now", "!Since", "!Until", "!Sleep", "!After", "!NewTimer", "!Tick"},
	"syscall":         {"Errno", "Errno.Error", "E*"},
	"os":              {}, "os/exec": {}, "net": {}, "math/rand": {}, "crypto/rand": {},
}

func allowed(path, name string) bool {
	admitted := false
	for _, pattern := range pureSymbols[path] {
		denied := strings.HasPrefix(pattern, "!")
		pattern = strings.TrimPrefix(pattern, "!")
		match := pattern == name || strings.HasSuffix(pattern, "*") && strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
		if match && denied {
			return false
		}
		admitted = admitted || match
	}
	return admitted
}

var goMinorPattern = regexp.MustCompile(`go[0-9]+\.[0-9]+`)

func checkVersion(active string) error {
	minor := goMinorPattern.FindString(active)
	if minor != reviewedGoMinor {
		return fmt.Errorf("stdlib allowlist reviewed for %s; active toolchain %s requires review", reviewedGoMinor, strings.TrimSpace(active))
	}
	return nil
}
