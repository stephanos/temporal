package gomadfs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"internal/gomadsim"
)

func TestFilesystemProcessTransportPartialResultsAndMappingCache(t *testing.T) {
	if os.Getenv("GOMAD_TEST_FILESYSTEM_CHILD") == "1" {
		testFilesystemProcessClient(t)
		return
	}
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	responseRead, responseWrite, err := os.Pipe()
	if err != nil {
		requestRead.Close()
		requestWrite.Close()
		t.Fatal(err)
	}
	for _, file := range []*os.File{requestRead, requestWrite, responseRead, responseWrite} {
		t.Cleanup(func() { file.Close() })
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFilesystemProcessTransportPartialResultsAndMappingCache$", "-test.v")
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOMAD3_SIMULATION_") || strings.HasPrefix(entry, "GOMADSEED=") || strings.HasPrefix(entry, "GOMAD_TEST_FILESYSTEM_CHILD=") {
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "GOMAD_TEST_FILESYSTEM_CHILD=1", "GOMAD3_SIMULATION_ROLE=node", "GOMAD3_SIMULATION_MODEL_REQUEST_FD=3", "GOMAD3_SIMULATION_MODEL_RESPONSE_FD=4")
	command.ExtraFiles = []*os.File{requestWrite, responseRead}
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	requestWrite.Close()
	responseRead.Close()
	served := make(chan error, 1)
	go func() { served <- serveFilesystemProcessFixture(requestRead, responseWrite) }()
	waitErr := command.Wait()
	requestRead.Close()
	responseWrite.Close()
	serveErr := <-served
	if waitErr != nil || serveErr != nil {
		t.Fatalf("process client=%v fixture=%v: %s", waitErr, serveErr, output.Bytes())
	}
}

func serveFilesystemProcessFixture(source io.Reader, destination io.Writer) error {
	cases := []struct {
		command processVolumeCommand
		result  processVolumeResult
	}{
		{processVolumeCommand{Operation: processVolumeOpenOp, Path: "/file", OpenFlags: OpenFlags{Read: true, Write: true}}, processVolumeResult{Handle: 31, Path: "/file"}},
		{processVolumeCommand{Operation: processVolumeHandleReadAtOp, Handle: 31, Offset: 2, ReadLength: 3}, processVolumeResult{BytesTransferred: 2, Data: []byte("cd"), Err: io.EOF}},
		{processVolumeCommand{Operation: processVolumeHandleReadOp, Handle: 31, ReadLength: 3}, processVolumeResult{BytesTransferred: 1, Data: []byte("abc")}},
		{processVolumeCommand{Operation: processVolumeHandleReadOp, Handle: 31, ReadLength: 1}, processVolumeResult{BytesTransferred: 2, Data: []byte("ab")}},
		{processVolumeCommand{Operation: processVolumeHandleWriteAtOp, Handle: 31, Offset: 1, Data: []byte("abc")}, processVolumeResult{BytesTransferred: 1, Err: syscall.ENOSPC}},
		{processVolumeCommand{Operation: processVolumeHandleWriteOp, Handle: 31, Data: []byte("abc")}, processVolumeResult{BytesTransferred: 4}},
		{processVolumeCommand{Operation: processVolumeHandleReadDirOp, Handle: 31, DirectoryCount: 1}, processVolumeResult{Err: io.EOF}},
		{processVolumeCommand{Operation: processVolumeHandleMapOp, Handle: 31, MapLength: 3}, processVolumeResult{Handle: 32}},
		{processVolumeCommand{Operation: processVolumeMappingBytesOp, Handle: 32}, processVolumeResult{Data: []byte("abc")}},
		{processVolumeCommand{Operation: processVolumeMappingCloseOp, Handle: 32}, processVolumeResult{Err: syscall.EIO}},
		{processVolumeCommand{Operation: processVolumeMappingCloseOp, Handle: 32}, processVolumeResult{}},
		{processVolumeCommand{Operation: processVolumeHandleMapOp, Handle: 31, MapLength: 1}, processVolumeResult{Handle: 33}},
		{processVolumeCommand{Operation: processVolumeMappingBytesOp, Handle: 33}, processVolumeResult{}},
		{processVolumeCommand{Operation: processVolumeMappingBytesOp, Handle: 33}, processVolumeResult{}},
		{processVolumeCommand{Operation: processVolumeHandleMapOp, Handle: 31, MapLength: 3}, processVolumeResult{Handle: 34}},
		{processVolumeCommand{Operation: processVolumeMappingBytesOp, Handle: 34}, processVolumeResult{Data: []byte("xyz")}},
		{processVolumeCommand{Operation: processVolumeHandleCloseOp, Handle: 31}, processVolumeResult{Err: syscall.EIO}},
		{processVolumeCommand{Operation: processVolumeHandleCloseOp, Handle: 31}, processVolumeResult{}},
	}
	for index, test := range cases {
		var header [4]byte
		if _, err := io.ReadFull(source, header[:]); err != nil {
			return fmt.Errorf("request %d: %w", index, err)
		}
		length := binary.BigEndian.Uint32(header[:])
		if length == 0 || length > 1<<20 {
			return fmt.Errorf("fixture request length=%d", length)
		}
		encoded := make([]byte, length)
		if _, err := io.ReadFull(source, encoded); err != nil {
			return err
		}
		frame, err := gomadsim.DecodeModelTransportFrame(encoded)
		if err != nil {
			return err
		}
		want, err := encodeProcessVolumeCommand(test.command)
		if err != nil {
			return err
		}
		if frame.Node != "node" || frame.Incarnation != 1 || !bytes.Equal(frame.Payload, want) {
			return fmt.Errorf("request %d wrong domain/command: %+v", index, frame)
		}
		payload, ok := encodeProcessVolumeResponse(test.result)
		if !ok {
			return errors.New("fixture response encoding")
		}
		frame.Response, frame.Payload = true, payload
		encoded, err = gomadsim.EncodeModelTransportFrame(frame)
		if err != nil {
			return err
		}
		binary.BigEndian.PutUint32(header[:], uint32(len(encoded)))
		if _, err := destination.Write(append(header[:], encoded...)); err != nil {
			return err
		}
	}
	return nil
}

func testFilesystemProcessClient(t *testing.T) {
	t.Helper()
	if gomadsim.ProcessRole() != 2 {
		t.Fatal("real runtime process transport is required")
	}
	run := gomadsim.Begin(64, 1)
	domain := gomadsim.Register(run, "node", "10.0.0.1", 1)
	previous, ok := gomadsim.Enter(domain)
	if !ok {
		t.Fatal("enter process domain")
	}
	defer gomadsim.Leave(previous)
	defer gomadsim.Finish(run)
	file, err := processOpen("/file", OpenFlags{Read: true, Write: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 3)
	if n, err := file.ReadAt(buffer, 2); n != 2 || err != io.EOF || !bytes.Equal(buffer, []byte{'c', 'd', 0}) {
		t.Fatalf("partial read=%d,%v,%q", n, err, buffer)
	}
	for _, destination := range [][]byte{buffer, buffer[:1]} {
		if n, err := file.Read(destination); n != 0 || err != syscall.EIO {
			t.Fatalf("malformed read=%d,%v", n, err)
		}
	}
	if n, err := file.WriteAt([]byte("abc"), 1); n != 1 || err != syscall.ENOSPC {
		t.Fatalf("partial write=%d,%v", n, err)
	}
	if n, err := file.Write([]byte("abc")); n != 0 || err != syscall.EIO {
		t.Fatalf("malformed write=%d,%v", n, err)
	}
	if entries, err := file.ReadDir(1); entries == nil || len(entries) != 0 || err != io.EOF {
		t.Fatalf("process directory normalization=%v,%v", entries, err)
	}
	mapping, err := file.Map(0, 3, false)
	if err != nil {
		t.Fatal(err)
	}
	data, err := mapping.Bytes()
	if err != nil || string(data) != "abc" {
		t.Fatalf("mapping=%q,%v", data, err)
	}
	data[0] = 'X'
	if again, err := mapping.Bytes(); err != nil || string(again) != "Xbc" {
		t.Fatalf("cached mapping=%q,%v", again, err)
	}
	if err := mapping.Close(); err != syscall.EIO {
		t.Fatalf("failed close=%v", err)
	}
	if again, err := mapping.Bytes(); err != nil || string(again) != "Xbc" {
		t.Fatalf("cache lost after failed close=%q,%v", again, err)
	}
	if err := mapping.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := mapping.Bytes(); err != syscall.EINVAL {
		t.Fatalf("closed mapping=%v", err)
	}
	empty, err := file.Map(0, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if data, err := empty.Bytes(); data != nil || err != nil {
			t.Fatalf("nil cache=%v,%v", data, err)
		}
	}
	cached, err := file.Map(0, 3, false)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := cached.Bytes(); err != nil || string(data) != "xyz" {
		t.Fatalf("cache before revoke=%q,%v", data, err)
	}
	if err := file.Close(); err != syscall.EIO {
		t.Fatalf("failed handle close=%v", err)
	}
	if file.Path() != "/file" {
		t.Fatalf("path=%q", file.Path())
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Map(-1, 0, true); err != syscall.ENOTSUP {
		t.Fatalf("writable precedence=%v", err)
	}
	if _, err := file.Read(buffer); err != ErrClosed {
		t.Fatalf("closed handle=%v", err)
	}
	if !gomadsim.Revoke(domain) {
		t.Fatal("revoke")
	}
	if data, err := cached.Bytes(); err != nil || string(data) != "xyz" {
		t.Fatalf("cached revoked bytes=%q,%v", data, err)
	}
	if err := cached.Close(); err != syscall.ESTALE {
		t.Fatalf("revoked close=%v", err)
	}
	if _, err := empty.Bytes(); err != syscall.ESTALE {
		t.Fatalf("uncached stale mapping=%v", err)
	}
}

func TestFilesystemProcessRegistryRollbackAndSuccessfulClose(t *testing.T) {
	fs := New()
	processVolumeResources.Lock()
	saved, next := processVolumeResources.values, processVolumeResources.next
	processVolumeResources.values = make(map[uint64]processVolumeResource)
	processVolumeResources.Unlock()
	t.Cleanup(func() {
		processVolumeResources.Lock()
		processVolumeResources.values, processVolumeResources.next = saved, next
		processVolumeResources.Unlock()
	})
	open := processVolumeCommand{Operation: processVolumeOpenOp, Path: "/file", OpenFlags: OpenFlags{Read: true, Write: true, Create: true}}
	for index := uint64(1); index <= maximumHandles; index++ {
		processVolumeResources.values[index] = processVolumeResource{domain: 99}
	}
	if result := applyProcessVolumeOperation(1, fs, open); !errors.Is(result.Err, syscall.EMFILE) || fs.Statistics().OpenHandles != 0 {
		t.Fatalf("open rollback=%+v stats=%+v", result, fs.Statistics())
	}
	clear(processVolumeResources.values)
	opened := applyProcessVolumeOperation(1, fs, open)
	if opened.Err != nil {
		t.Fatal(opened.Err)
	}
	defer removeProcessVolumeResource(opened.Handle)
	for index := uint64(1); len(processVolumeResources.values) < maximumHandles; index++ {
		if index != opened.Handle {
			processVolumeResources.values[index] = processVolumeResource{domain: 99}
		}
	}
	if result := applyProcessVolumeOperation(1, fs, processVolumeCommand{Operation: processVolumeHandleMapOp, Handle: opened.Handle, MapLength: 3}); !errors.Is(result.Err, syscall.EMFILE) || fs.Statistics().MappedBytes != 0 {
		t.Fatalf("map rollback=%+v stats=%+v", result, fs.Statistics())
	}
	for handle, resource := range processVolumeResources.values {
		if resource.domain == 99 {
			delete(processVolumeResources.values, handle)
		}
	}
	mapped := applyProcessVolumeOperation(1, fs, processVolumeCommand{Operation: processVolumeHandleMapOp, Handle: opened.Handle, MapLength: 3})
	if mapped.Err != nil {
		t.Fatal(mapped.Err)
	}
	defer removeProcessVolumeResource(mapped.Handle)
	for _, command := range []processVolumeCommand{{Operation: processVolumeHandleReadOp, Handle: opened.Handle}, {Operation: processVolumeMappingBytesOp, Handle: mapped.Handle}} {
		if result := applyProcessVolumeOperation(2, fs, command); result.Err != syscall.ESTALE {
			t.Fatalf("wrong domain=%+v", result)
		}
	}
	if result := applyProcessVolumeOperation(1, fs, processVolumeCommand{Operation: processVolumeHandleReadOp, Handle: mapped.Handle}); result.Err != syscall.ENOTSUP {
		t.Fatalf("wrong kind=%+v", result)
	}
	fs.mu.Lock()
	fs.unavailable = syscall.EIO
	fs.mu.Unlock()
	for _, command := range []processVolumeCommand{{Operation: processVolumeHandleCloseOp, Handle: opened.Handle}, {Operation: processVolumeMappingCloseOp, Handle: mapped.Handle}} {
		if result := applyProcessVolumeOperation(1, fs, command); result.Err != syscall.EIO {
			t.Fatalf("failed close=%+v", result)
		}
		if _, ok := processVolumeResourceFor(1, command.Handle); !ok {
			t.Fatal("failed close removed resource")
		}
	}
	fs.mu.Lock()
	fs.unavailable = nil
	fs.mu.Unlock()
	for _, command := range []processVolumeCommand{{Operation: processVolumeHandleCloseOp, Handle: opened.Handle}, {Operation: processVolumeMappingCloseOp, Handle: mapped.Handle}} {
		if result := applyProcessVolumeOperation(1, fs, command); result.Err != nil {
			t.Fatal(result.Err)
		}
		if _, ok := processVolumeResourceFor(1, command.Handle); ok {
			t.Fatal("successful close retained resource")
		}
	}
	first := applyProcessVolumeOperation(1, fs, open)
	second := applyProcessVolumeOperation(2, fs, open)
	if first.Err != nil || second.Err != nil {
		t.Fatalf("reopen=%+v,%+v", first, second)
	}
	defer removeProcessVolumeResource(second.Handle)
	revokeProcessVolumeResources(1)
	if result := applyProcessVolumeOperation(1, fs, processVolumeCommand{Operation: processVolumeHandleStatOp, Handle: first.Handle}); result.Err != syscall.ESTALE {
		t.Fatalf("revoked domain=%+v", result)
	}
	if result := applyProcessVolumeOperation(2, fs, processVolumeCommand{Operation: processVolumeHandleStatOp, Handle: second.Handle}); result.Err != nil {
		t.Fatalf("other domain revoked=%+v", result)
	}
}
