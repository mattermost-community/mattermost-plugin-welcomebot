package command

import (
	"fmt"
	"strings"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func ListChannelWelcomes(
	m usecase.CommandMessenger,
	wr usecase.ChannelWelcomeRepo,
	cr usecase.ChannelRepo,
) {
	personalIDs, publishedIDs, appErr := wr.ListChannelsWithWelcome()

	if appErr != nil {
		response := fmt.Sprintf("error occurred while listing channels: `%s`", appErr)
		m.PostCommandResponse(response)
		return
	}

	var builder strings.Builder

	personalChannels := fetchChannels(personalIDs, cr)
	publishedChannels := fetchChannels(publishedIDs, cr)

	if len(publishedChannels) > 0 {
		builder.WriteString("Channels with published welcome message:\n")

		for _, channel := range publishedChannels {
			row := fmt.Sprintf("~%s\n", channel.Name)
			builder.WriteString(row)
		}
	}

	if len(personalChannels) > 0 {
		builder.WriteString("Channels with personal welcome message:\n")

		for _, channel := range personalChannels {
			row := fmt.Sprintf("~%s\n", channel.Name)
			builder.WriteString(row)
		}
	}

	m.PostCommandResponse(builder.String())
}

func fetchChannels(list []string, cr usecase.ChannelRepo) []*model.Channel {
	result := make([]*model.Channel, 0, len(list))

	for _, channelID := range list {
		channel, appErr := cr.Get(channelID)

		if appErr != nil {
			mlog.Error(
				"error occurred while retreiving channel",
				mlog.String("channelId", channelID),
				mlog.Err(appErr),
			)
		} else {
			result = append(result, channel)
		}
	}

	return result
}
