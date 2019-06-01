package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

const welcomeBotUsername = "welcomebot"
const actionTypeAutomatic = "automatic"
const actionTypeButton = "button"

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
	// Number of seconds to wait before sending the message
	DelayInSeconds int

	// The message to send.  This is a go template that can access any member in MessageTemplate
	Message []string

	// The message to send as a slack attachment.  This is a go template that can access any member in MessageTemplate
	AttachmentMessage []string

	// This message will fire when it matches the supplied team
	TeamName string

	// Actions that can be taken with this message
	Actions []*ConfigMessageAction
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

	if err := p.ensureWelcomeBotUser(); err != nil {
		p.API.LogError(err.Error())
		return err
	}

	return nil
}

func (p *Plugin) ensureWelcomeBotUser() *model.AppError {
	var err *model.AppError

	user, _ := p.API.GetUserByUsername(welcomeBotUsername)

	// Ensure the configured user exists.
	if user == nil {
		randBytes := make([]byte, 15)
		rand.Read(randBytes)
		password := base64.StdEncoding.EncodeToString(randBytes)
		fmt.Println(password)

		user, err = p.API.CreateUser(&model.User{
			Username:  welcomeBotUsername,
			Password:  password,
			Email:     fmt.Sprintf("%s@mattermost.com", welcomeBotUsername),
			Nickname:  "Welcome Bot",
			FirstName: "Welcome",
			LastName:  "Bot",
			Position:  "Bot",
		})

		if err != nil {
			return err
		}
	}

	p.welcomeBotUserID = user.Id

	return nil
}
