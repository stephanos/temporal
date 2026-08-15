package runner

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestDecodeCoordinatorMessagesForwardsProgressAndFinalResponse(t *testing.T) {
	payload := []byte("{\"type\":\"progress\",\"progress\":{\"Phase\":\"preparing\",\"BatchPath\":\"/batch\",\"Selected\":3,\"Attempted\":0,\"Running\":0,\"Succeeded\":0,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"Artifacts\":null}}\n{\"type\":\"result\",\"response\":{\"CampaignResult\":{\"BatchPath\":\"/batch\",\"SelectionCount\":3,\"Attempted\":3,\"Succeeded\":3,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"StopReason\":\"seeds_exhausted\",\"Artifacts\":null},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n")
	var progress []CampaignEvent
	response, err := decodeCoordinatorMessages(bytes.NewReader(payload), func(update CampaignEvent) error {
		progress = append(progress, update)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 1 || progress[0].Phase != ProgressPreparing || response.CampaignResult.Attempted != 3 {
		t.Fatalf("progress = %#v, response = %#v", progress, response)
	}
}

func TestDecodeCoordinatorMessagesRejectsProgressAfterResult(t *testing.T) {
	payload := []byte("{\"type\":\"result\",\"response\":{\"CampaignResult\":{},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n{\"type\":\"progress\",\"progress\":{\"Phase\":\"running\"}}\n")
	if _, err := decodeCoordinatorMessages(bytes.NewReader(payload), nil); err == nil {
		t.Fatal("decodeCoordinatorMessages() succeeded")
	}
}

func TestDecodeCoordinatorMessagesPreservesDrainFailure(t *testing.T) {
	drainFailure := errors.New("drain failed")
	input := &sequencedReader{reads: []readerResult{{data: []byte("x")}, {err: drainFailure}}}
	_, err := decodeCoordinatorMessages(input, nil)
	if err == nil || !errors.Is(err, drainFailure) {
		t.Fatalf("decodeCoordinatorMessages() error = %v", err)
	}
}

type readerResult struct {
	data []byte
	err  error
}

type sequencedReader struct {
	reads []readerResult
}

func (reader *sequencedReader) Read(output []byte) (int, error) {
	if len(reader.reads) == 0 {
		return 0, io.EOF
	}
	result := reader.reads[0]
	reader.reads = reader.reads[1:]
	return copy(output, result.data), result.err
}
