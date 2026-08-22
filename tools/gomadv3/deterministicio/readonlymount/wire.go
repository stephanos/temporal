package readonlymount

import (
	"errors"
	"fmt"
	"io"
	"os"

	iowire "go.temporal.io/server/tools/gomadv3/deterministicio/internal/wire"
)

type Status uint16

const (
	StatusOK        Status = Status(iowire.MountStatusOK)
	StatusUnmounted Status = Status(iowire.MountStatusUnmounted)
	StatusNotExist  Status = Status(iowire.MountStatusNotExist)
)

type Response struct {
	Ordinal uint64
	Status  Status
	Entry   Entry
}

func WriteLookupRequest(writer io.Writer, ordinal uint64, name string) error {
	return iowire.WriteMountLookupRequest(writer, iowire.MountRequest{Ordinal: ordinal, Path: name}, mountWireLimits(DefaultLimits()))
}

func ReadResponse(reader io.Reader, limits Limits) (Response, error) {
	decoded, err := iowire.ReadMountResponse(reader, mountWireLimits(limits))
	if err != nil {
		return Response{}, err
	}
	response := Response{Ordinal: decoded.Ordinal, Status: Status(decoded.Status), Entry: Entry{Kind: Kind(decoded.Entry.Kind), Mode: os.FileMode(decoded.Entry.Mode), Data: decoded.Entry.Data, Children: make([]Child, 0, len(decoded.Entry.Children))}}
	for _, child := range decoded.Entry.Children {
		response.Entry.Children = append(response.Entry.Children, Child{Name: child.Name, Kind: Kind(child.Kind), Mode: os.FileMode(child.Mode)})
	}
	return response, nil
}

func (broker *Broker) Serve(requests io.Reader, responses io.Writer) error {
	var expected uint64
	for {
		request, err := iowire.ReadMountLookupRequest(requests, mountWireLimits(broker.limits))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read read-only mount request: %w", err)
		}
		if request.Ordinal != expected {
			return fmt.Errorf("out of order read-only mount request %d, want %d", request.Ordinal, expected)
		}
		expected++
		status := StatusOK
		entry, err := broker.Lookup(request.Path)
		if err != nil {
			switch {
			case errors.Is(err, ErrReplayDivergence):
				return err
			case errors.Is(err, os.ErrNotExist):
				normalized, normalizeErr := normalizeLookup(request.Path, broker.limits.PathBytes)
				if normalizeErr != nil {
					return normalizeErr
				}
				broker.mu.Lock()
				_, _, mounted := broker.resolve(normalized)
				broker.mu.Unlock()
				if mounted {
					status = StatusNotExist
				} else {
					status = StatusUnmounted
				}
			default:
				return err
			}
		}
		if err := writeResponse(responses, Response{Ordinal: request.Ordinal, Status: status, Entry: entry}, broker.limits); err != nil {
			return fmt.Errorf("write read-only mount response: %w", err)
		}
	}
}

func writeResponse(writer io.Writer, response Response, limits Limits) error {
	encoded := iowire.MountResponse{Ordinal: response.Ordinal, Status: iowire.MountStatus(response.Status), Entry: iowire.MountEntry{Kind: iowire.MountKind(response.Entry.Kind), Mode: uint32(response.Entry.Mode), Data: response.Entry.Data, Children: make([]iowire.MountChild, 0, len(response.Entry.Children))}}
	for _, child := range response.Entry.Children {
		encoded.Entry.Children = append(encoded.Entry.Children, iowire.MountChild{Name: child.Name, Kind: iowire.MountKind(child.Kind), Mode: uint32(child.Mode)})
	}
	return iowire.WriteMountResponse(writer, encoded, mountWireLimits(limits))
}

func mountWireLimits(limits Limits) iowire.MountLimits {
	return iowire.MountLimits{PathBytes: limits.PathBytes, FileBytes: limits.SingleFileBytes, DirectoryEntries: limits.DirectoryEntries}
}
