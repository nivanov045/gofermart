package models

import (
	"encoding/json"
)

type RewardType int

const (
	RewardTypeUnknown RewardType = iota
	RewardTypePercent
	RewardTypePoints
)

type Product struct {
	Match      string     `json:"match"`
	Reward     float64    `json:"reward"`
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
	case "pt":
		p.RewardType = RewardTypePoints
	default:
		p.RewardType = RewardTypeUnknown
	}

	if p.RewardType == RewardTypeUnknown {
		return NewUnknownTypeError(p.RewardType, aliasValue.RewardType)
	}

	return nil
}

func (p *Product) MarshalJSON() ([]byte, error) {
	type ProductAlias Product

	var rewardType string
	switch p.RewardType {
	case RewardTypePercent:
		rewardType = "%"
	case RewardTypePoints:
		rewardType = "pt"
	default:
		return nil, NewUnknownTypeError(p.RewardType, "")
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
