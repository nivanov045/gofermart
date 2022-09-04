package products

import (
	"encoding/json"
	"fmt"
)

type RewardType int

const (
	RewardTypeUnknown = iota
	RewardTypePercent
	RewardTypePoints
)

type Product struct {
	Match      string     `json:"match"`
	Reward     int        `json:"reward"`
	RewardType RewardType `json:"reward_type"`
}

func (p *Product) UnmarshalJSON(data []byte) error {
	type ProductAlias Product

	aliasValue := &struct {
		*ProductAlias
		RewardType string `json:"reward_type"`
	}{
		ProductAlias: (*ProductAlias)(p),
	}

	if err := json.Unmarshal(data, aliasValue); err != nil {
		return err
	}

	switch aliasValue.RewardType {
	case "%":
		p.RewardType = RewardTypePercent
		break
	case "pt":
		p.RewardType = RewardTypePoints
		break
	default:
		p.RewardType = RewardTypeUnknown
	}

	if p.RewardType == RewardTypeUnknown {
		return fmt.Errorf("unknown reward type '%s'", aliasValue.RewardType)
	}

	return nil
}

func (p *Product) MarshalJSON() ([]byte, error) {
	type ProductAlias Product

	var rewardType string
	switch p.RewardType {
	case RewardTypePercent:
		rewardType = "%"
		break
	case RewardTypePoints:
		rewardType = "pt"
		break
	default:
		return nil, fmt.Errorf("unknown reward type: '%v'", p.RewardType)
	}

	aliasValue := &struct {
		*ProductAlias
		RewardType string `json:"reward_type"`
	}{
		ProductAlias: (*ProductAlias)(p),
		RewardType:   rewardType,
	}

	return json.Marshal(aliasValue)
}
