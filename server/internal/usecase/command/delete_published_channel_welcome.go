package command

import (
	"fmt"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
)

func DeletePublishedChanelWelcome(
	m usecase.CommandMessenger,
	wr usecase.ChannelWelcomeRepo,
	channelID string,
) {
	welcome, err := wr.GetPublishedChanelWelcome(channelID)

	if err != nil {
		msg := fmt.Sprintf("error occurred while retrieving the welcome message for the chanel: `%s`", err)
		m.PostCommandResponse(msg)
		return
	}

	if welcome == nil {
		m.PostCommandResponse("welcome message has not been set yet")
		return
	}

	if err := wr.DeletePublishedChanelWelcome(channelID); err != nil {
		msg := fmt.Sprintf("error occurred while deleting the welcome message for the chanel: `%s`", err)
		m.PostCommandResponse(msg)

		return
	}

	m.PostCommandResponse("welcome message has been deleted")
}
