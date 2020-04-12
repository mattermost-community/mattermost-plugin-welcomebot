package main

import (
	"sync/atomic"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

const (
	botUsername    = "welcomebot"
	botDisplayName = "Welcomebot"
	botDescription = "A bot account created by the Welcomebot plugin."

	WELCOMEBOT_CHANNEL_WELCOME_KEY = "chanmsg_"
)

// Plugin represents the welcome bot plugin
type Plugin struct {
	plugin.MattermostPlugin

	// botUserID of the created bot account.
	botUserID string

	welcomeMessages atomic.Value
}

// OnActivate ensure the bot account exists
func (p *Plugin) OnActivate() error {
	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botUserID, appErr := p.Helpers.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	p.API.RegisterCommand(getCommand())

	return nil
}
