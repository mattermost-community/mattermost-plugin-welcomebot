package usecase

type CommandMessenger interface {
	PostCommandResponse(message string)
}
