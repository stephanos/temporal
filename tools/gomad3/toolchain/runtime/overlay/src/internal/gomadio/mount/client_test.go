package mount_test

import (
	"bytes"
	"errors"
	"syscall"
	"testing"

	"internal/gomadio/mount"
	"internal/gomadwire"
)

func TestClientLookupOwnsFramingAndOrdinals(t *testing.T) {
	limits := gomadwire.MountLimits{PathBytes: 4096, FileBytes: 16 << 20, DirectoryEntries: 100_000}
	var responses bytes.Buffer
	for ordinal, data := range []string{"first", "second"} {
		if err := gomadwire.WriteMountResponse(&responses, gomadwire.MountResponse{
			Ordinal: uint64(ordinal), Status: gomadwire.MountStatusOK,
			Entry: gomadwire.MountEntry{Kind: gomadwire.MountKindFile, Mode: 0o640, Data: []byte(data)},
		}, limits); err != nil {
			t.Fatal(err)
		}
	}
	var requests bytes.Buffer
	client := mount.New(&requests, &responses, limits)
	for ordinal, path := range []string{"/first", "/second"} {
		entry, status, err := client.Lookup(path)
		if err != nil {
			t.Fatal(err)
		}
		if status != gomadwire.MountStatusOK || string(entry.Data) != []string{"first", "second"}[ordinal] {
			t.Fatalf("Lookup() = %#v, %d", entry, status)
		}
		request, err := gomadwire.ReadMountLookupRequest(&requests, limits)
		if err != nil {
			t.Fatal(err)
		}
		if request.Ordinal != uint64(ordinal) || request.Path != path {
			t.Fatalf("request = %#v", request)
		}
	}
}

func TestClientRejectsMismatchedResponseOrdinal(t *testing.T) {
	limits := gomadwire.MountLimits{PathBytes: 4096, FileBytes: 1, DirectoryEntries: 1}
	var response bytes.Buffer
	if err := gomadwire.WriteMountResponse(&response, gomadwire.MountResponse{Ordinal: 1, Status: gomadwire.MountStatusUnmounted}, limits); err != nil {
		t.Fatal(err)
	}
	client := mount.New(&bytes.Buffer{}, &response, limits)
	if _, _, err := client.Lookup("/path"); !errors.Is(err, syscall.EPROTO) {
		t.Fatalf("Lookup() error = %v", err)
	}
}

func TestClientRejectsOversizedPathBeforeWriting(t *testing.T) {
	limits := gomadwire.MountLimits{PathBytes: 2, FileBytes: 1, DirectoryEntries: 1}
	var requests bytes.Buffer
	client := mount.New(&requests, &bytes.Buffer{}, limits)
	if _, _, err := client.Lookup("/long"); !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Fatalf("Lookup() error = %v", err)
	}
	if requests.Len() != 0 {
		t.Fatalf("Lookup() wrote %d request bytes", requests.Len())
	}
}
