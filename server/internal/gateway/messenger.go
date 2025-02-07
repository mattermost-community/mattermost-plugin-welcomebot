package gateway

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type Messenger struct {
	api       plugin.API
	BotUserID string
}

func NewMessenger(p BotAPIProvider) *Messenger {
	return &Messenger{
		api:       p.APIHandle(),
		BotUserID: p.BotUserIDHandle(),
	}
}

func (m *Messenger) PostDirect(channelID string, message string) *model.AppError {
	post := &model.Post{
		UserId:    m.BotUserID,
		ChannelId: channelID,
		Message:   message,
	}

	_, appErr := m.api.CreatePost(post)

	return appErr
}

func (m *Messenger) PostChannelEphemeral(channelID string, userID string, message string) {
	post := &model.Post{
		UserId:    m.BotUserID,
		ChannelId: channelID,
		Message:   message,
	}

	m.api.SendEphemeralPost(userID, post)
}

func (m *Messenger) Post(channelID string, message string) *model.AppError {
	post := &model.Post{
		UserId:    m.BotUserID,
		ChannelId: channelID,
		Message:   message,
	}

	_, appErr := m.api.CreatePost(post)

	return appErr
}
