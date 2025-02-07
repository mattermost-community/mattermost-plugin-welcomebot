package command

import (
	"errors"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase/command"
	"github.com/mattermost/mattermost/server/public/model"
)

type GetPublishedChanelWelcome struct{}

func (c *GetPublishedChanelWelcome) Trigger() string {
	return "get_published_channel_welcome"
}

func (c *GetPublishedChanelWelcome) Help() string {
	return "`/welcomebot get_published_channel_welcome` - print the published welcome message set for the given channel (if any)"
}

func (c *GetPublishedChanelWelcome) Execute(p handler.BotAPIProvider, args *model.CommandArgs) {
	command.GetPublishedChanelWelcome(
		p.Container().NewCommandMessenger(args),
		p.Container().ChannelWelcomeRepo(),
		args.ChannelId,
	)
}

func (c *GetPublishedChanelWelcome) Validate(parameters []string) error {
	if len(parameters) > 0 {
		return errors.New("`get_published_channel_welcome` command does not accept any extra parameters")
	}

	return nil
}

func (c *GetPublishedChanelWelcome) AutocompleteData() *model.AutocompleteData {
	return model.NewAutocompleteData("get_published_channel_welcome", "", "Print the welcome message set for the channel")
}
