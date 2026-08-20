package cli

import (
	"testing"

	"go.temporal.io/server/tools/gomadv3/runner"
)

func TestParseCampaignShardUsesZeroBasedIndex(t *testing.T) {
	shard, err := parseCampaignShard("2/8")
	if err != nil {
		t.Fatal(err)
	}
	if shard.Index != 2 || shard.Count != 8 {
		t.Fatalf("parseCampaignShard() = %#v", shard)
	}
	for _, value := range []string{"", "1", "1/", "/2", "1/0", "2/2", "01/2", "1/02", "01/x", "1/2/3"} {
		if _, err := parseCampaignShard(value); err == nil {
			t.Fatalf("parseCampaignShard(%q) succeeded", value)
		}
	}
}

func TestFormatMissingOrdinalsCompactsRanges(t *testing.T) {
	if got := formatMissingOrdinals(nil); got != "none" {
		t.Fatalf("formatMissingOrdinals(nil) = %q", got)
	}
	if got := formatMissingOrdinals([]runner.CampaignOrdinalRange{{Start: 1, End: 1}, {Start: 3, End: 7}}); got != "1,3-7" {
		t.Fatalf("formatMissingOrdinals() = %q", got)
	}
}
