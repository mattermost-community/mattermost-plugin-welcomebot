package command

import (
	"fmt"
	"strings"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
)

func SetPersonalChanelWelcome(
	m usecase.CommandMessenger,
	wr usecase.ChannelWelcomeRepo,
	cr usecase.ChannelRepo,
	fullCommand string,
	channelID string,
) {
	channel, appErr := cr.Get(channelID)
	if appErr != nil {
		response := fmt.Sprintf("error occurred while checking the type of the chanelId `%s`: `%s`", channelID, appErr)
		m.PostCommandResponse(response)
		return
	}

	if channel.Type == model.ChannelTypePrivate {
		m.PostCommandResponse("welcome messages are not supported for direct channels")
		return
	}

	parsedCommand := strings.SplitN(fullCommand, "set_personal_channel_welcome", 2)

	if len(parsedCommand) != 2 {
		response := fmt.Sprintf("error ocured while parsing command %s", fullCommand)
		m.PostCommandResponse(response)

		return
	}

	message := parsedCommand[1]
	message = strings.TrimSpace(message)

	if message == "" {
		m.PostCommandResponse("unable to store empty message")
		return
	}

	if appErr := wr.SetPersonalChanelWelcome(channel.Id, message); appErr != nil {
		response := fmt.Sprintf("error occurred while storing the welcome message for the chanel: `%s`", appErr)
		m.PostCommandResponse(response)
		return
	}

	response := fmt.Sprintf("stored the welcome message:\n%s", message)
	m.PostCommandResponse(response)
}
