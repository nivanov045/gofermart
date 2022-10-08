package models

import (
	"errors"
	"fmt"
)

var ErrIncorrectRewardValue = errors.New("incorrect reward value")

type UnknownRewardTypeError struct {
	RewardType    RewardType
	RewardTypeStr string
}

func (e UnknownRewardTypeError) Error() string {
	return fmt.Sprintf("unknown reward type: '%v' (%v)", e.RewardTypeStr, e.RewardType)
}

func NewUnknownTypeError(rewardType RewardType, rewardTypeStr string) error {
	return &UnknownRewardTypeError{RewardType: rewardType, RewardTypeStr: rewardTypeStr}
}
