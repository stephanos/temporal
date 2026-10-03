package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	pebbleModulePath                       = "github.com/cockroachdb/pebble"
	pebbleVersion                          = "v0.0.0-20260703021901-41f35d3cb7df"
	pebbleSum                              = "h1:p7vkumDcPw0de7t8pYA95HPC4cYQZGDG6b57d4Om5cA="
	pebbleOriginalSourceInventorySHA256    = "sha256:38ac96801b64ddcf554d47bfee949983fcb232755a101321afa3ac33873c7193"
	pebbleReplacementSourceInventorySHA256 = "sha256:dcdc0a8dbb71945e676a1402a57ee2a6f29b060ea443c79e0b28d904718d9e03"
)

var pebblePreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:dc14e106cfe39a04f990140a53e6768ed57956894f7ca01602b7036d06e10807",
	"linux/amd64":  "sha256:0418ad6cf4a755d075fda10de95dd9f0b5a62c789f3ec1c76d230b8e12fcdb91",
}

var pebblePreparedSourceSetSHA256 = hostPin(pebblePreparedSourceSetSHA256ByHost)

// Pebble's native wrappers promote unsupported os.File methods even when the
// workload uses MemFS. Refusing their construction removes those callbacks.
var pebbleRewrites = []sourceRewrite{
	{path: "vfs/vfs.go", sourceSHA256: "sha256:279bf6c538a0f2bbab84f867cedb91f5ffc001ded69f918a0f7843c7dd5626c5", replacementSHA256: "sha256:6d3da4860e92b227d1e39aafed337647adde5182caaf48ada4d6c51c3d16244e", rewrites: []anchorRewrite{
		{anchor: []byte(`func (defaultFS) Create(name string, category DiskWriteCategory) (File, error) {
	const openFlags = os.O_RDWR | os.O_CREATE | os.O_EXCL | syscall.O_CLOEXEC

	osFile, err := os.OpenFile(name, openFlags, 0666)
	// If the file already exists, remove it and try again.
	//
	// NB: We choose to remove the file instead of truncating it, despite the
	// fact that we can't do so atomically, because it's more resistant to
	// misuse when using hard links.

	// We must loop in case another goroutine/thread/process is also
	// attempting to create the a file at the same path.
	for oserror.IsExist(err) {
		if removeErr := os.Remove(name); removeErr != nil && !oserror.IsNotExist(removeErr) {
			return wrapOSFile(osFile), errors.WithStack(removeErr)
		}
		osFile, err = os.OpenFile(name, openFlags, 0666)
	}
	return wrapOSFile(osFile), errors.WithStack(err)
}`), replacement: []byte(`func (defaultFS) Create(name string, category DiskWriteCategory) (File, error) {
	// If the file already exists, remove it and try again.
	//
	// NB: We choose to remove the file instead of truncating it, despite the
	// fact that we can't do so atomically, because it's more resistant to
	// misuse when using hard links.
	// We must loop in case another goroutine/thread/process is also
	// attempting to create the a file at the same path.
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (defaultFS) Link(oldname, newname string) error {
	return errors.WithStack(os.Link(oldname, newname))
}`), replacement: []byte(`func (defaultFS) Link(oldname, newname string) error {
	return errors.New("gomad: Pebble hard links are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (defaultFS) Open(name string, opts ...OpenOption) (File, error) {
	osFile, err := os.OpenFile(name, os.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	file := wrapOSFile(osFile)
	for _, opt := range opts {
		opt.Apply(file)
	}
	return file, nil
}`), replacement: []byte(`func (defaultFS) Open(name string, opts ...OpenOption) (File, error) {
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (defaultFS) OpenReadWrite(
	name string, category DiskWriteCategory, opts ...OpenOption,
) (File, error) {
	osFile, err := os.OpenFile(name, os.O_RDWR|syscall.O_CLOEXEC|os.O_CREATE, 0666)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	file := wrapOSFile(osFile)
	for _, opt := range opts {
		opt.Apply(file)
	}
	return file, nil
}`), replacement: []byte(`func (defaultFS) OpenReadWrite(
	name string, category DiskWriteCategory, opts ...OpenOption,
) (File, error) {
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (fs defaultFS) ReuseForWrite(
	oldname, newname string, category DiskWriteCategory,
) (File, error) {
	if err := fs.Rename(oldname, newname); err != nil {
		return nil, errors.WithStack(err)
	}
	f, err := os.OpenFile(newname, os.O_RDWR|os.O_CREATE|syscall.O_CLOEXEC, 0666)
	return wrapOSFile(f), errors.WithStack(err)
}`), replacement: []byte(`func (fs defaultFS) ReuseForWrite(
	oldname, newname string, category DiskWriteCategory,
) (File, error) {
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`	"syscall"
`), replacement: []byte(``)},
	}},
	{path: "vfs/default_unix.go", sourceSHA256: "sha256:eddb79c38f11844c43d0c8bbfbd62e88f0c31a4b8534d26060a2ee13d1a1b2d8", replacementSHA256: "sha256:a270f4419fc54c7ac09d27201557abc0e82ede1039c9175b0c82a2039b8aba24", rewrites: []anchorRewrite{
		{anchor: []byte(`func wrapOSFileImpl(osFile *os.File) File {
	return &unixFile{File: osFile, fd: osFile.Fd()}
}`), replacement: []byte(`func wrapOSFileImpl(osFile *os.File) File {
	if osFile == nil {
		return nil
	}
	panic("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (defaultFS) OpenDir(name string) (File, error) {
	f, err := os.OpenFile(name, syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &unixFile{f, InvalidFd}, nil
}`), replacement: []byte(`func (defaultFS) OpenDir(name string) (File, error) {
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
	}},
	{path: "vfs/default_linux.go", sourceSHA256: "sha256:9b2c9e213de1cba57afbf6bfab983ecfba58977933717e0bafb2754abd9cd1d2", replacementSHA256: "sha256:016cbff5dabf80f34d3abca234b2a06b26fe9aa3d50823a8596d44dcf2018874", rewrites: []anchorRewrite{
		{anchor: []byte(`func wrapOSFileImpl(f *os.File) File {
	lf := &linuxFile{File: f, fd: f.Fd()}
	if lf.fd != InvalidFd {
		lf.useSyncRange = isSyncRangeSupported(lf.fd)
	}
	return lf
}`), replacement: []byte(`func wrapOSFileImpl(f *os.File) File {
	if f == nil {
		return nil
	}
	panic("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
		{anchor: []byte(`func (defaultFS) OpenDir(name string) (File, error) {
	f, err := os.OpenFile(name, syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &linuxDir{f}, nil
}`), replacement: []byte(`func (defaultFS) OpenDir(name string) (File, error) {
	return nil, errors.New("gomad: Pebble OS-backed files are unavailable; inject a memory-backed FS")
}`)},
	}},
}

var pebbleAdapter = rewrittenModule{
	module: pebbleModulePath, version: pebbleVersion, sum: pebbleSum,
	cacheElements:                 []string{"github.com", "cockroachdb", "pebble@" + pebbleVersion},
	replacementDirectory:          "pebble",
	originalInventorySHA256:       pebbleOriginalSourceInventorySHA256,
	replacementInventorySHA256:    pebbleReplacementSourceInventorySHA256,
	preparedPackage:               pebbleModulePath + "/vfs",
	preparedSourceSetSHA256ByHost: pebblePreparedSourceSetSHA256ByHost,
	rewrites:                      pebbleRewrites,
}

func preparePebble(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, pebbleAdapter)
}
