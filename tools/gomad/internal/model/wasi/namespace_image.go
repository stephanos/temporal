package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"io"
	"math"
	"slices"
	"strings"
)

const namespaceChunkBytes = 64 << 10

type namespaceDeclaration struct {
	Chunks uint64 `json:"chunks"`
	Files  uint64 `json:"files"`
	Bytes  uint64 `json:"bytes"`
	SHA256 string `json:"sha256"`
}
type namespaceImageEntry struct {
	path string
	node *node
}

// The namespace has already been validated and cloned by NewEnvironment. This
// snapshot borrows its payloads through image carriage, before any callback; it
// never reopens host files.
type namespaceImage struct {
	entries                      []namespaceImageEntry
	writable, captured, preopens []string
	cwd                          string
	stdin                        []byte
	limits                       Limits
	epoch                        uint64
}

func helperOwnsNamespace(e *Environment) bool { return e.helperOwned || e.splitOwned }

func declareNamespaceImage(e *Environment) (*namespaceImage, *namespaceDeclaration, error) {
	image := &namespaceImage{writable: slices.Clone(e.config.WritableDirectories), cwd: e.config.WorkingDirectory, stdin: e.namespace.descriptors[0].node.data, limits: e.config.Limits, epoch: e.config.Clock.EpochNanos}
	stack := []namespaceImageEntry{{"/", e.namespace.root}}
	for len(stack) != 0 {
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if entry.node != e.namespace.root {
			image.entries = append(image.entries, entry)
		}
		for name, child := range entry.node.children {
			stack = append(stack, namespaceImageEntry{strings.TrimSuffix(entry.path, "/") + "/" + name, child})
		}
	}
	slices.SortFunc(image.entries, func(a, b namespaceImageEntry) int {
		if a.node.ino < b.node.ino {
			return -1
		}
		if a.node.ino > b.node.ino {
			return 1
		}
		return 0
	})
	for fd := uint32(3); fd < e.namespace.nextFD; fd++ {
		d := e.namespace.descriptors[fd]
		image.preopens = append(image.preopens, d.preopen)
		if d.node.readonly {
			image.captured = append(image.captured, d.preopen)
		}
	}
	counter := namespaceImageCounter{digest: sha256.New()}
	if err := image.write(&counter); err != nil {
		return nil, nil, err
	}
	return image, &namespaceDeclaration{counter.bytes/namespaceChunkBytes + boolUint64(counter.bytes%namespaceChunkBytes != 0), uint64(len(image.entries)) + 1, counter.bytes, hex.EncodeToString(counter.digest.Sum(nil))}, nil
}

func boolUint64(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func (image *namespaceImage) write(output io.Writer) error {
	var failure error
	write := func(data []byte) {
		if failure != nil {
			return
		}
		var count int
		count, failure = output.Write(data)
		if failure == nil && count != len(data) {
			failure = io.ErrShortWrite
		}
	}
	number := func(value uint64) { var data [8]byte; binary.LittleEndian.PutUint64(data[:], value); write(data[:]) }
	blob := func(data []byte) { number(uint64(len(data))); write(data) }
	text := func(value string) {
		number(uint64(len(value)))
		for len(value) != 0 && failure == nil {
			count := min(len(value), namespaceChunkBytes)
			write([]byte(value[:count]))
			value = value[count:]
		}
	}
	write([]byte("GMWN\x01\x00\x00\x00"))
	for _, value := range []uint64{image.limits.Files, image.limits.Descriptors, image.limits.FilesystemBytes, image.limits.OutputBytes, image.epoch, uint64(len(image.entries)), uint64(len(image.writable)), uint64(len(image.captured)), uint64(len(image.preopens))} {
		number(value)
	}
	for _, names := range [][]string{image.captured, image.writable, image.preopens} {
		for _, name := range names {
			text(name)
		}
	}
	text(image.cwd)
	blob(image.stdin)
	for _, entry := range image.entries {
		kind := byte(2)
		if entry.node.directory {
			kind = 1
		}
		write([]byte{kind, byte(boolUint64(entry.node.readonly))})
		text(entry.path)
		blob(entry.node.data)
	}
	write([]byte{0})
	return failure
}

type namespaceImageCounter struct {
	digest hash.Hash
	bytes  uint64
}

func (counter *namespaceImageCounter) Write(data []byte) (int, error) {
	if uint64(len(data)) > math.MaxUint64-counter.bytes {
		return 0, invalid("namespace image size overflow")
	}
	count, err := counter.digest.Write(data)
	counter.bytes += uint64(count)
	return count, err
}

type namespaceChunkWriter struct {
	sink     *frameSink
	data     []byte
	sequence uint64
}

func (writer *namespaceChunkWriter) flush() error {
	if len(writer.data) == 0 {
		return nil
	}
	if err := writer.sink.send(struct {
		Type     string `json:"type"`
		Sequence uint64 `json:"sequence"`
		Data     []byte `json:"data_base64"`
	}{"namespace_input", writer.sequence, writer.data}); err != nil {
		return err
	}
	writer.sequence++
	writer.data = writer.data[:0]
	return nil
}
func (writer *namespaceChunkWriter) Write(data []byte) (int, error) {
	written := 0
	for len(data) != 0 {
		count := min(len(data), namespaceChunkBytes-len(writer.data))
		writer.data = append(writer.data, data[:count]...)
		data = data[count:]
		written += count
		if len(writer.data) == namespaceChunkBytes {
			if err := writer.flush(); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}
func (sink *frameSink) sendHelperNamespace() error {
	if !sink.environment.namespaceOwned {
		return nil
	}
	if sink.namespaceImage == nil {
		return invalid("missing owned namespace image")
	}
	writer := namespaceChunkWriter{sink: sink, data: make([]byte, 0, namespaceChunkBytes)}
	if err := sink.namespaceImage.write(&writer); err != nil {
		return err
	}
	if err := writer.flush(); err != nil {
		return err
	}
	if err := sink.send(struct {
		Type string `json:"type"`
	}{"namespace_inputs_end"}); err != nil {
		return err
	}
	if sink.storageImage != nil {
		if err := sink.sendStorageInput("storage_image", sink.storageImage); err != nil {
			return err
		}
	}
	if sink.storageContext != nil {
		return sink.sendStorageInput("storage_context", sink.storageContext)
	}
	return nil
}
