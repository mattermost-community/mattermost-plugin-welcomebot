package model

type ChannelWelcome struct {
	ID        string
	ChannelID string
	Message   string
	Type      WelcomeMessageType
}

type WelcomeMessageType string

const (
	WelcomeMessageTypePublished WelcomeMessageType = "PB"
	WelcomeMessageTypePersonal  WelcomeMessageType = "PR"
)
