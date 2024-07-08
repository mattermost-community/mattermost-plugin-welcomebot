package main

import "github.com/mattermost/mattermost/server/public/model"

// MessageTemplate represents all the data that can be used in the template for a welcomebot message
type MessageTemplate struct {
	WelcomeBot      *model.User
	User            *model.User
	Team            *model.Team
	Townsquare      *model.Channel
	DirectMessage   *model.Channel
	UserDisplayName string
}

// GlobalMessageTemplate represents all the data that can be used in the template for a welcomebot global message
type GlobalMessageTemplate struct {
	WelcomeBot      *model.User
	User            *model.User
	DirectMessage   *model.Channel
	UserDisplayName string
}
