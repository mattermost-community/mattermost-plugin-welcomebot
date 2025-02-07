package command

import (
	"testing"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestGetPersonalChanelWelcome(t *testing.T) {
	channelID := "test-channel"

	setupMocks := func() (*usecase.MockCommandMessenger, *usecase.MockChannelWelcomeRepo) {
		m := new(usecase.MockCommandMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		m.On("PostCommandResponse", mock.Anything).Return()

		return m, wr
	}

	t.Run("message exists", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(&model.ChannelWelcome{Message: "Hello, friend!"}, nil)

		GetPersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "Welcome message is:\nHello, friend!")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("store returns an error", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(nil, &mmodel.AppError{Message: "some error"})

		GetPersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while retrieving the welcome message for the chanel: `some error`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("message not set", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(nil, nil)

		GetPersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome message has not been set yet")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})
}
