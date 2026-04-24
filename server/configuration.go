package main

import "time"

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

// defaultAutoJoinDelaySeconds is used when ChannelWelcomeAutoJoinDelaySeconds
// is not set or is set to zero. 5 seconds gives most clients enough time to
// render the channel before the ephemeral arrives.
const defaultAutoJoinDelaySeconds = 5

// Configuration from config.json
type Configuration struct {
	WelcomeMessages []*ConfigMessage

	// ChannelWelcomeAutoJoinDelaySeconds controls how long the plugin waits before
	// sending the channel welcome ephemeral. Increase if users report that the
	// welcome appears before the channel has fully loaded. Defaults to 5 seconds.
	ChannelWelcomeAutoJoinDelaySeconds int
}

// getWelcomeMessages returns the list of welcome messages from configuration.
func (p *Plugin) getWelcomeMessages() []*ConfigMessage {
	return p.welcomeMessages.Load().([]*ConfigMessage)
}

// getChannelWelcomeAutoJoinDelay returns the configured delay, falling back to
// defaultAutoJoinDelaySeconds if the value was not set.
func (p *Plugin) getChannelWelcomeAutoJoinDelay() time.Duration {
	d := p.channelWelcomeAutoJoinDelay.Load()
	if d <= 0 {
		d = defaultAutoJoinDelaySeconds
	}
	return time.Duration(d) * time.Second
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var c Configuration

	if err := p.API.LoadPluginConfiguration(&c); err != nil {
		p.API.LogError(err.Error())
		return err
	}

	if c.WelcomeMessages == nil {
		c.WelcomeMessages = []*ConfigMessage{}
	}
	p.welcomeMessages.Store(c.WelcomeMessages)

	delay := c.ChannelWelcomeAutoJoinDelaySeconds
	if delay <= 0 {
		delay = defaultAutoJoinDelaySeconds
	}
	p.channelWelcomeAutoJoinDelay.Store(int64(delay))

	return nil
}
