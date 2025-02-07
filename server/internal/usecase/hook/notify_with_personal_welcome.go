package hook

import (
	"time"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func NotifyWithPersonalWelcome(
	wr usecase.ChannelWelcomeRepo,
	cr usecase.ChannelRepo,
	m usecase.Messenger,
	channelMember *model.ChannelMember,
) {
	if channelInfo, appErr := cr.Get(channelMember.ChannelId); appErr != nil {
		mlog.Error(
			"error occurred while checking the type of the chanel",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	} else if channelInfo.Type == model.ChannelTypePrivate {
		return
	}

	welcome, appErr := wr.GetPersonalChanelWelcome(channelMember.ChannelId)
	if appErr != nil {
		mlog.Error(
			"error occurred while retrieving the welcome message",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	}

	if welcome == nil {
		return
	}

	dmChannel, err := cr.GetDirect(channelMember.UserId)
	if err != nil {
		mlog.Error(
			"error occurred while creating direct channel to the user",
			mlog.String("UserId", channelMember.UserId),
			mlog.Err(err),
		)
		return
	}

	if appErr := m.PostDirect(dmChannel.Id, welcome.Message); appErr != nil {
		mlog.Error("failed to post welcome message to the channel",
			mlog.String("channelId", dmChannel.Id),
			mlog.Err(appErr),
		)
	}

	time.Sleep(1 * time.Second)
	m.PostChannelEphemeral(channelMember.ChannelId, channelMember.UserId, welcome.Message)
}
