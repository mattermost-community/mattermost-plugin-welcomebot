package gateway

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Wrapper around messenger with a user/channel/bot context

type CommandMessenger struct {
	api       plugin.API
	UserID    string
	BotUserID string
	ChannelID string
}

func NewCommandMessenger(p BotAPIProvider, args *model.CommandArgs) *CommandMessenger {
	return &CommandMessenger{
		api:       p.APIHandle(),
		BotUserID: p.BotUserIDHandle(),
		UserID:    args.UserId,
		ChannelID: args.ChannelId,
	}
}

func (m *CommandMessenger) PostCommandResponse(message string) {
	post := &model.Post{
		UserId:    m.BotUserID,
		ChannelId: m.ChannelID,
		Message:   message,
	}

	_ = m.api.SendEphemeralPost(m.UserID, post)
}
