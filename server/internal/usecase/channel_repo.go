package usecase

import (
	"github.com/mattermost/mattermost/server/public/model"
)

type ChannelRepo interface {
	Get(string) (*model.Channel, *model.AppError)
	GetDirect(string) (*model.Channel, *model.AppError)
}
