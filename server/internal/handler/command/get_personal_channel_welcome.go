package command

import (
	"errors"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase/command"
	"github.com/mattermost/mattermost/server/public/model"
)

type GetPersonalChanelWelcome struct{}

func (c *GetPersonalChanelWelcome) Trigger() string {
	return "get_personal_channel_welcome"
}

func (c *GetPersonalChanelWelcome) Help() string {
	return "`/welcomebot get_personal_channel_welcome` - print the personal welcome message set for the given channel (if any)"
}

func (c *GetPersonalChanelWelcome) Execute(p handler.BotAPIProvider, args *model.CommandArgs) {
	command.GetPersonalChanelWelcome(
		p.Container().NewCommandMessenger(args),
		p.Container().ChannelWelcomeRepo(),
		args.ChannelId,
	)
}

func (c *GetPersonalChanelWelcome) Validate(parameters []string) error {
	if len(parameters) > 0 {
		return errors.New("`get_personal_channel_welcome` command does not accept any extra parameters")
	}

	return nil
}

func (c *GetPersonalChanelWelcome) AutocompleteData() *model.AutocompleteData {
	return model.NewAutocompleteData("get_personal_channel_welcome", "", "Print the welcome message set for the channel")
}
