package runner

import (
	"fmt"

	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaign"
)

// CampaignShard identifies one disjoint ordinal-modulo partition of a campaign.
type CampaignShard struct {
	Index uint64 `json:"index"`
	Count uint64 `json:"count"`
}

func (shard CampaignShard) Validate() error {
	if shard.Count == 0 || shard.Index >= shard.Count {
		return fmt.Errorf("campaign shard %d/%d is invalid", shard.Index, shard.Count)
	}
	return nil
}

func (shard CampaignShard) Owns(ordinal uint64) bool {
	return shard.Count != 0 && ordinal%shard.Count == shard.Index
}

func (shard CampaignShard) SelectionCount(total uint64) uint64 {
	if shard.Count == 0 || shard.Index >= shard.Count || total <= shard.Index {
		return 0
	}
	return 1 + (total-1-shard.Index)/shard.Count
}

func normalizedCampaignShard(shard CampaignShard) CampaignShard {
	if shard.Count == 0 {
		return CampaignShard{Count: 1}
	}
	return shard
}

func campaignStoreShard(shard CampaignShard) *campaign.CampaignShard {
	if shard.Count == 0 {
		return nil
	}
	return &campaign.CampaignShard{Index: record.Uint64String(shard.Index), Count: record.Uint64String(shard.Count)}
}
