package handler

import (
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
)

type DependencyContainer interface {
	ChannelRepo() usecase.ChannelRepo
	ChannelWelcomeRepo() usecase.ChannelWelcomeRepo
	NewCommandMessenger(*model.CommandArgs) usecase.CommandMessenger
	Messenger() usecase.Messenger
}
