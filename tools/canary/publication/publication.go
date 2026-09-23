// Package publication publishes each admitted iteration's fn-26 receipt, unchanged, and then its
// provenance, exclusively and idempotently, under names that are their own identities. It runs
// after the cleanup attempt, whatever its outcome. A lost or unconstructible iteration has no
// receipt and publishes nothing.
package publication

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/publish"
)

// ProvenanceSuffix ends every provenance's published name, after its identity.
const ProvenanceSuffix = ".provenance.json"

// Item is one iteration to publish: its Run, the decision fn-26 made, and the rendered receipt
// and provenance. An unconstructible iteration's Item has no receipt and is not published.
type Item struct {
	RunID      string
	Status     string
	Receipt    []byte
	Provenance []byte
}

// Published is one iteration's publication: each document's identity and where it stands.
type Published struct {
	RunID      string
	Receipt    publish.Publication
	Provenance publish.Publication
	// ReceiptIdentity and ProvenanceIdentity are the names' identities.
	ReceiptIdentity    string
	ProvenanceIdentity string
}

// UnreportedError is a publication that succeeded but could not be recorded as published: the
// named ambiguity. The documents stand, and nothing is published again because of it.
type UnreportedError struct {
	RunID string
	Err   error
}

func (e *UnreportedError) Error() string {
	return fmt.Sprintf("publication-unreported: Run %s is published but could not be recorded: %v", e.RunID, e.Err)
}

func (e *UnreportedError) Unwrap() error { return e.Err }

// decisions are the statuses that have a receipt.
var decisions = map[string]bool{
	evaluation.DecisionAccepted: true, evaluation.DecisionRejected: true, evaluation.DecisionIncomplete: true,
}

// Publish checks every item before it publishes any, so a crossed or unreadable document, or a
// name that already holds other bytes, changes nothing. Then, in order, it publishes each receipt
// and then its provenance under root and calls recorded, stopping at the first failure. A conflict
// is a publish.ConflictError and is never overwritten; a recorded that fails is an
// UnreportedError. It returns what it published.
func Publish(ctx context.Context, root string, items []Item, recorded func(runID string) error) ([]Published, error) {
	var publishable []Item
	for _, item := range items {
		if !decisions[item.Status] {
			if item.Receipt != nil || item.Provenance != nil {
				return nil, fmt.Errorf("iteration %s is %s and has no receipt to publish", item.RunID, item.Status)
			}
			continue
		}
		if err := check(item); err != nil {
			return nil, fmt.Errorf("iteration %s: %w", item.RunID, err)
		}
		for _, document := range []struct {
			name     string
			contents []byte
		}{
			{evaluation.ReceiptIdentity(item.Receipt) + ".json", item.Receipt},
			{assessment.ProvenanceIdentity(item.Provenance) + ProvenanceSuffix, item.Provenance},
		} {
			if err := publish.Check(root, document.name, document.contents); err != nil {
				return nil, err
			}
		}
		publishable = append(publishable, item)
	}
	var published []Published
	for _, item := range publishable {
		receiptIdentity := evaluation.ReceiptIdentity(item.Receipt)
		provenanceIdentity := assessment.ProvenanceIdentity(item.Provenance)
		receipt, err := publish.Publish(ctx, root, receiptIdentity+".json", item.Receipt)
		if err != nil {
			return published, err
		}
		provenance, err := publish.Publish(ctx, root, provenanceIdentity+ProvenanceSuffix, item.Provenance)
		if err != nil {
			return published, err
		}
		published = append(published, Published{
			RunID: item.RunID, Receipt: receipt, Provenance: provenance,
			ReceiptIdentity: receiptIdentity, ProvenanceIdentity: provenanceIdentity,
		})
		if recorded != nil {
			if err := recorded(item.RunID); err != nil {
				return published, &UnreportedError{RunID: item.RunID, Err: err}
			}
		}
	}
	return published, nil
}

// check reads both documents back and requires them to be about each other: the provenance names
// this receipt's identity, its Profile and its Run, and the receipt's decision is the item's.
func check(item Item) error {
	receipt, err := evaluation.DecodeReceipt(item.Receipt)
	if err != nil {
		return err
	}
	provenance, err := assessment.DecodeProvenance(item.Provenance)
	if err != nil {
		return err
	}
	switch {
	case provenance.Receipt != evaluation.ReceiptIdentity(item.Receipt):
		return errors.New("the provenance names another receipt")
	case provenance.EvaluationProfile != receipt.Profile.Identity:
		return errors.New("the provenance names another Evaluation Profile")
	case provenance.Invocation.RunID != receipt.Run.RunID || receipt.Run.RunID != item.RunID:
		return errors.New("the provenance, the receipt and the iteration name different Runs")
	case receipt.Decision != item.Status:
		return fmt.Errorf("the receipt decides %s, the iteration is %s", receipt.Decision, item.Status)
	}
	return nil
}
