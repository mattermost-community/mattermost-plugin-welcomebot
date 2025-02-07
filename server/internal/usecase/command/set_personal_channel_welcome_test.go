package command

import (
	"testing"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestSetPersonalChanelWelcome(t *testing.T) {
	channelID := "test-channel"

	setupMocks := func() (*usecase.MockCommandMessenger, *usecase.MockChannelWelcomeRepo, *usecase.MockChannelRepo) {
		m := new(usecase.MockCommandMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		cr := new(usecase.MockChannelRepo)
		m.On("PostCommandResponse", mock.Anything).Return()

		return m, wr, cr
	}

	validChannel := &model.Channel{
		Id:   channelID,
		Type: model.ChannelTypeOpen,
	}

	validCommand := "set_personal_channel_welcome foo bar keke   "

	t.Run("happy path", func(t *testing.T) {
		m, wr, cr := setupMocks()

		wr.On("SetPersonalChanelWelcome", mock.Anything, mock.Anything).Return(nil)
		cr.On("Get", channelID).Return(validChannel, nil)

		SetPersonalChanelWelcome(m, wr, cr, validCommand, channelID)

		m.AssertCalled(t, "PostCommandResponse", "stored the welcome message:\nfoo bar keke")
		wr.AssertCalled(t, "SetPersonalChanelWelcome", "test-channel", "foo bar keke")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 1)
	})

	t.Run("with broken command", func(t *testing.T) {
		m, wr, cr := setupMocks()

		wr.On("SetPersonalChanelWelcome", mock.Anything, mock.Anything).Return(nil)
		cr.On("Get", channelID).Return(validChannel, nil)

		SetPersonalChanelWelcome(m, wr, cr, "kek", channelID)

		m.AssertCalled(t, "PostCommandResponse", "error ocured while parsing command kek")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 0)
	})

	t.Run("with a command without message", func(t *testing.T) {
		m, wr, cr := setupMocks()

		wr.On("SetPersonalChanelWelcome", mock.Anything, mock.Anything).Return(nil)
		cr.On("Get", channelID).Return(validChannel, nil)

		SetPersonalChanelWelcome(m, wr, cr, "set_personal_channel_welcome     ", channelID)

		m.AssertCalled(t, "PostCommandResponse", "unable to store empty message")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 0)
	})

	t.Run("private channel", func(t *testing.T) {
		m, wr, cr := setupMocks()

		wr.On("SetPersonalChanelWelcome", mock.Anything, mock.Anything).Return(nil)
		privateChannel := &model.Channel{
			Id:   channelID,
			Type: model.ChannelTypePrivate,
		}
		cr.On("Get", channelID).Return(privateChannel, nil)

		SetPersonalChanelWelcome(m, wr, cr, validCommand, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome messages are not supported for direct channels")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 0)
	})

	t.Run("persist message error", func(t *testing.T) {
		m, wr, cr := setupMocks()

		wr.On("SetPersonalChanelWelcome", mock.Anything, mock.Anything).Return(&model.AppError{Message: "persist error"})
		cr.On("Get", channelID).Return(validChannel, nil)

		SetPersonalChanelWelcome(m, wr, cr, validCommand, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while storing the welcome message for the chanel: `persist error`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 1)
	})

	t.Run("retrieve channel error", func(t *testing.T) {
		m, wr, cr := setupMocks()
		cr.On("Get", channelID).Return(nil, &model.AppError{Message: "receiving error"})

		SetPersonalChanelWelcome(m, wr, cr, validCommand, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while checking the type of the chanelId `test-channel`: `receiving error`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
		wr.AssertNumberOfCalls(t, "SetPersonalChanelWelcome", 0)
	})
}
