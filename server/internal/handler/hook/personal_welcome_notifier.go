package hook

import (
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/handler"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase/hook"
	"github.com/mattermost/mattermost/server/public/model"
)

type PersonalWelcomeNotifier struct{}

func (n *PersonalWelcomeNotifier) Execute(p handler.BotAPIProvider, channelMember *model.ChannelMember) {
	hook.NotifyWithPersonalWelcome(
		p.Container().ChannelWelcomeRepo(),
		p.Container().ChannelRepo(),
		p.Container().Messenger(),
		channelMember,
	)
}
