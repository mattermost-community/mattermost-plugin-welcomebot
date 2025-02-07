package main

import (
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/core"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/gateway"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/repo"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
)

type Container struct {
	plugin *core.Plugin
}

func (c *Container) ChannelRepo() usecase.ChannelRepo {
	return repo.NewChannelRepo(c.plugin)
}

func (c *Container) ChannelWelcomeRepo() usecase.ChannelWelcomeRepo {
	return repo.NewChannelWelcomeRepo(c.plugin)
}

func (c *Container) Messenger() usecase.Messenger {
	return gateway.NewMessenger(c.plugin)
}

func (c *Container) NewCommandMessenger(args *model.CommandArgs) usecase.CommandMessenger {
	return gateway.NewCommandMessenger(c.plugin, args)
}
