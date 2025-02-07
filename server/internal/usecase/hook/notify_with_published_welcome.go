package hook

import (
	"time"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func NotifyWithPublishedWelcome(
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

	welcome, appErr := wr.GetPublishedChanelWelcome(channelMember.ChannelId)
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

	time.Sleep(1 * time.Second)
	appErr = m.Post(channelMember.ChannelId, welcome.Message)

	if appErr != nil {
		mlog.Error(
			"error ocuring while posting message to the channel",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	}
}
