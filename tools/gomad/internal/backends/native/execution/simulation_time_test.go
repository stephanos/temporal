package execution

import (
	"bytes"
	"context"
	"testing"
)

func TestSimulationTimeWireRoundTripsFixedFrames(t *testing.T) {
	request := simulationTimeRequest{Generation: 7, Current: simulationInitialTime + 11, Deadline: simulationInitialTime + 23, Arrivals: 2}
	encodedRequest, err := encodeSimulationTimeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(encodedRequest) != simulationTimeRequestBytes {
		t.Fatalf("encoded request bytes = %d", len(encodedRequest))
	}
	decodedRequest, err := decodeSimulationTimeRequest(encodedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if decodedRequest != request {
		t.Fatalf("decoded request = %#v, want %#v", decodedRequest, request)
	}

	response := simulationTimeResponse{Generation: 7, Kind: simulationTimeAdvance, Time: simulationInitialTime + 23}
	encodedResponse, err := encodeSimulationTimeResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(encodedResponse) != simulationTimeResponseBytes {
		t.Fatalf("encoded response bytes = %d", len(encodedResponse))
	}
	decodedResponse, err := decodeSimulationTimeResponse(encodedResponse)
	if err != nil {
		t.Fatal(err)
	}
	if decodedResponse != response {
		t.Fatalf("decoded response = %#v, want %#v", decodedResponse, response)
	}

	encodedResponse[0] = 0
	_, err = decodeSimulationTimeResponse(encodedResponse)
	if err == nil {
		t.Fatal("decode accepted a changed response magic")
	}
}

func TestServeSimulationTimeExchangesBoundedFrames(t *testing.T) {
	request := simulationTimeRequest{Generation: 3, Current: simulationInitialTime, Deadline: simulationInitialTime + 5}
	encoded, err := encodeSimulationTimeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	err = serveSimulationTime(context.Background(), bytes.NewReader(encoded), &destination, func(_ context.Context, actual simulationTimeRequest) (simulationTimeResponse, error) {
		if actual != request {
			t.Fatalf("request = %#v, want %#v", actual, request)
		}
		return simulationTimeResponse{Generation: actual.Generation, Kind: simulationTimeAdvance, Time: actual.Deadline}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := decodeSimulationTimeResponse(destination.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 3, Kind: simulationTimeAdvance, Time: simulationInitialTime + 5}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
}
