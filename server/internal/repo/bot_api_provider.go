package repo

import (
	"github.com/mattermost/mattermost/server/public/plugin"
)

type BotAPIProvider interface {
	APIHandle() plugin.API
	BotUserIDHandle() string
}
