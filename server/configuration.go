package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	actionTypeAutomatic = "automatic"
	actionTypeButton    = "button"
)

// ConfigMessageAction are actions that can be taken from the welcome message
type ConfigMessageAction struct {
	// The action type of button or automatic
	ActionType string

	// The text on the button if a button type
	ActionDisplayName string

	// The action name that should be URL safe
	ActionName string

	// The message that's display after this action was successful
	ActionSuccessfulMessage []string

	// The names of the channels that a users should be added to
	ChannelsAddedTo []string
}

// ConfigMessage represents the message to send in channel
type ConfigMessage struct {
	// This message will fire when it matches the supplied team
	TeamName string

	// Actions that can be taken with this message
	Actions []*ConfigMessageAction

	// The message to send.  This is a go template that can access any member in MessageTemplate
	Message []string

	// The message to send as a slack attachment.  This is a go template that can access any member in MessageTemplate
	AttachmentMessage []string

	// Number of seconds to wait before sending the message
	DelayInSeconds int

	// Whether or not to include guest users
	IncludeGuests bool
}

// Configuration from config.json
type Configuration struct {
	WelcomeMessages []*ConfigMessage
}

// List of the welcome messages from the configuration
func (p *Plugin) getWelcomeMessages() []*ConfigMessage {
	return p.welcomeMessages.Load().([]*ConfigMessage)
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var c Configuration

	if err := p.API.LoadPluginConfiguration(&c); err != nil {
		p.API.LogError(err.Error())
		return err
	}

	p.welcomeMessages.Store(c.WelcomeMessages)

	return nil
}

// Takes a teamID to construct the correct key, and then retrieve the necessary value from the pair stored in the DB
func (p *Plugin) GetTeamWelcomeMessageFromKV(teamID string) ([]byte, *model.AppError) {
	key := makeTeamWelcomeMessageKey(teamID)
	return p.API.KVGet(key)
}

func makeTeamWelcomeMessageKey(teamID string) string {
	return fmt.Sprintf("%s%s", welcomebotTeamWelcomeKey, teamID)
}
