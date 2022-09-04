package products

import (
	"fmt"
)

type UnknownTypeError struct {
	RewardType    RewardType
	RewardTypeStr string
}

func (e *UnknownTypeError) Error() string {
	return fmt.Sprintf("unknown reward type: '%v' (%v)", e.RewardTypeStr, e.RewardType)
}

func NewUnknownTypeError(rewardType RewardType, rewardTypeStr string) error {
	return &UnknownTypeError{RewardType: rewardType, RewardTypeStr: rewardTypeStr}
}
