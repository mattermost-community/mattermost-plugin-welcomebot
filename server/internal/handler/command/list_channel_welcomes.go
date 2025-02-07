package command

import (
	"errors"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase/command"
	"github.com/mattermost/mattermost/server/public/model"
)

type ListChannelWelcomes struct{}

func (c *ListChannelWelcomes) Trigger() string {
	return "list_channel_welcomes"
}

func (c *ListChannelWelcomes) Help() string {
	return "`/welcomebot list_channel_welcomes` - print all channels with configured welcome messages"
}

func (c *ListChannelWelcomes) Execute(p handler.BotAPIProvider, args *model.CommandArgs) {
	command.ListChannelWelcomes(
		p.Container().NewCommandMessenger(args),
		p.Container().ChannelWelcomeRepo(),
		p.Container().ChannelRepo(),
	)
}

func (c *ListChannelWelcomes) Validate(parameters []string) error {
	if len(parameters) > 0 {
		return errors.New("`list_channel_welcomes` command does not accept any extra parameters")
	}

	return nil
}

func (c *ListChannelWelcomes) AutocompleteData() *model.AutocompleteData {
	return model.NewAutocompleteData("list_channel_welcomes", "", "print all channels with configured welcome messages")
}
