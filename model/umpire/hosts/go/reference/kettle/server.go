package kettle

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "umpire.local/model/reference/kettle/v1"
)

type server struct {
	pb.UnimplementedKettleServiceServer
	mu    sync.Mutex
	id    string
	state pb.State
}

func Serve(listener net.Listener, options Options) func() {
	if err := options.Validate(); err != nil {
		panic(err) //nolint:forbidigo // The stop-only reference API must fail invalid setup.
	}
	transport := grpc.NewServer()
	pb.RegisterKettleServiceServer(transport, &server{})
	done := make(chan struct{})
	var serveErr error
	go func() { defer close(done); serveErr = transport.Serve(listener) }()
	return func() {
		transport.Stop()
		<-done
		if serveErr != nil && serveErr != grpc.ErrServerStopped {
			panic(serveErr) //nolint:forbidigo // The stop-only reference API must surface transport failure.
		}
	}
}

func (s *server) current(id string) (*pb.State, error) {
	if err := validateKettleID(id); err != nil {
		return nil, err
	}
	if s.id == "" || s.id != id {
		return nil, status.Error(codes.NotFound, "kettle has not been filled")
	}
	return &pb.State{Temperature: s.state.Temperature, HeatingOn: s.state.HeatingOn, Beeping: s.state.Beeping}, nil
}

func (s *server) Fill(_ context.Context, req *pb.FillRequest) (*pb.FillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateKettleID(req.GetKettleId()); err != nil {
		return nil, err
	}
	if err := validateFill(req.GetKettleId(), req.GetOptions()); err != nil {
		return nil, err
	}
	if s.id != "" && s.id != req.GetKettleId() {
		return nil, status.Error(codes.ResourceExhausted, "reference service supports one kettle")
	}
	if s.id == "" {
		s.id = req.GetKettleId()
		s.state.Temperature = 20
	}
	state, err := s.current(req.GetKettleId())
	return &pb.FillResponse{State: state}, err
}

func (s *server) Heat(_ context.Context, req *pb.HeatRequest) (*pb.HeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(req.GetKettleId()); err != nil {
		return nil, err
	}
	if s.state.HeatingOn == req.GetEnabled() {
		return nil, status.Error(codes.FailedPrecondition, "heat toggle is disabled in current state")
	}
	s.state.HeatingOn = req.GetEnabled()
	s.state.Beeping = false
	state, err := s.current(req.GetKettleId())
	return &pb.HeatResponse{State: state}, err
}

func (s *server) Pour(_ context.Context, req *pb.PourRequest) (*pb.PourResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.current(req.GetKettleId())
	return &pb.PourResponse{State: state}, err
}

func (s *server) Describe(_ context.Context, req *pb.DescribeRequest) (*pb.DescribeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.current(req.GetKettleId())
	return &pb.DescribeResponse{State: state}, err
}

func (s *server) PollHeating(_ context.Context, req *pb.PollHeatingRequest) (*pb.PollHeatingResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.current(req.GetKettleId())
	if err != nil {
		return nil, err
	}
	if !state.HeatingOn {
		return nil, status.Error(codes.FailedPrecondition, "poll heating requires heating enabled")
	}
	action := pb.HeatingAction_HEATING_ACTION_HEAT
	if state.Temperature >= 100 {
		action = pb.HeatingAction_HEATING_ACTION_FAILOVER
	}
	return &pb.PollHeatingResponse{State: state, Action: action, KettleId: req.GetKettleId()}, nil
}

func (s *server) RespondHeating(_ context.Context, req *pb.RespondHeatingRequest) (*pb.RespondHeatingResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(req.GetKettleId()); err != nil {
		return nil, err
	}
	if err := validateHeatingAction(req.GetAction()); err != nil {
		return nil, err
	}
	if !s.state.HeatingOn || req.GetAction() == pb.HeatingAction_HEATING_ACTION_HEAT && s.state.Temperature >= 100 || req.GetAction() == pb.HeatingAction_HEATING_ACTION_FAILOVER && s.state.Temperature < 100 {
		return nil, status.Error(codes.FailedPrecondition, "heating action is disabled in current state")
	}
	if req.GetAction() == pb.HeatingAction_HEATING_ACTION_HEAT {
		s.state.Temperature++
	} else {
		s.state.HeatingOn = false
		s.state.Beeping = true
	}
	state, err := s.current(req.GetKettleId())
	return &pb.RespondHeatingResponse{State: state, Action: req.GetAction()}, err
}
