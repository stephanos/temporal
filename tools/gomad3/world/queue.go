package world

import (
	"bytes"
	"container/heap"
)

type eventHeap []*eventState

func (queue eventHeap) Len() int { return len(queue) }

func (queue eventHeap) Less(left, right int) bool {
	return lessEvent(queue[left], queue[right])
}

func (queue eventHeap) Swap(left, right int) {
	queue[left], queue[right] = queue[right], queue[left]
	queue[left].heapIndex = left
	queue[right].heapIndex = right
}

func (queue *eventHeap) Push(value any) {
	event := value.(*eventState)
	event.heapIndex = len(*queue)
	*queue = append(*queue, event)
}

func (queue *eventHeap) Pop() any {
	old := *queue
	last := len(old) - 1
	event := old[last]
	old[last] = nil
	event.heapIndex = -1
	*queue = old[:last]
	return event
}

func lessEvent(left, right *eventState) bool {
	if left.readiness.At != right.readiness.At {
		return left.readiness.At < right.readiness.At
	}
	leftRequest := left.request.request
	rightRequest := right.request.request
	if leftRequest.Priority != rightRequest.Priority {
		return leftRequest.Priority < rightRequest.Priority
	}
	for _, pair := range [][2]string{
		{leftRequest.Resource.Adapter, rightRequest.Resource.Adapter},
		{leftRequest.Resource.Kind, rightRequest.Resource.Kind},
		{leftRequest.Resource.Key, rightRequest.Resource.Key},
		{leftRequest.Kind, rightRequest.Kind},
		{left.readiness.Kind, right.readiness.Kind},
		{left.readiness.EquivalenceClass, right.readiness.EquivalenceClass},
	} {
		if pair[0] != pair[1] {
			return pair[0] < pair[1]
		}
	}
	if left.readiness.EquivalenceClass != "" {
		if comparison := bytes.Compare(left.choiceRank[:], right.choiceRank[:]); comparison != 0 {
			return comparison < 0
		}
	}
	return left.readiness.RequestID < right.readiness.RequestID
}

var _ heap.Interface = (*eventHeap)(nil)
