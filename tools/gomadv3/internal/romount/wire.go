package romount

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	requestHeaderBytes  = 24
	responseHeaderBytes = 40
	wireVersion         = 1
	operationLookup     = 1
)

var (
	requestMagic  = [8]byte{'G', 'O', 'M', 'A', 'D', 'R', 'O', 1}
	responseMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'R', 'S', 1}
)

type Status uint16

const (
	StatusOK Status = iota
	StatusUnmounted
	StatusNotExist
	StatusError
)

type Response struct {
	Ordinal uint64
	Status  Status
	Entry   Entry
}

func WriteLookupRequest(writer io.Writer, ordinal uint64, name string) error {
	var header [requestHeaderBytes]byte
	copy(header[:8], requestMagic[:])
	binary.BigEndian.PutUint16(header[8:10], wireVersion)
	binary.BigEndian.PutUint16(header[10:12], operationLookup)
	binary.BigEndian.PutUint64(header[12:20], ordinal)
	binary.BigEndian.PutUint32(header[20:24], uint32(len(name)))
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, name); err != nil {
		return err
	}
	return nil
}

func ReadResponse(reader io.Reader, limits Limits) (Response, error) {
	var header [responseHeaderBytes]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return Response{}, err
	}
	if string(header[:8]) != string(responseMagic[:]) || binary.BigEndian.Uint16(header[8:10]) != wireVersion {
		return Response{}, errors.New("invalid read-only mount response header")
	}
	response := Response{
		Status:  Status(binary.BigEndian.Uint16(header[10:12])),
		Ordinal: binary.BigEndian.Uint64(header[12:20]),
		Entry: Entry{
			Kind: Kind(header[20]), Mode: os.FileMode(binary.BigEndian.Uint32(header[24:28])),
		},
	}
	dataBytes := binary.BigEndian.Uint64(header[28:36])
	children := binary.BigEndian.Uint32(header[36:40])
	if dataBytes > limits.SingleFileBytes || uint64(children) > limits.DirectoryEntries {
		return Response{}, ErrCapacity
	}
	response.Entry.Data = make([]byte, int(dataBytes))
	if _, err := io.ReadFull(reader, response.Entry.Data); err != nil {
		return Response{}, err
	}
	response.Entry.Children = make([]Child, 0, children)
	for range children {
		var childHeader [8]byte
		if _, err := io.ReadFull(reader, childHeader[:]); err != nil {
			return Response{}, err
		}
		nameBytes := binary.BigEndian.Uint16(childHeader[:2])
		if uint64(nameBytes) > limits.PathBytes {
			return Response{}, ErrCapacity
		}
		name := make([]byte, nameBytes)
		if _, err := io.ReadFull(reader, name); err != nil {
			return Response{}, err
		}
		response.Entry.Children = append(response.Entry.Children, Child{Name: string(name), Kind: Kind(childHeader[2]), Mode: os.FileMode(binary.BigEndian.Uint32(childHeader[4:8]))})
	}
	return response, nil
}

func (broker *Broker) Serve(requests io.Reader, responses io.Writer) error {
	var expected uint64
	for {
		var header [requestHeaderBytes]byte
		if _, err := io.ReadFull(requests, header[:]); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read read-only mount request: %w", err)
		}
		if string(header[:8]) != string(requestMagic[:]) || binary.BigEndian.Uint16(header[8:10]) != wireVersion || binary.BigEndian.Uint16(header[10:12]) != operationLookup {
			return errors.New("invalid read-only mount request header")
		}
		ordinal := binary.BigEndian.Uint64(header[12:20])
		if ordinal != expected {
			return fmt.Errorf("out of order read-only mount request %d, want %d", ordinal, expected)
		}
		expected++
		pathBytes := binary.BigEndian.Uint32(header[20:24])
		if uint64(pathBytes) > broker.limits.PathBytes {
			return fmt.Errorf("oversized read-only mount request path")
		}
		name := make([]byte, pathBytes)
		if _, err := io.ReadFull(requests, name); err != nil {
			return fmt.Errorf("read read-only mount request path: %w", err)
		}
		status := StatusOK
		entry, err := broker.Lookup(string(name))
		if err != nil {
			switch {
			case errors.Is(err, ErrReplayDivergence):
				return err
			case errors.Is(err, os.ErrNotExist):
				normalized, normalizeErr := normalizeLookup(string(name), broker.limits.PathBytes)
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
		if err := writeResponse(responses, Response{Ordinal: ordinal, Status: status, Entry: entry}); err != nil {
			return fmt.Errorf("write read-only mount response: %w", err)
		}
	}
}

func writeResponse(writer io.Writer, response Response) error {
	var header [responseHeaderBytes]byte
	copy(header[:8], responseMagic[:])
	binary.BigEndian.PutUint16(header[8:10], wireVersion)
	binary.BigEndian.PutUint16(header[10:12], uint16(response.Status))
	binary.BigEndian.PutUint64(header[12:20], response.Ordinal)
	header[20] = byte(response.Entry.Kind)
	binary.BigEndian.PutUint32(header[24:28], uint32(response.Entry.Mode))
	binary.BigEndian.PutUint64(header[28:36], uint64(len(response.Entry.Data)))
	binary.BigEndian.PutUint32(header[36:40], uint32(len(response.Entry.Children)))
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	if len(response.Entry.Data) != 0 {
		if _, err := writer.Write(response.Entry.Data); err != nil {
			return err
		}
	}
	for _, child := range response.Entry.Children {
		if len(child.Name) > int(^uint16(0)) {
			return ErrCapacity
		}
		var childHeader [8]byte
		binary.BigEndian.PutUint16(childHeader[:2], uint16(len(child.Name)))
		childHeader[2] = byte(child.Kind)
		binary.BigEndian.PutUint32(childHeader[4:8], uint32(child.Mode))
		if _, err := writer.Write(childHeader[:]); err != nil {
			return err
		}
		if _, err := io.WriteString(writer, child.Name); err != nil {
			return err
		}
	}
	return nil
}
