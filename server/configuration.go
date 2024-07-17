package main

import (
	"encoding/json"
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

// Custom JSON unmarshal function for Configuration.
// To allow for configuration in the System Console we need to support Configuration.WelcomeMessages
// being either a string or slice of *ConfigMessage.
func (c *Configuration) UnmarshalJSON(b []byte) error {
	var s struct {
		WelcomeMessages string
	}

	err := json.Unmarshal(b, &s)
	if err == nil {
		var configMessages []*ConfigMessage
		err = json.Unmarshal([]byte(s.WelcomeMessages), &configMessages)
		if err != nil {
			return err
		}

		c.WelcomeMessages = configMessages

		return nil
	}

	var tc struct {
		WelcomeMessages []*ConfigMessage
	}

	err = json.Unmarshal(b, &tc)
	if err != nil {
		return err
	}

	c.WelcomeMessages = tc.WelcomeMessages

	return nil
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
