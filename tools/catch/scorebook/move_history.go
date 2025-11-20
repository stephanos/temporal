// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package scorebook

import (
	"sync"

	"go.temporal.io/server/tools/catch/scorebook/types"
)

// MoveHistory keeps track of all moves for querying during tests.
type MoveHistory struct {
	mu    sync.RWMutex
	moves []types.Move
}

// NewMoveHistory creates a new move history tracker.
func NewMoveHistory() *MoveHistory {
	return &MoveHistory{
		moves: make([]types.Move, 0),
	}
}

// Add records a move in the history.
func (h *MoveHistory) Add(move types.Move) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.moves = append(h.moves, move)
}

// AddAll records multiple moves in the history.
func (h *MoveHistory) AddAll(moves []types.Move) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.moves = append(h.moves, moves...)
}

// QueryByWorkflowID returns all moves for a specific workflow ID.
func (h *MoveHistory) QueryByWorkflowID(workflowID string) []types.Move {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []types.Move
	for _, move := range h.moves {
		// Check if the move has a workflow ID field by type assertion
		// Different move types store workflow ID in different ways
		if hasWorkflowID(move, workflowID) {
			result = append(result, move)
		}
	}
	return result
}

// QueryByType returns all moves of a specific type.
func (h *MoveHistory) QueryByType(moveType string) []types.Move {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []types.Move
	for _, move := range h.moves {
		if move.MoveType() == moveType {
			result = append(result, move)
		}
	}
	return result
}

// QueryByWorkflowIDAndType returns moves matching both workflow ID and type.
func (h *MoveHistory) QueryByWorkflowIDAndType(workflowID string, moveType string) []types.Move {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []types.Move
	for _, move := range h.moves {
		if move.MoveType() == moveType && hasWorkflowID(move, workflowID) {
			result = append(result, move)
		}
	}
	return result
}

// All returns all moves in the history.
func (h *MoveHistory) All() []types.Move {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]types.Move, len(h.moves))
	copy(result, h.moves)
	return result
}

// Clear removes all moves from history.
func (h *MoveHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.moves = make([]types.Move, 0)
}

// hasWorkflowID checks if a move is related to the given workflow ID.
// This checks the target entity identity for workflow-related entities.
func hasWorkflowID(move types.Move, workflowID string) bool {
	identity := move.TargetEntity()
	if identity == nil {
		return false
	}

	// Check the entity ID itself
	if identity.EntityID.ID == workflowID {
		return true
	}

	// Check parent ID (e.g., WorkflowTask has TaskQueue as parent, but task ID contains workflow ID)
	// For WorkflowTask entities, the ID is typically "taskQueue:workflowID:runID"
	// Check if the entity ID contains the workflow ID
	if len(identity.EntityID.ID) > len(workflowID) {
		// Simple substring match - could be more sophisticated
		for i := 0; i <= len(identity.EntityID.ID)-len(workflowID); i++ {
			if identity.EntityID.ID[i:i+len(workflowID)] == workflowID {
				return true
			}
		}
	}

	return false
}
