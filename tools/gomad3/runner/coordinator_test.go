package runner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
)

func TestDecodeCoordinatorMessagesForwardsProgressAndFinalResponse(t *testing.T) {
	payload := []byte("{\"type\":\"progress\",\"progress\":{\"Phase\":\"preparing\",\"campaign_path\":\"/campaign\",\"Selected\":3,\"Attempted\":0,\"Running\":0,\"Succeeded\":0,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"Artifacts\":null}}\n{\"type\":\"result\",\"response\":{\"CampaignResult\":{\"campaign_path\":\"/campaign\",\"SelectionCount\":3,\"Attempted\":3,\"Succeeded\":3,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"StopReason\":\"seeds_exhausted\",\"Artifacts\":null},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n")
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

type cancellationCoordinatorProvider struct {
	*native.Provider
	observed backends.ExecutionResult
}

func (provider *cancellationCoordinatorProvider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	result, err := provider.Provider.Execute(ctx, request)
	provider.observed = result
	return result, err
}

func TestCoordinatorOutputFailureCancelsAndReapsBeforeDraining(t *testing.T) {
	for _, test := range []struct {
		mode, reason string
	}{
		{mode: "malformed", reason: "coordinator_decode"},
		{mode: "progress", reason: "progress_output"},
	} {
		t.Run(test.mode, func(t *testing.T) {
			t.Setenv("GOMAD3_COORDINATOR_CANCEL_CASE", test.mode)
			config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
			config.OverallTimeout = 2 * time.Second
			provider := &cancellationCoordinatorProvider{Provider: native.New(native.Config{
				CoordinatorCommand: []string{os.Args[0], "-test.run=^TestCoordinatorCancellationHelper$"},
				RunnerBuild:        config.RunnerBuild,
			})}
			config.Backend = provider
			progressFailure := errors.New("progress failed")
			config.Progress = func(CampaignEvent) error { return progressFailure }
			_, err := runIsolated(t.Context(), campaignRequestFromSpec(config))
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason || !provider.observed.Cancelled || !provider.observed.GroupGone {
				t.Fatalf("coordinator failure = %v, transport = %#v", err, provider.observed)
			}
			if test.mode == "progress" && !errors.Is(err, progressFailure) {
				t.Fatalf("progress cause was lost: %v", err)
			}
		})
	}
}

func TestCoordinatorCancellationHelper(t *testing.T) {
	mode := os.Getenv("GOMAD3_COORDINATOR_CANCEL_CASE")
	if mode == "" {
		t.Skip("coordinator subprocess only")
	}
	payload := "malformed\n"
	if mode == "progress" {
		payload = "{\"type\":\"progress\",\"progress\":{\"Phase\":\"running\"}}\n"
	}
	if _, err := io.WriteString(os.Stdout, payload); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	<-timer.C
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
