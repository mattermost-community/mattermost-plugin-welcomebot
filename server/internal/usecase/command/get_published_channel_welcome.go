package command

import (
	"fmt"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
)

func GetPublishedChanelWelcome(
	m usecase.CommandMessenger,
	wr usecase.ChannelWelcomeRepo,
	channelID string,
) {
	welcome, appErr := wr.GetPublishedChanelWelcome(channelID)

	if appErr != nil {
		message := fmt.Sprintf("error occurred while retrieving the welcome message for the chanel: `%s`", appErr)
		m.PostCommandResponse(message)
		return
	}

	if welcome == nil {
		m.PostCommandResponse("welcome message has not been set yet")
		return
	}

	message := fmt.Sprintf("Welcome message is:\n%s", welcome.Message)
	m.PostCommandResponse(message)
}
