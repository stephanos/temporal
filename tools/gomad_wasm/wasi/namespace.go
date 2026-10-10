package wasi

import (
	"encoding/binary"
	"math"
	"path"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
)

const (
	errnoBadf            uint16 = 8
	errnoExist           uint16 = 20
	errnoInval           uint16 = 28
	errnoIsdir           uint16 = 31
	errnoNoent           uint16 = 44
	errnoNotdir          uint16 = 54
	errnoNotempty        uint16 = 55
	errnoNotsup          uint16 = 58
	errnoRofs            uint16 = 69
	errnoNotcapable      uint16 = 76
	rightRead            uint64 = 1 << 1
	rightSeek            uint64 = 1 << 2
	rightSetFlags        uint64 = 1 << 3
	rightSync            uint64 = 1 << 4
	rightWrite           uint64 = 1 << 6
	rightCreateDirectory uint64 = 1 << 9
	rightCreateFile      uint64 = 1 << 10
	rightPathSetSize     uint64 = 1 << 19
	rightPathOpen        uint64 = 1 << 13
	rightReaddir         uint64 = 1 << 14
	rightRenameSource    uint64 = 1 << 16
	rightRenameTarget    uint64 = 1 << 17
	rightPathStat        uint64 = 1 << 18
	rightFileStat        uint64 = 1 << 21
	rightSetSize         uint64 = 1 << 22
	rightRemoveDirectory uint64 = 1 << 25
	rightUnlink          uint64 = 1 << 26
	rightPoll            uint64 = 1 << 27
	rightsAll            uint64 = (1 << 29) - 1
)

type node struct {
	ino                 uint64
	directory, readonly bool
	data                []byte
	children            map[string]*node
	modified            uint64
}
type descriptor struct {
	node               *node
	offset             uint64
	rights, inheriting uint64
	flags              uint16
	preopen            string
	std                uint32
}
type namespace struct {
	root                    *node
	descriptors             map[uint32]*descriptor
	nextFD                  uint32
	nextInode, files, bytes uint64
}
type fdInput struct {
	FD uint32 `json:"fd"`
}
type fdLengthInput struct {
	FD     uint32 `json:"fd"`
	Length uint32 `json:"length"`
}
type pathInput struct {
	FD   uint32 `json:"fd"`
	Path []byte `json:"path_base64"`
}
type openInput struct {
	FD         uint32 `json:"fd"`
	Path       []byte `json:"path_base64"`
	Dirflags   uint32 `json:"dirflags"`
	Oflags     uint16 `json:"oflags"`
	Base       uint64 `json:"rights_base"`
	Inheriting uint64 `json:"rights_inheriting"`
	Flags      uint16 `json:"fdflags"`
}
type fdOutput struct {
	FD uint32 `json:"fd"`
}
type fdstat struct {
	Filetype   uint8  `json:"filetype"`
	Flags      uint16 `json:"flags"`
	Base       uint64 `json:"rights_base"`
	Inheriting uint64 `json:"rights_inheriting"`
}
type filestat struct {
	Dev      uint64 `json:"dev"`
	Ino      uint64 `json:"ino"`
	Filetype uint8  `json:"filetype"`
	Nlink    uint64 `json:"nlink"`
	Size     uint64 `json:"size"`
	Atim     uint64 `json:"atim"`
	Mtim     uint64 `json:"mtim"`
	Ctim     uint64 `json:"ctim"`
}

func newNamespace(config Config) (*namespace, error) {
	ns := &namespace{descriptors: map[uint32]*descriptor{}, nextFD: 3, nextInode: 1}
	ns.root = ns.newNode(true, true)
	preopens := slices.Clone(config.WritableDirectories)
	var capturedRoots []string
	var snapshot readonlymount.Snapshot
	if config.CapturedInputs.Manifest.Schema != "" {
		mappings, _, captured, err := readonlymount.DecodeCapturedInputs(config.CapturedInputs.Manifest, config.CapturedInputs.Descriptor, func(name string, size uint64) ([]byte, error) {
			data, ok := config.CapturedInputs.Payloads[name]
			if !ok || uint64(len(data)) != size {
				return nil, invalid("captured payload")
			}
			return slices.Clone(data), nil
		})
		if err != nil {
			return nil, err
		}
		snapshot = captured
		for _, mapping := range mappings {
			preopens = append(preopens, mapping.Target)
			capturedRoots = append(capturedRoots, mapping.Target)
		}
	} else if len(config.CapturedInputs.Descriptor) != 0 || len(config.CapturedInputs.Payloads) != 0 {
		return nil, invalid("captured manifest required")
	}
	capturedNodes := map[string]*node{}
	for _, entry := range snapshot.Entries {
		parent := ns.ensureDirectories(path.Dir(entry.Path), true)
		if parent == nil || parent.children[path.Base(entry.Path)] != nil {
			return nil, invalid("captured tree")
		}
		n := ns.newNode(entry.Kind == readonlymount.KindDirectory, true)
		n.data = slices.Clone(entry.Data)
		ns.bytes += uint64(len(n.data))
		parent.children[path.Base(entry.Path)] = n
		capturedNodes[entry.Path] = n
	}
	for _, entry := range snapshot.Entries {
		n := capturedNodes[entry.Path]
		if n.directory {
			if len(n.children) != len(entry.Children) {
				return nil, invalid("capture must contain complete directory inventory")
			}
			for _, child := range entry.Children {
				found := n.children[child.Name]
				if found == nil || found.directory != (child.Kind == readonlymount.KindDirectory) {
					return nil, invalid("captured child inventory")
				}
			}
		}
	}
	for _, directory := range config.WritableDirectories {
		for _, root := range capturedRoots {
			if root == "/" || directory == root || strings.HasPrefix(directory, root+"/") {
				return nil, invalid("writable mount overlaps captured input")
			}
		}
		n := ns.ensureDirectories(directory, false)
		if n == nil {
			return nil, invalid("writable directory beneath file")
		}
		n.readonly = false
	}
	cwd := ns.lookup(config.WorkingDirectory)
	if cwd == nil || !cwd.directory {
		return nil, invalid("working directory absent from captured namespace")
	}
	ns.bytes += uint64(len(config.Stdin))
	if ns.files > config.Limits.Files || ns.bytes > config.Limits.FilesystemBytes {
		return nil, capacity("initial-namespace")
	}
	for fd := uint32(0); fd < 3; fd++ {
		n := &node{ino: ns.nextInode, data: []byte{}}
		ns.nextInode++
		rights := rightWrite | rightSetFlags | rightFileStat | rightPoll
		if fd == 0 {
			n.data = slices.Clone(config.Stdin)
			rights = rightRead | rightSetFlags | rightFileStat | rightPoll
		}
		ns.descriptors[fd] = &descriptor{node: n, rights: rights, std: fd + 1}
	}
	slices.Sort(preopens)
	preopens = slices.Compact(preopens)
	if uint64(len(preopens))+3 > config.Limits.Descriptors {
		return nil, capacity("preopen-descriptors")
	}
	for _, name := range preopens {
		n := ns.lookup(name)
		if n == nil || !n.directory {
			return nil, invalid("preopen directory")
		}
		ns.descriptors[ns.nextFD] = &descriptor{node: n, rights: rightsAll, inheriting: rightsAll, preopen: name}
		ns.nextFD++
	}
	return ns, nil
}
func (ns *namespace) newNode(directory, readonly bool) *node {
	n := &node{ino: ns.nextInode, directory: directory, readonly: readonly, data: []byte{}}
	if directory {
		n.children = map[string]*node{}
	}
	ns.nextInode++
	ns.files++
	return n
}
func (ns *namespace) ensureDirectories(name string, readonly bool) *node {
	n := ns.root
	for _, component := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		if component == "" {
			continue
		}
		if !n.directory {
			return nil
		}
		next := n.children[component]
		if next == nil {
			next = ns.newNode(true, readonly)
			n.children[component] = next
		}
		n = next
	}
	return n
}
func (ns *namespace) lookup(name string) *node {
	n := ns.root
	for _, component := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		if component == "" {
			continue
		}
		if !n.directory {
			return nil
		}
		n = n.children[component]
		if n == nil {
			return nil
		}
	}
	return n
}
func (e *Environment) getFD(fd uint32, right uint64) (*descriptor, uint16) {
	d := e.namespace.descriptors[fd]
	if d == nil {
		return nil, errnoBadf
	}
	if d.rights&right != right {
		return nil, errnoNotcapable
	}
	return d, 0
}
func resolvePath(d *descriptor, name []byte) (*node, string, uint16) {
	if !d.node.directory {
		return nil, "", errnoNotdir
	}
	text := string(name)
	if text == "" {
		return nil, "", errnoNoent
	}
	if len(text) > 4096 || strings.HasPrefix(text, "/") || strings.IndexByte(text, 0) >= 0 {
		return nil, "", errnoNotcapable
	}
	depth := 0
	for _, component := range strings.Split(text, "/") {
		switch component {
		case "", ".":
		case "..":
			if depth == 0 {
				return nil, "", errnoNotcapable
			}
			depth--
		default:
			depth++
		}
	}
	stack := []string{}
	nodes := []*node{d.node}
	for _, component := range strings.Split(text, "/") {
		if component == "" {
			continue
		}
		current := nodes[len(nodes)-1]
		if current == nil {
			return nil, "", errnoNoent
		}
		if !current.directory {
			return nil, "", errnoNotdir
		}
		switch component {
		case ".":
			continue
		case "..":
			if len(stack) == 0 {
				return nil, "", errnoNotcapable
			}
			stack = stack[:len(stack)-1]
			nodes = nodes[:len(nodes)-1]
		default:
			stack = append(stack, component)
			nodes = append(nodes, current.children[component])
		}
	}
	if len(stack) == 0 {
		return d.node, "", 0
	}
	if final := nodes[len(nodes)-1]; strings.HasSuffix(text, "/") && final != nil && !final.directory {
		return nil, "", errnoNotdir
	}
	return nodes[len(nodes)-2], stack[len(stack)-1], 0
}
func childAt(parent *node, name string) *node {
	if name == "" {
		return parent
	}
	return parent.children[name]
}
func nodeType(n *node) uint8 {
	if n.directory {
		return 3
	}
	return 4
}
func (e *Environment) stat(n *node) filestat {
	timestamp := e.config.Clock.EpochNanos + n.modified
	return filestat{1, n.ino, nodeType(n), 1, uint64(len(n.data)), timestamp, timestamp, timestamp}
}
func (e *Environment) namespaceCall(call Call) (any, uint16, string, error) {
	switch call.Op {
	case "fd_prestat_get", "fd_fdstat_get", "fd_filestat_get", "fd_close", "fd_sync":
		var input fdInput
		if err := decodeInput(call.Input, &input, "fd"); err != nil {
			return nil, 0, "", err
		}
		right := uint64(0)
		if call.Op == "fd_filestat_get" {
			right = rightFileStat
		}
		if call.Op == "fd_sync" {
			right = rightSync
		}
		d, errno := e.getFD(input.FD, right)
		if errno != 0 {
			return nil, errno, "", nil
		}
		switch call.Op {
		case "fd_prestat_get":
			if d.preopen == "" {
				return nil, errnoBadf, "", nil
			}
			return struct {
				Length uint32 `json:"name_length"`
			}{uint32(len(d.preopen))}, 0, "", nil
		case "fd_fdstat_get":
			filetype := nodeType(d.node)
			if d.std != 0 {
				filetype = 2
			}
			return fdstat{filetype, d.flags, d.rights, d.inheriting}, 0, "", nil
		case "fd_filestat_get":
			return e.stat(d.node), 0, "", nil
		case "fd_close":
			delete(e.namespace.descriptors, input.FD)
		}
		return struct{}{}, 0, "", nil
	case "fd_prestat_dir_name":
		var input fdLengthInput
		if err := decodeInput(call.Input, &input, "fd", "length"); err != nil {
			return nil, 0, "", err
		}
		d, errno := e.getFD(input.FD, 0)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if d.preopen == "" {
			return nil, errnoBadf, "", nil
		}
		if int(input.Length) != len(d.preopen) {
			return nil, errnoInval, "", nil
		}
		return bytesOutput{[]byte(d.preopen)}, 0, "", nil
	case "fd_fdstat_set_flags":
		var input struct {
			FD    uint32 `json:"fd"`
			Flags uint16 `json:"flags"`
		}
		if err := decodeInput(call.Input, &input, "fd", "flags"); err != nil {
			return nil, 0, "", err
		}
		d, errno := e.getFD(input.FD, rightSetFlags)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if input.Flags&^uint16(1|4) != 0 {
			return nil, 0, "descriptor flags outside volatile profile", nil
		}
		d.flags = input.Flags
		return struct{}{}, 0, "", nil
	case "fd_read", "fd_pread":
		var input struct {
			FD     uint32 `json:"fd"`
			Length uint32 `json:"length"`
			Offset uint64 `json:"offset"`
		}
		keys := []string{"fd", "length"}
		if call.Op == "fd_pread" {
			keys = append(keys, "offset")
		}
		if err := decodeInput(call.Input, &input, keys...); err != nil {
			return nil, 0, "", err
		}
		right := rightRead
		if call.Op == "fd_pread" {
			right |= rightSeek
		}
		d, errno := e.getFD(input.FD, right)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if d.node.directory {
			return nil, errnoIsdir, "", nil
		}
		offset := d.offset
		if call.Op == "fd_pread" {
			offset = input.Offset
		}
		if offset > math.MaxInt64 {
			return nil, errnoInval, "", nil
		}
		data := []byte{}
		if offset < uint64(len(d.node.data)) {
			end := min(uint64(len(d.node.data)), offset+uint64(input.Length))
			data = slices.Clone(d.node.data[offset:end])
		}
		if call.Op == "fd_read" {
			d.offset += uint64(len(data))
		}
		return bytesOutput{data}, 0, "", nil
	case "fd_write", "fd_pwrite":
		var input struct {
			FD     uint32 `json:"fd"`
			Data   []byte `json:"data_base64"`
			Offset uint64 `json:"offset"`
		}
		keys := []string{"fd", "data_base64"}
		if call.Op == "fd_pwrite" {
			keys = append(keys, "offset")
		}
		if err := decodeInput(call.Input, &input, keys...); err != nil {
			return nil, 0, "", err
		}
		if len(input.Data) > bufferLimit {
			return nil, 0, "", capacity("write-buffer")
		}
		if input.FD == 1 || input.FD == 2 {
			if e.offeredOutput+uint64(len(input.Data)) > e.config.Limits.OutputBytes {
				return nil, 0, "", capacity("offered-output-bytes")
			}
			e.offeredOutput += uint64(len(input.Data))
		}
		right := rightWrite
		if call.Op == "fd_pwrite" {
			right |= rightSeek
		}
		d, errno := e.getFD(input.FD, right)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if d.node.directory {
			return nil, errnoIsdir, "", nil
		}
		if d.node.readonly {
			return nil, errnoRofs, "", nil
		}
		if d.std == 2 {
			e.stdout = append(e.stdout, input.Data...)
			return writtenOutput{uint32(len(input.Data))}, 0, "", nil
		}
		if d.std == 3 {
			e.stderr = append(e.stderr, input.Data...)
			return writtenOutput{uint32(len(input.Data))}, 0, "", nil
		}
		offset := d.offset
		if call.Op == "fd_pwrite" {
			offset = input.Offset
		}
		if d.flags&1 != 0 {
			offset = uint64(len(d.node.data))
		}
		if offset > math.MaxInt64 || uint64(len(input.Data)) > math.MaxInt64-offset {
			return nil, errnoInval, "", nil
		}
		size := offset + uint64(len(input.Data))
		if len(input.Data) == 0 {
			size = uint64(len(d.node.data))
		}
		if err := e.resize(d.node, max(size, uint64(len(d.node.data)))); err != nil {
			return nil, 0, "", err
		}
		if len(input.Data) > 0 {
			copy(d.node.data[offset:], input.Data)
			d.node.modified = e.now
		}
		if call.Op == "fd_write" {
			d.offset = offset + uint64(len(input.Data))
		}
		return writtenOutput{uint32(len(input.Data))}, 0, "", nil
	case "fd_filestat_set_size":
		var input struct {
			FD   uint32 `json:"fd"`
			Size uint64 `json:"size"`
		}
		if err := decodeInput(call.Input, &input, "fd", "size"); err != nil {
			return nil, 0, "", err
		}
		d, errno := e.getFD(input.FD, rightSetSize)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if d.node.directory {
			return nil, errnoIsdir, "", nil
		}
		if d.node.readonly {
			return nil, errnoRofs, "", nil
		}
		if err := e.resize(d.node, input.Size); err != nil {
			return nil, 0, "", err
		}
		d.node.modified = e.now
		return struct{}{}, 0, "", nil
	case "fd_seek":
		var input struct {
			FD     uint32 `json:"fd"`
			Offset int64  `json:"offset"`
			Whence uint8  `json:"whence"`
		}
		if err := decodeInput(call.Input, &input, "fd", "offset", "whence"); err != nil {
			return nil, 0, "", err
		}
		d, errno := e.getFD(input.FD, rightSeek)
		if errno != 0 {
			return nil, errno, "", nil
		}
		if d.node.directory || d.std != 0 {
			return nil, errnoInval, "", nil
		}
		base := uint64(0)
		switch input.Whence {
		case 0:
		case 1:
			base = d.offset
		case 2:
			base = uint64(len(d.node.data))
		default:
			return nil, errnoInval, "", nil
		}
		if input.Offset >= 0 {
			if base > math.MaxInt64-uint64(input.Offset) {
				return nil, errnoInval, "", nil
			}
			base += uint64(input.Offset)
		} else {
			delta := uint64(-(input.Offset + 1)) + 1
			if delta > base {
				return nil, errnoInval, "", nil
			}
			base -= delta
		}
		d.offset = base
		return offsetOutput{base}, 0, "", nil
	case "fd_readdir":
		return e.readdir(call)
	case "path_open":
		return e.open(call)
	case "path_create_directory", "path_remove_directory", "path_unlink_file", "path_filestat_get":
		return e.pathOperation(call)
	case "path_rename":
		return e.rename(call)
	case "clock_time_get", "poll_oneoff":
		return e.clockCall(call)
	default:
		return nil, 0, "operation outside stock WASI profile: " + call.Op, nil
	}
}
func (e *Environment) resize(n *node, size uint64) error {
	if size > e.config.Limits.FilesystemBytes || e.namespace.bytes-uint64(len(n.data)) > e.config.Limits.FilesystemBytes-size {
		return capacity("filesystem-bytes")
	}
	old := uint64(len(n.data))
	if size > old {
		n.data = append(n.data, make([]byte, size-old)...)
	} else {
		n.data = n.data[:size]
	}
	e.namespace.bytes = e.namespace.bytes - old + size
	return nil
}
func (e *Environment) open(call Call) (any, uint16, string, error) {
	var input openInput
	if err := decodeInput(call.Input, &input, "fd", "path_base64", "dirflags", "oflags", "rights_base", "rights_inheriting", "fdflags"); err != nil {
		return nil, 0, "", err
	}
	d, errno := e.getFD(input.FD, rightPathOpen)
	if errno != 0 {
		return nil, errno, "", nil
	}
	if input.Dirflags&^uint32(1) != 0 || input.Oflags&^uint16(15) != 0 || input.Base&^rightsAll != 0 || input.Inheriting&^rightsAll != 0 {
		return nil, errnoInval, "", nil
	}
	if input.Flags&^uint16(1|4) != 0 {
		return nil, 0, "open flags outside volatile profile", nil
	}
	if input.Oflags&3 == 3 {
		return nil, errnoInval, "", nil
	}
	if input.Oflags&1 != 0 && d.rights&rightCreateFile == 0 || input.Oflags&8 != 0 && d.rights&rightPathSetSize == 0 {
		return nil, errnoNotcapable, "", nil
	}
	if input.Base&d.inheriting != input.Base || input.Inheriting&d.inheriting != input.Inheriting {
		return nil, errnoNotcapable, "", nil
	}
	parent, name, errno := resolvePath(d, input.Path)
	if errno != 0 {
		return nil, errno, "", nil
	}
	n := childAt(parent, name)
	if n == nil && strings.HasSuffix(string(input.Path), "/") {
		return nil, errnoNotdir, "", nil
	}
	if n != nil && input.Oflags&5 == 5 {
		return nil, errnoExist, "", nil
	}
	if n == nil && input.Oflags&1 == 0 {
		return nil, errnoNoent, "", nil
	}
	if n != nil && input.Oflags&2 != 0 && !n.directory {
		return nil, errnoNotdir, "", nil
	}
	if n != nil && n.directory && input.Oflags&8 != 0 {
		return nil, errnoIsdir, "", nil
	}
	if parent.readonly && (n == nil || input.Oflags&8 != 0 || input.Base&rightWrite != 0) {
		return nil, errnoRofs, "", nil
	}
	if input.Oflags&8 != 0 && input.Base&rightSetSize == 0 {
		return nil, errnoNotcapable, "", nil
	}
	ns := e.namespace
	if uint64(len(ns.descriptors)) >= e.config.Limits.Descriptors || ns.nextFD == math.MaxUint32 {
		return nil, 0, "", capacity("descriptors")
	}
	if n == nil && ns.files == e.config.Limits.Files {
		return nil, 0, "", capacity("files")
	}
	if n == nil {
		n = ns.newNode(false, false)
		n.modified = e.now
		parent.children[name] = n
	}
	if input.Oflags&8 != 0 {
		if err := e.resize(n, 0); err != nil {
			return nil, 0, "", err
		}
		n.modified = e.now
	}
	fd := ns.nextFD
	ns.nextFD++
	ns.descriptors[fd] = &descriptor{node: n, rights: input.Base, inheriting: input.Inheriting, flags: input.Flags}
	return fdOutput{fd}, 0, "", nil
}
func (e *Environment) pathOperation(call Call) (any, uint16, string, error) {
	var input struct {
		FD    uint32 `json:"fd"`
		Path  []byte `json:"path_base64"`
		Flags uint32 `json:"flags"`
	}
	keys := []string{"fd", "path_base64"}
	if call.Op == "path_filestat_get" {
		keys = append(keys, "flags")
	}
	if err := decodeInput(call.Input, &input, keys...); err != nil {
		return nil, 0, "", err
	}
	rights := map[string]uint64{"path_create_directory": rightCreateDirectory, "path_remove_directory": rightRemoveDirectory, "path_unlink_file": rightUnlink, "path_filestat_get": rightPathStat}
	d, errno := e.getFD(input.FD, rights[call.Op])
	if errno != 0 {
		return nil, errno, "", nil
	}
	if input.Flags&^uint32(1) != 0 {
		return nil, errnoInval, "", nil
	}
	parent, name, errno := resolvePath(d, input.Path)
	if errno != 0 {
		return nil, errno, "", nil
	}
	n := childAt(parent, name)
	if call.Op == "path_filestat_get" {
		if n == nil {
			return nil, errnoNoent, "", nil
		}
		return e.stat(n), 0, "", nil
	}
	if parent.readonly || n != nil && n.readonly {
		return nil, errnoRofs, "", nil
	}
	if call.Op == "path_create_directory" {
		if n != nil {
			return nil, errnoExist, "", nil
		}
		if e.namespace.files == e.config.Limits.Files {
			return nil, 0, "", capacity("files")
		}
		parent.children[name] = e.namespace.newNode(true, false)
		return struct{}{}, 0, "", nil
	}
	if n == nil {
		return nil, errnoNoent, "", nil
	}
	if name == "" {
		return nil, errnoNotcapable, "", nil
	}
	if call.Op == "path_remove_directory" {
		if !n.directory {
			return nil, errnoNotdir, "", nil
		}
		if len(n.children) != 0 {
			return nil, errnoNotempty, "", nil
		}
	} else if n.directory {
		return nil, errnoIsdir, "", nil
	}
	delete(parent.children, name)
	return struct{}{}, 0, "", nil
}
func (e *Environment) readdir(call Call) (any, uint16, string, error) {
	var input struct {
		FD     uint32 `json:"fd"`
		Length uint32 `json:"length"`
		Cookie uint64 `json:"cookie"`
	}
	if err := decodeInput(call.Input, &input, "fd", "length", "cookie"); err != nil {
		return nil, 0, "", err
	}
	d, errno := e.getFD(input.FD, rightReaddir)
	if errno != 0 {
		return nil, errno, "", nil
	}
	if !d.node.directory {
		return nil, errnoNotdir, "", nil
	}
	names := make([]string, 0, len(d.node.children))
	for name := range d.node.children {
		names = append(names, name)
	}
	slices.Sort(names)
	if input.Cookie > uint64(len(names)) {
		return nil, errnoInval, "", nil
	}
	data := []byte{}
	for i := input.Cookie; i < uint64(len(names)) && len(data) < int(input.Length); i++ {
		name := names[i]
		n := d.node.children[name]
		entry := make([]byte, 24+len(name))
		binary.LittleEndian.PutUint64(entry, i+1)
		binary.LittleEndian.PutUint64(entry[8:], n.ino)
		binary.LittleEndian.PutUint32(entry[16:], uint32(len(name)))
		entry[20] = nodeType(n)
		copy(entry[24:], name)
		data = append(data, entry[:min(len(entry), int(input.Length)-len(data))]...)
	}
	return bytesOutput{data}, 0, "", nil
}
func (e *Environment) rename(call Call) (any, uint16, string, error) {
	var input struct {
		FD      uint32 `json:"fd"`
		Path    []byte `json:"path_base64"`
		NewFD   uint32 `json:"new_fd"`
		NewPath []byte `json:"new_path_base64"`
	}
	if err := decodeInput(call.Input, &input, "fd", "path_base64", "new_fd", "new_path_base64"); err != nil {
		return nil, 0, "", err
	}
	source, errno := e.getFD(input.FD, rightRenameSource)
	if errno != 0 {
		return nil, errno, "", nil
	}
	target, errno := e.getFD(input.NewFD, rightRenameTarget)
	if errno != 0 {
		return nil, errno, "", nil
	}
	parent, name, errno := resolvePath(source, input.Path)
	if errno != 0 {
		return nil, errno, "", nil
	}
	newParent, newName, errno := resolvePath(target, input.NewPath)
	if errno != 0 {
		return nil, errno, "", nil
	}
	n := childAt(parent, name)
	if n == nil {
		return nil, errnoNoent, "", nil
	}
	if strings.HasSuffix(string(input.NewPath), "/") && !n.directory {
		return nil, errnoNotdir, "", nil
	}
	if name == "" || newName == "" {
		return nil, errnoNotcapable, "", nil
	}
	old := newParent.children[newName]
	if parent.readonly || newParent.readonly || n.readonly || old != nil && old.readonly {
		return nil, errnoRofs, "", nil
	}
	if parent == newParent && name == newName {
		return struct{}{}, 0, "", nil
	}
	if n.directory && containsNode(n, newParent) {
		return nil, errnoInval, "", nil
	}
	if old != nil {
		if old.directory != n.directory {
			if old.directory {
				return nil, errnoIsdir, "", nil
			}
			return nil, errnoNotdir, "", nil
		}
		if old.directory && len(old.children) > 0 {
			return nil, errnoNotempty, "", nil
		}
	}
	delete(parent.children, name)
	newParent.children[newName] = n
	return struct{}{}, 0, "", nil
}
func containsNode(root, query *node) bool {
	if root == query {
		return true
	}
	for _, child := range root.children {
		if child.directory && containsNode(child, query) {
			return true
		}
	}
	return false
}
