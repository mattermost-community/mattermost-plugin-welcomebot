package core

import (
	"fmt"
	"sync/atomic"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

const (
	botUsername    = "welcomebot"
	botDisplayName = "Welcomebot"
	botDescription = "A bot account created by the Welcomebot plugin."
)

type CommandInterface interface {
	Trigger() string
	Help() string
	Execute(handler.BotAPIProvider, *model.CommandArgs)
	Validate([]string) error
	AutocompleteData() *model.AutocompleteData
}

type Hook interface {
	Execute(handler.BotAPIProvider, *model.ChannelMember)
}

// Plugin represents the welcome bot plugin
type Plugin struct {
	plugin.MattermostPlugin

	BotUserID string
	client    *pluginapi.Client
	// BotUserID of the created bot account.
	Manifest                  *model.Manifest
	welcomeMessages           atomic.Value
	commands                  map[string]CommandInterface
	container                 handler.DependencyContainer
	userHasJoinedChannelHooks []Hook
}

func NewPlugin(manifest *model.Manifest) Plugin {
	return Plugin{
		Manifest:                  manifest,
		commands:                  make(map[string]CommandInterface),
		userHasJoinedChannelHooks: make([]Hook, 0),
	}
}

func (p *Plugin) RegisterDependencyContainer(c handler.DependencyContainer) {
	p.container = c
}

func (p *Plugin) Container() handler.DependencyContainer {
	return p.container
}

func (p *Plugin) RegisterCommand(c CommandInterface) {
	_, conflict := p.commands[c.Trigger()]

	if conflict {
		msg := fmt.Sprintf("Command trigger %s already registered", c.Trigger())
		panic(msg)
	}

	p.commands[c.Trigger()] = c
}

func (p *Plugin) RegisterUserHasJoinedChannelHook(h Hook) {
	p.userHasJoinedChannelHooks = append(p.userHasJoinedChannelHooks, h)
}

func (p *Plugin) APIHandle() plugin.API {
	return p.API
}

func (p *Plugin) BotUserIDHandle() string {
	return p.BotUserID
}

// OnActivate ensure the bot account exists
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	BotUserID, appErr := p.client.Bot.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.BotUserID = BotUserID

	err := p.API.RegisterCommand(p.GetCommand())
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}
