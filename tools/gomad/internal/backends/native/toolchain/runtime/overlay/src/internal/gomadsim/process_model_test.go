package gomadsim_test

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"internal/gomadsim"
)

func TestProcessModelExchangeCorrelatesConcurrentResponses(t *testing.T) {
	if os.Getenv("GOMAD_TEST_PROCESS_MODEL_CHILD") == "1" {
		testProcessModelExchangeResponses(t)
		return
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessModelExchangeCorrelatesConcurrentResponses$")
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOMAD3_SIMULATION_") || strings.HasPrefix(entry, "GOMADSEED=") || strings.HasPrefix(entry, "GOMAD_TEST_PROCESS_MODEL_CHILD=") {
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "GOMAD_TEST_PROCESS_MODEL_CHILD=1", "GOMAD3_SIMULATION_ROLE=node", "GOMAD3_SIMULATION_MODEL_REQUEST_FD=3", "GOMAD3_SIMULATION_MODEL_RESPONSE_FD=4")
	command.ExtraFiles = []*os.File{requestWrite, responseRead}

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
		third.Response, third.Payload, third.Error = true, nil, "host model failure"
		served <- writeProcessModelTestFrame(responseWrite, third)
	}()
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		t.Fatalf("process model child: %v: %s", runErr, output)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func testProcessModelExchangeResponses(t *testing.T) {
	t.Helper()
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
