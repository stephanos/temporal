package translate

var go126CompatibilityPolicy = stdlibCompatibilityPolicy{
	targetGoVersion:         "go1.26",
	hooks:                   go126Hooks,
	hooksByArch:             generatedStdlibHooksByArch,
	skippedPackages:         go126SkippedPackages,
	keepAsmPackages:         go126KeepAsmPackages,
	acceptedLinknames:       go126AcceptedLinknames,
	acceptedNoBodyLinknames: go126AcceptedNoBodyLinknames,
	globalsDontTranslate:    go126GlobalsDontTranslate,
}

var go126SkippedPackages = map[string]bool{
	"runtime": true, // XXX wait what
	"errors":  true,
	"reflect": true,
	// "strings": true,
	"strconv": true,
	// embed: true, // this will also have io.EOF problems... unless we can just use the original there please???
	// XXX: for all unconverted packages, figure out all references to converted packages, and have a plan.
	"embed":    true, // XXX can we link back to the original here somehow???
	"math":     true,
	"math/big": true,

	"unsafe": true,

	"runtime/coverage": true, // XXX for now
	"runtime/metrics":  true, // XXX for now
	"runtime/pprof":    true, // XXX for now

	// XXX: rewrite internal/cpu to golang.org/x/sys/cpu?
	"vendor/golang.org/x/sys/cpu": true, // XXX for now

	"unique": true, // XXX: yes

	"testing":                     true,
	"testing/internal/testdeps":   true,
	"internal/reflectlite":        true,
	gomadruntimePackage:           true,
	gomadruntimePackage + "_test": true, // eh
	gomadruntimePackage + ".test": true, // eh
	// reflectPackage: true,
}

var go126KeepAsmPackages = map[string]bool{
	"crypto/aes":                                   true,
	"crypto/internal/boring/sig":                   true,
	"crypto/internal/fips140/aes":                  true,
	"crypto/internal/fips140/aes/gcm":              true,
	"crypto/internal/fips140/bigmod":               true,
	"crypto/internal/fips140/edwards25519/field":   true,
	"crypto/internal/fips140/nistec":               true,
	"crypto/internal/fips140/sha256":               true,
	"crypto/internal/fips140/sha3":                 true,
	"crypto/internal/fips140/sha512":               true,
	"crypto/internal/nistec":                       true,
	"crypto/md5":                                   true,
	"crypto/sha1":                                  true,
	"crypto/sha256":                                true,
	"crypto/sha512":                                true,
	"crypto/subtle":                                true,
	"crypto/internal/bigmod":                       true,
	"crypto/internal/edwards25519/field":           true,
	"vendor/golang.org/x/crypto/chacha20":          true,
	"vendor/golang.org/x/crypto/internal/poly1305": true,
	"vendor/golang.org/x/crypto/chacha20poly1305":  true,
	"vendor/golang.org/x/crypto/sha3":              true,
	"hash/crc32":                                   true,

	"net/url":  true, // XXX: linkname setpath nonsense
	"net/http": true, // XXX: linkname roundtrip nonsense

	"github.com/cespare/xxhash/v2":                       true,
	"github.com/golang/snappy":                           true,
	"github.com/klauspost/compress/zstd/internal/xxhash": true,
}

var go126GlobalsDontTranslate = map[packageSelector]bool{
	{Pkg: "crypto/sha512", Selector: "_K"}:                  true,
	{Pkg: "crypto/internal/fips140/sha256", Selector: "_K"}: true,
	{Pkg: "crypto/internal/fips140/sha512", Selector: "_K"}: true,
	// {pkg: "sync/atomic", selector: "firstStoreInProgress"}: true, // no way... universe strikes once again
	{Pkg: "encoding/json", Selector: "fieldCache"}:   true,
	{Pkg: "encoding/json", Selector: "encoderCache"}: true,
	// this would be nice but breaks determinism because sometimes the cache works.......
	// XXX: register all types in our own registry, warm up the cache?
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "decomps"}:          true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfcValues"}:        true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfcIndex"}:         true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfcSparseValues"}:  true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfkcValues"}:       true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfkcIndex"}:        true,
	{Pkg: "vendor/golang.org/x/text/unicode/norm", Selector: "nfkcSparseValues"}: true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "decomps"}:                 true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfcValues"}:               true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfcIndex"}:                true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfcSparseValues"}:         true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfkcValues"}:              true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfkcIndex"}:               true,
	{Pkg: "golang.org/x/text/unicode/norm", Selector: "nfkcSparseValues"}:        true,
	{Pkg: "vendor/golang.org/x/text/unicode/bidi", Selector: "bidiValues"}:       true,
	{Pkg: "vendor/golang.org/x/text/unicode/bidi", Selector: "bidiIndex"}:        true,
	{Pkg: "golang.org/x/text/unicode/bidi", Selector: "bidiValues"}:              true,
	{Pkg: "golang.org/x/text/unicode/bidi", Selector: "bidiIndex"}:               true,
	{Pkg: "vendor/golang.org/x/net/idna", Selector: "idnaValues"}:                true,
	{Pkg: "vendor/golang.org/x/net/idna", Selector: "idnaIndex"}:                 true,
	{Pkg: "vendor/golang.org/x/net/idna", Selector: "idnaSparseValues"}:          true,
	{Pkg: "golang.org/x/net/idna", Selector: "idnaValues"}:                       true,
	{Pkg: "golang.org/x/net/idna", Selector: "idnaIndex"}:                        true,
	{Pkg: "golang.org/x/net/idna", Selector: "idnaSparseValues"}:                 true,
	{Pkg: reflectPackage, Selector: "mapInterfaceType"}:                          true,
	{Pkg: reflectPackage, Selector: "jankHashMap"}:                               true,

	// xxx amd64 asm
	{Pkg: "crypto/sha256", Selector: "useSHA"}:                                true,
	{Pkg: "crypto/sha256", Selector: "useAVX2"}:                               true,
	{Pkg: "vendor/golang.org/x/crypto/chacha20poly1305", Selector: "useAVX2"}: true,
	{Pkg: "crypto/internal/bigmod", Selector: "supportADX"}:                   true,

	{Pkg: "github.com/cespare/xxhash/v2", Selector: "primes"}: true,
}

var go126Hooks = map[packageSelector]packageSelector{
	{Pkg: "crypto/internal/constanttime", Selector: "boolToUint8"}: {Pkg: stdlibHooksPackage, Selector: "CryptoInternalConstanttime_boolToUint8"},
	{Pkg: "crypto/internal/fips140/subtle", Selector: "xorBytes"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "crypto/subtle", Selector: "setDITEnabled"}:              {Pkg: stdlibHooksPackage, Selector: "InternalRuntimeSys_EnableDIT"},
	{Pkg: "crypto/subtle", Selector: "setDITDisabled"}:             {Pkg: stdlibHooksPackage, Selector: "InternalRuntimeSys_DisableDIT"},

	{Pkg: "golang.org/x/sys/unix", Selector: "RawSyscall"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "RawSyscall6"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "RawSyscallNoError"}: {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "Syscall"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "Syscall6"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "SyscallNoError"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "golang.org/x/sys/unix", Selector: "vgetrandom"}:        {Pkg: stdlibHooksPackage, Selector: "InternalSyscallUnix_vgetrandom"},

	// amd64 only?
	{Pkg: "golang.org/x/sys/unix", Selector: "gettimeofday"}: {Pkg: stdlibHooksPackage},

	{Pkg: "hash/maphash", Selector: "runtime_rand"}: {Pkg: stdlibHooksPackage},

	{Pkg: "internal/abi", Selector: "FuncPCABI0"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/abi", Selector: "FuncPCABIInternal"}: {Pkg: stdlibHooksPackage},

	{Pkg: "internal/chacha8rand", Selector: "block"}: {Pkg: stdlibHooksPackage},

	{Pkg: "internal/bytealg", Selector: "Compare"}:                        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "Count"}:                          {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "CountString"}:                    {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "Equal"}:                          {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "Index"}:                          {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "IndexByte"}:                      {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "IndexByteString"}:                {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "IndexString"}:                    {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "MakeNoZero"}:                     {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "abigen_runtime_cmpstring"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "abigen_runtime_memequal"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/bytealg", Selector: "abigen_runtime_memequal_varlen"}: {Pkg: stdlibHooksPackage},

	// arm only?
	{Pkg: "internal/cpu", Selector: "getMIDR"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "internal/cpu", Selector: "getisar0"}: {Pkg: stdlibHooksPackage},
	{Pkg: "internal/cpu", Selector: "getpfr0"}:  {Pkg: stdlibHooksPackage},

	// amd only?
	{Pkg: "internal/cpu", Selector: "cpuid"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "internal/cpu", Selector: "xgetbv"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "internal/cpu", Selector: "getGOAMD64level"}: {Pkg: stdlibHooksPackage},

	{Pkg: "internal/godebug", Selector: "registerMetric"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "internal/godebug", Selector: "setNewIncNonDefault"}: {Pkg: stdlibHooksPackage},
	{Pkg: "internal/godebug", Selector: "setUpdate"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "internal/godebug", Selector: "write"}:               {Pkg: stdlibHooksPackage},

	{Pkg: "internal/poll", Selector: "runtimeNano"}:                    {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_Semacquire"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_Semrelease"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_isPollServerDescriptor"}: {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollClose"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollOpen"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollReset"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollServerInit"}:         {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollSetDeadline"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollUnblock"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollWait"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "internal/poll", Selector: "runtime_pollWaitCanceled"}:       {Pkg: stdlibHooksPackage},

	{Pkg: "internal/runtime/syscall", Selector: "Syscall6"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/syscall/linux", Selector: "Syscall6"}: {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "DisableDIT"}:         {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "DITEnabled"}:         {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "EnableDIT"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "GetCallerPC"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "GetCallerSP"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "internal/runtime/sys", Selector: "GetClosurePtr"}:      {Pkg: stdlibHooksPackage},

	{Pkg: "internal/race", Selector: "Acquire"}:       {Pkg: stdlibHooksPackage, Selector: "InternalRace_Acquire"},
	{Pkg: "internal/race", Selector: "Release"}:       {Pkg: stdlibHooksPackage, Selector: "InternalRace_Release"},
	{Pkg: "internal/race", Selector: "ReleaseMerge"}:  {Pkg: stdlibHooksPackage, Selector: "InternalRace_ReleaseMerge"},
	{Pkg: "internal/race", Selector: "Disable"}:       {Pkg: stdlibHooksPackage, Selector: "InternalRace_Disable"},
	{Pkg: "internal/race", Selector: "Enable"}:        {Pkg: stdlibHooksPackage, Selector: "InternalRace_Enable"},
	{Pkg: "internal/race", Selector: "Read"}:          {Pkg: stdlibHooksPackage, Selector: "InternalRace_Read"},
	{Pkg: "internal/race", Selector: "ReadPC"}:        {Pkg: stdlibHooksPackage, Selector: "InternalRace_ReadPC"},
	{Pkg: "internal/race", Selector: "ReadObjectPC"}:  {Pkg: stdlibHooksPackage, Selector: "InternalRace_ReadObjectPC"},
	{Pkg: "internal/race", Selector: "Write"}:         {Pkg: stdlibHooksPackage, Selector: "InternalRace_Write"},
	{Pkg: "internal/race", Selector: "WritePC"}:       {Pkg: stdlibHooksPackage, Selector: "InternalRace_WritePC"},
	{Pkg: "internal/race", Selector: "WriteObjectPC"}: {Pkg: stdlibHooksPackage, Selector: "InternalRace_WriteObjectPC"},
	{Pkg: "internal/race", Selector: "ReadRange"}:     {Pkg: stdlibHooksPackage, Selector: "InternalRace_ReadRange"},
	{Pkg: "internal/race", Selector: "WriteRange"}:    {Pkg: stdlibHooksPackage, Selector: "InternalRace_WriteRange"},
	{Pkg: "internal/race", Selector: "Errors"}:        {Pkg: stdlibHooksPackage, Selector: "InternalRace_Errors"},

	{Pkg: "internal/synctest", Selector: "Run"}:          {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_Run"},
	{Pkg: "internal/synctest", Selector: "Wait"}:         {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_Wait"},
	{Pkg: "internal/synctest", Selector: "IsInBubble"}:   {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_IsInBubble"},
	{Pkg: "internal/synctest", Selector: "associate"}:    {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_associate"},
	{Pkg: "internal/synctest", Selector: "disassociate"}: {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_disassociate"},
	{Pkg: "internal/synctest", Selector: "isAssociated"}: {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_isAssociated"},
	{Pkg: "internal/synctest", Selector: "acquire"}:      {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_acquire"},
	{Pkg: "internal/synctest", Selector: "release"}:      {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_release"},
	{Pkg: "internal/synctest", Selector: "inBubble"}:     {Pkg: stdlibHooksPackage, Selector: "InternalSynctest_inBubble"},

	{Pkg: "internal/sync", Selector: "runtime_rand"}:            {Pkg: stdlibHooksPackage, Selector: "HashMaphash_runtime_rand"},
	{Pkg: "internal/sync", Selector: "runtime_SemacquireMutex"}: {Pkg: stdlibHooksPackage, Selector: "Sync_runtime_SemacquireMutex"},
	{Pkg: "internal/sync", Selector: "runtime_Semrelease"}:      {Pkg: stdlibHooksPackage, Selector: "Sync_runtime_Semrelease"},
	{Pkg: "internal/sync", Selector: "runtime_canSpin"}:         {Pkg: stdlibHooksPackage, Selector: "Sync_runtime_canSpin"},
	{Pkg: "internal/sync", Selector: "runtime_doSpin"}:          {Pkg: stdlibHooksPackage, Selector: "Sync_runtime_doSpin"},
	{Pkg: "internal/sync", Selector: "runtime_nanotime"}:        {Pkg: stdlibHooksPackage, Selector: "Sync_runtime_nanotime"},
	{Pkg: "internal/sync", Selector: "throw"}:                   {Pkg: stdlibHooksPackage, Selector: "Sync_throw"},
	{Pkg: "internal/sync", Selector: "fatal"}:                   {Pkg: stdlibHooksPackage, Selector: "Sync_fatal"},

	{Pkg: "internal/syscall/unix", Selector: "GetRandom"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "internal/syscall/unix", Selector: "fcntl"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "internal/syscall/unix", Selector: "Openat"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "internal/syscall/unix", Selector: "Unlinkat"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "internal/syscall/unix", Selector: "vgetrandom"}: {Pkg: stdlibHooksPackage, Selector: "InternalSyscallUnix_vgetrandom"},

	{Pkg: "iter", Selector: "coroswitch"}: {Pkg: stdlibHooksPackage},
	{Pkg: "iter", Selector: "newcoro"}:    {Pkg: stdlibHooksPackage},

	{Pkg: "maps", Selector: "clone"}: {Pkg: stdlibHooksPackage},

	{Pkg: "math/rand", Selector: "runtime_rand"}: {Pkg: stdlibHooksPackage},

	{Pkg: "math/rand/v2", Selector: "runtime_rand"}: {Pkg: stdlibHooksPackage},

	{Pkg: "net", Selector: "runtime_rand"}: {Pkg: stdlibHooksPackage},

	{Pkg: "os", Selector: "runtime_args"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "runtime_beforeExit"}: {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "runtime_rand"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "sigpipe"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "checkClonePidfd"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "ignoreSIGSYS"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "os", Selector: "restoreSIGSYS"}:      {Pkg: stdlibHooksPackage},

	{Pkg: "os/signal", Selector: "signalWaitUntilIdle"}: {Pkg: stdlibHooksPackage},
	{Pkg: "os/signal", Selector: "signal_disable"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "os/signal", Selector: "signal_enable"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "os/signal", Selector: "signal_ignore"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "os/signal", Selector: "signal_ignored"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "os/signal", Selector: "signal_recv"}:         {Pkg: stdlibHooksPackage},

	{Pkg: "runtime/debug", Selector: "SetTraceback"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "WriteHeapDump"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "freeOSMemory"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "modinfo"}:         {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "readGCStats"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "setGCPercent"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "setMaxStack"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "setMaxThreads"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "setMemoryLimit"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/debug", Selector: "setPanicOnFault"}: {Pkg: stdlibHooksPackage},

	{Pkg: "runtime/trace", Selector: "userLog"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/trace", Selector: "userRegion"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/trace", Selector: "userTaskCreate"}: {Pkg: stdlibHooksPackage},
	{Pkg: "runtime/trace", Selector: "userTaskEnd"}:    {Pkg: stdlibHooksPackage},

	{Pkg: "sync", Selector: "fatal"}:                       {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_LoadAcquintptr"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_Semacquire"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_SemacquireWaitGroup"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_SemacquireMutex"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_SemacquireRWMutex"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_SemacquireRWMutexR"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_Semrelease"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_StoreReluintptr"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_canSpin"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_doSpin"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_nanotime"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_notifyListAdd"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_notifyListCheck"}:     {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_notifyListNotifyAll"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_notifyListNotifyOne"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_notifyListWait"}:      {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_procPin"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_procUnpin"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_randn"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "runtime_registerPoolCleanup"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync", Selector: "throw"}:                       {Pkg: stdlibHooksPackage},

	{Pkg: "sync/atomic", Selector: "AddInt32"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AddInt64"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AddPointer"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AddUint32"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AddUint64"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AddUintptr"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AndInt32"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AndInt64"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AndUint32"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AndUint64"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "AndUintptr"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapInt32"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapInt64"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapPointer"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapUint32"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapUint64"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "CompareAndSwapUintptr"}: {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadInt32"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadInt64"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadPointer"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadUint32"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadUint64"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "LoadUintptr"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "OrInt32"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "OrInt64"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "OrUint32"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "OrUint64"}:              {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "OrUintptr"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StoreInt32"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StoreInt64"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StorePointer"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StoreUint32"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StoreUint64"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "StoreUintptr"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapInt32"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapInt64"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapPointer"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapUint32"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapUint64"}:            {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "SwapUintptr"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "runtime_procPin"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "sync/atomic", Selector: "runtime_procUnpin"}:     {Pkg: stdlibHooksPackage},

	{Pkg: "syscall", Selector: "Exit"}:                        {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "Getpagesize"}:                 {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "RawSyscall6"}:                 {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "cgocaller"}:                   {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "hasWaitingReaders"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "rawSyscallNoError"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "rawVforkSyscall"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtimeClearenv"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtimeSetenv"}:               {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtimeUnsetenv"}:             {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_AfterExec"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_AfterFork"}:           {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_AfterForkInChild"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_BeforeExec"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_BeforeFork"}:          {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_doAllThreadsSyscall"}: {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_entersyscall"}:        {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_envs"}:                {Pkg: stdlibHooksPackage},
	{Pkg: "syscall", Selector: "runtime_exitsyscall"}:         {Pkg: stdlibHooksPackage},

	{Pkg: "syscall", Selector: "FcntlFlock"}: {Pkg: stdlibHooksPackage},

	// amd64 only?
	{Pkg: "syscall", Selector: "gettimeofday"}: {Pkg: stdlibHooksPackage},

	{Pkg: "time", Selector: "Sleep"}:       {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "now"}:         {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "newTimer"}:    {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "resetTimer"}:  {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "runtimeNano"}: {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "stopTimer"}:   {Pkg: stdlibHooksPackage},
	{Pkg: "time", Selector: "runtimeNow"}:  {Pkg: stdlibHooksPackage, Selector: "Time_now"},
	{Pkg: "time", Selector: "runtimeIsBubbled"}: {
		Pkg: stdlibHooksPackage, Selector: "InternalSynctest_IsInBubble",
	},

	{Pkg: "weak", Selector: "runtime_registerWeakPointer"}: {Pkg: stdlibHooksPackage, Selector: "Weak_runtime_registerWeakPointer"},
	{Pkg: "weak", Selector: "runtime_makeStrongFromWeak"}:  {Pkg: stdlibHooksPackage, Selector: "Weak_runtime_makeStrongFromWeak"},

	{Pkg: "crypto/internal/fips140", Selector: "fatal"}:        {Pkg: stdlibHooksPackage, Selector: "CryptoInternalFips140_fatal"},
	{Pkg: "crypto/internal/fips140", Selector: "getIndicator"}: {Pkg: stdlibHooksPackage, Selector: "CryptoInternalFips140_getIndicator"},
	{Pkg: "crypto/internal/fips140", Selector: "setIndicator"}: {Pkg: stdlibHooksPackage, Selector: "CryptoInternalFips140_setIndicator"},
	{Pkg: "crypto/fips140", Selector: "setBypass"}:             {Pkg: stdlibHooksPackage, Selector: "CryptoFips140_setBypass"},
	{Pkg: "crypto/fips140", Selector: "isBypassed"}:            {Pkg: stdlibHooksPackage, Selector: "CryptoFips140_isBypassed"},
	{Pkg: "crypto/fips140", Selector: "unsetBypass"}:           {Pkg: stdlibHooksPackage, Selector: "CryptoFips140_unsetBypass"},
	{Pkg: "crypto/internal/sysrand", Selector: "fatal"}:        {Pkg: stdlibHooksPackage, Selector: "Sync_fatal"},
	{Pkg: "crypto/rand", Selector: "fatal"}:                    {Pkg: stdlibHooksPackage, Selector: "Sync_fatal"},
}

var go126AcceptedLinknames = map[packageSelector]packageSelector{
	{Pkg: "hash/maphash", Selector: "runtime_memhash"}:                      {Pkg: "runtime", Selector: "memhash"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess1"}:          {Pkg: "runtime", Selector: "mapaccess1"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess2"}:          {Pkg: "runtime", Selector: "mapaccess2"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign"}:           {Pkg: "runtime", Selector: "mapassign"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess1_fast32"}:   {Pkg: "runtime", Selector: "mapaccess1_fast32"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess2_fast32"}:   {Pkg: "runtime", Selector: "mapaccess2_fast32"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign_fast32"}:    {Pkg: "runtime", Selector: "mapassign_fast32"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign_fast32ptr"}: {Pkg: "runtime", Selector: "mapassign_fast32ptr"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapdelete_fast32"}:    {Pkg: "runtime", Selector: "mapdelete_fast32"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess1_fast64"}:   {Pkg: "runtime", Selector: "mapaccess1_fast64"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess2_fast64"}:   {Pkg: "runtime", Selector: "mapaccess2_fast64"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign_fast64"}:    {Pkg: "runtime", Selector: "mapassign_fast64"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign_fast64ptr"}: {Pkg: "runtime", Selector: "mapassign_fast64ptr"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapdelete_fast64"}:    {Pkg: "runtime", Selector: "mapdelete_fast64"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess1_faststr"}:  {Pkg: "runtime", Selector: "mapaccess1_faststr"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapaccess2_faststr"}:  {Pkg: "runtime", Selector: "mapaccess2_faststr"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapassign_faststr"}:   {Pkg: "runtime", Selector: "mapassign_faststr"},
	{Pkg: "internal/runtime/maps", Selector: "runtime_mapdelete_faststr"}:   {Pkg: "runtime", Selector: "mapdelete_faststr"},
}

var go126AcceptedNoBodyLinknames = map[packageSelector]bool{
	{Pkg: "crypto/internal/fips140hash", Selector: "sha3Unwrap"}: true,
	{Pkg: "internal/runtime/maps", Selector: "typeString"}:       true,
	{Pkg: "internal/runtime/maps", Selector: "fatal"}:            true,
	{Pkg: "internal/runtime/maps", Selector: "rand"}:             true,
	{Pkg: "internal/runtime/maps", Selector: "typedmemmove"}:     true,
	{Pkg: "internal/runtime/maps", Selector: "typedmemclr"}:      true,
	{Pkg: "internal/runtime/maps", Selector: "newarray"}:         true,
	{Pkg: "internal/runtime/maps", Selector: "newobject"}:        true,
}
