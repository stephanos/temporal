package execution

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad3/internal/model/clock"
)

const simulationInitialTime = clock.InitialTime

type simulationTimeResponseKind uint8

type simulationTimeRequest struct {
	Generation uint64
	Current    int64
	Deadline   int64
	Arrivals   uint32
}

type simulationTimeResponse struct {
	Generation uint64
	Kind       simulationTimeResponseKind
	Time       int64
}

func encodeSimulationActivationTime(current int64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, uint64(current))
	return encoded
}

func decodeSimulationActivationTime(encoded []byte) (int64, error) {
	if len(encoded) != 8 {
		return 0, errors.New("process simulation activation time is invalid")
	}
	current := int64(binary.BigEndian.Uint64(encoded))
	if current < simulationInitialTime {
		return 0, errors.New("process simulation activation time is invalid")
	}
	return current, nil
}

func serveSimulationTime(ctx context.Context, source io.Reader, destination io.Writer, handler func(context.Context, simulationTimeRequest) (simulationTimeResponse, error)) error {
	if handler == nil {
		return errors.New("simulation time handler is unavailable")
	}
	for {
		encodedRequest := make([]byte, simulationTimeRequestBytes)
		if _, err := io.ReadFull(source, encodedRequest); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read simulation time request: %w", err)
		}
		request, err := decodeSimulationTimeRequest(encodedRequest)
		if err != nil {
			return err
		}
		response, err := handler(ctx, request)
		if err != nil {
			return err
		}
		encodedResponse, err := encodeSimulationTimeResponse(response)
		if err != nil {
			return err
		}
		written, err := destination.Write(encodedResponse)
		if err != nil {
			return fmt.Errorf("write simulation time response: %w", err)
		}
		if written != len(encodedResponse) {
			return io.ErrShortWrite
		}
	}
}

func quiesceSimulationTime(ctx context.Context, arbiter *clock.BrokerClock, participant *clock.BrokerParticipant, request simulationTimeRequest) (simulationTimeResponse, error) {
	response, err := arbiter.Quiesce(ctx, participant, clock.BrokerTimeRequest{Generation: request.Generation, Current: request.Current, Deadline: request.Deadline, Arrivals: request.Arrivals})
	return simulationTimeResponse{Generation: response.Generation, Kind: simulationTimeResponseKind(response.Kind), Time: response.Time}, err
}
