package determinism

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type reviewRecord struct {
	Timestamp string   `json:"timestamp"`
	Reviewed  Reviewed `json:"reviewed"`
}

func checkReviewHistory(data []byte, reviewed Reviewed) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var records []reviewRecord
	if err := decoder.Decode(&records); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("review history has trailing data")
	}
	if len(records) == 0 {
		return errors.New("review history is empty")
	}
	var previous time.Time
	for index, record := range records {
		timestamp, err := time.Parse(time.RFC3339Nano, record.Timestamp)
		if err != nil || !strings.HasSuffix(record.Timestamp, "Z") {
			return fmt.Errorf("review history entry %d requires an explicit UTC timestamp", index)
		}
		if !timestamp.After(previous) {
			return fmt.Errorf("review history entry %d timestamp must advance", index)
		}
		if timestamp.Format(time.DateOnly) != record.Reviewed.Date {
			return fmt.Errorf("review history entry %d date differs from timestamp", index)
		}
		if record.Reviewed.GoToolchain == "" || record.Reviewed.Wasmtime == "" || record.Reviewed.Rust == "" {
			return fmt.Errorf("review history entry %d has incomplete versions", index)
		}
		previous = timestamp
	}
	if records[len(records)-1].Reviewed != reviewed {
		return errors.New("inventory reviewed date or versions differ from latest review history entry; append a new dated review record")
	}
	return nil
}

func TestInventoryPinsRequireDatedReviewRecord(t *testing.T) {
	t.Parallel()
	inventory := loadInventory(t)
	history := mustRead(t, "review-history.json")
	require.NoError(t, checkReviewHistory(history, inventory.Reviewed))
	changed := inventory.Reviewed
	changed.Wasmtime += ".new"
	require.ErrorContains(t, checkReviewHistory(history, changed), "append a new dated review record")
}

func TestInventoryReviewHistoryRejectsStaleReview(t *testing.T) {
	t.Parallel()
	initial := reviewRecord{Timestamp: "2026-10-20T15:00:00Z", Reviewed: Reviewed{Date: "2026-10-20", GoToolchain: "go1.27.1", Wasmtime: "47.0.3", Rust: "1.99.1"}}
	updated := initial
	updated.Reviewed.Wasmtime = "47.0.4"
	for _, test := range []struct {
		name   string
		record reviewRecord
		want   string
	}{
		{"unchanged timestamp", updated, "timestamp must advance"},
		{"new timestamp", reviewRecord{Timestamp: "2026-10-20T16:00:00Z", Reviewed: updated.Reviewed}, ""},
		{"stale date", reviewRecord{Timestamp: "2026-10-21T15:00:00Z", Reviewed: updated.Reviewed}, "date differs"},
		{"missing UTC", reviewRecord{Timestamp: "2026-10-20T16:00:00+00:00", Reviewed: updated.Reviewed}, "explicit UTC"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal([]reviewRecord{initial, test.record})
			require.NoError(t, err)
			err = checkReviewHistory(data, test.record.Reviewed)
			if test.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.want)
			}
		})
	}
}
