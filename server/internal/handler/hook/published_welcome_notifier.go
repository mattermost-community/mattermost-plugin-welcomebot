package hook

import (
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase/hook"
	"github.com/mattermost/mattermost/server/public/model"
)

type PublishedWelcomeNotifier struct{}

func (n *PublishedWelcomeNotifier) Execute(p handler.BotAPIProvider, channelMember *model.ChannelMember) {
	hook.NotifyWithPublishedWelcome(
		p.Container().ChannelWelcomeRepo(),
		p.Container().ChannelRepo(),
		p.Container().Messenger(),
		channelMember,
	)
}
