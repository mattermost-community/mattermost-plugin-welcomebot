package usecase

import (
	"github.com/mattermost/mattermost/server/public/model"
)

type Messenger interface {
	PostDirect(channelID string, message string) *model.AppError
	PostChannelEphemeral(channelID string, userID string, message string)
	Post(channelID string, message string) *model.AppError
}
