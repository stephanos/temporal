package environment

import (
	"context"
	"errors"
	"sync"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type preparedFactory struct {
	mu           sync.Mutex
	capabilities []string
	session      Session
}

func PrepareOnce(capabilities []string, session Session) (Factory, error) {
	if len(capabilities) == 0 || session == nil {
		return nil, errors.New("prepared environment requires capabilities and a session")
	}
	return &preparedFactory{
		capabilities: append([]string(nil), capabilities...), session: session,
	}, nil
}

func (f *preparedFactory) Capabilities() []string {
	return append([]string(nil), f.capabilities...)
}

func (f *preparedFactory) Prepare(context.Context, protocol.Experiment) (Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.session == nil {
		return nil, errors.New("prepared environment is single use")
	}
	session := f.session
	f.session = nil
	return session, nil
}
