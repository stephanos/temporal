package testhooks

import (
	"context"

	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/namespace"
)

type ActivityDelivery struct {
	Execution chasm.ExecutionKey
	Stamp     int32
}

var ActivityDispatch = newKey[func(context.Context, ActivityDelivery) error, namespace.ID]()
