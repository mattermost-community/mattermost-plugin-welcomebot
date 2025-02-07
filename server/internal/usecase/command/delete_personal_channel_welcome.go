package command

import (
	"fmt"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
)

func DeletePersonalChanelWelcome(
	m usecase.CommandMessenger,
	wr usecase.ChannelWelcomeRepo,
	channelID string,
) {
	welcome, err := wr.GetPersonalChanelWelcome(channelID)

	if err != nil {
		msg := fmt.Sprintf("error occurred while retrieving the welcome message for the chanel: `%s`", err)
		m.PostCommandResponse(msg)
		return
	}

	if welcome == nil {
		m.PostCommandResponse("welcome message has not been set yet")
		return
	}

	if err := wr.DeletePersonalChanelWelcome(channelID); err != nil {
		msg := fmt.Sprintf("error occurred while deleting the welcome message for the chanel: `%s`", err)
		m.PostCommandResponse(msg)

		return
	}

	m.PostCommandResponse("welcome message has been deleted")
}
