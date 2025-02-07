package usecase

import (
	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	mmodel "github.com/mattermost/mattermost/server/public/model"
)

type ChannelWelcomeRepo interface {
	GetPersonalChanelWelcome(channelID string) (*pmodel.ChannelWelcome, *mmodel.AppError)
	DeletePersonalChanelWelcome(channelID string) *mmodel.AppError
	SetPersonalChanelWelcome(channelID string, message string) *mmodel.AppError

	GetPublishedChanelWelcome(channelID string) (*pmodel.ChannelWelcome, *mmodel.AppError)
	DeletePublishedChanelWelcome(channelID string) *mmodel.AppError
	SetPublishedChanelWelcome(channelID string, message string) *mmodel.AppError

	ListChannelsWithWelcome() ([]string, []string, *mmodel.AppError)
}
