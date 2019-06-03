package main

import (
	"sync/atomic"

	"github.com/mattermost/mattermost-server/plugin"
)

// Plugin represents the welcome bot plugin
type Plugin struct {
	plugin.MattermostPlugin

	welcomeBotUserID string

	welcomeMessages atomic.Value
}
