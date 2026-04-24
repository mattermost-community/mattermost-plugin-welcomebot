package main

import (
	"sync/atomic"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

const (
	botUsername    = "welcomebot"
	botDisplayName = "Welcomebot"
	botDescription = "A bot account created by the Welcomebot plugin."

	welcomebotChannelWelcomeKey = "chanmsg_"
)

// Plugin represents the welcome bot plugin.
// Field order is chosen to minimize GC pointer-scan bytes (fieldalignment):
// pointer-containing fields first, scalar fields last.
type Plugin struct {
	plugin.MattermostPlugin

	welcomeMessages atomic.Value

	client *pluginapi.Client

	// botUserID of the created bot account.
	botUserID string

	// channelWelcomeAutoJoinDelay is the number of seconds to wait before sending
	// the channel welcome ephemeral post. Stored atomically so it is safe to read
	// from hook goroutines without locking.
	channelWelcomeAutoJoinDelay atomic.Int64
}

// OnActivate ensures the bot account exists
func (p *Plugin) OnActivate() error {
	// Ensure welcomeMessages is always initialized before hooks can fire.
	// getWelcomeMessages() does a bare type assertion on Load() — if Store was
	// never called the assertion panics.
	p.welcomeMessages.Store([]*ConfigMessage{})

	if err := p.OnConfigurationChange(); err != nil {
		p.API.LogWarn("failed to load initial configuration on activate, using defaults",
			"error", err.Error(),
		)
	}

	p.client = pluginapi.NewClient(p.API, p.Driver)

	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botUserID, appErr := p.client.Bot.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	err := p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}
