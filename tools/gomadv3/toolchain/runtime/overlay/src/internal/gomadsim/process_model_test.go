package gomadsim_test

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strconv"
	"testing"

	"internal/gomadsim"
)

func TestProcessModelExchangeCorrelatesConcurrentResponses(t *testing.T) {
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	responseRead, responseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, file := range []*os.File{requestRead, requestWrite, responseRead, responseWrite} {
			if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				t.Error(err)
			}
		}
	})
	t.Setenv("GOMADV3_SIMULATION_ROLE", "node")
	t.Setenv("GOMADV3_SIMULATION_MODEL_REQUEST_FD", strconv.FormatUint(uint64(requestWrite.Fd()), 10))
	t.Setenv("GOMADV3_SIMULATION_MODEL_RESPONSE_FD", strconv.FormatUint(uint64(responseRead.Fd()), 10))

	served := make(chan error, 1)
	go func() {
		first, err := readProcessModelTestFrame(requestRead)
		if err != nil {
			served <- err
			return
		}
		second, err := readProcessModelTestFrame(requestRead)
		if err != nil {
			served <- err
			return
		}
		second.Response, second.Payload = true, []byte(second.Node)
		first.Response, first.Payload = true, []byte(first.Node)
		if err := writeProcessModelTestFrame(responseWrite, second); err != nil {
			served <- err
			return
		}
		if err := writeProcessModelTestFrame(responseWrite, first); err != nil {
			served <- err
			return
		}
		third, err := readProcessModelTestFrame(requestRead)
		if err != nil {
			served <- err
			return
		}
		third.Response, third.Error = true, "host model failure"
		served <- writeProcessModelTestFrame(responseWrite, third)
	}()
	results := make(chan string, 2)
	for _, node := range []string{"first", "second"} {
		go func(node string) {
			response, remoteErr, ok := gomadsim.ProcessModelExchange(node, 1, []byte(node), 64)
			if !ok || remoteErr != "" {
				results <- "failed"
				return
			}
			results <- string(response)
		}(node)
	}
	got := map[string]bool{<-results: true, <-results: true}
	if !got["first"] || !got["second"] || len(got) != 2 {
		t.Fatalf("responses = %v", got)
	}
	response, remoteErr, ok := gomadsim.ProcessModelExchange("third", 1, []byte("third"), 64)
	if !ok || remoteErr != "host model failure" || response != nil {
		t.Fatalf("remote error response = %q, %q, %t", response, remoteErr, ok)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func readProcessModelTestFrame(source io.Reader) (gomadsim.ModelTransportFrame, error) {
	var header [4]byte
	if _, err := io.ReadFull(source, header[:]); err != nil {
		return gomadsim.ModelTransportFrame{}, err
	}
	encoded := make([]byte, binary.BigEndian.Uint32(header[:]))
	if _, err := io.ReadFull(source, encoded); err != nil {
		return gomadsim.ModelTransportFrame{}, err
	}
	return gomadsim.DecodeModelTransportFrame(encoded)
}

func writeProcessModelTestFrame(destination io.Writer, frame gomadsim.ModelTransportFrame) error {
	encoded, err := gomadsim.EncodeModelTransportFrame(frame)
	if err != nil {
		return err
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(encoded)))
	if _, err := destination.Write(header[:]); err != nil {
		return err
	}
	_, err = destination.Write(encoded)
	return err
}
