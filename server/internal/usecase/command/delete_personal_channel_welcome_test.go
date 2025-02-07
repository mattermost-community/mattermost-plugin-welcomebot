package command

import (
	"testing"

	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestDeletePersonalChanelWelcome(t *testing.T) {
	channelID := "test-channel"

	setupMocks := func() (*usecase.MockCommandMessenger, *usecase.MockChannelWelcomeRepo) {
		m := new(usecase.MockCommandMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		m.On("PostCommandResponse", mock.Anything).Return()

		return m, wr
	}

	t.Run("happy path", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(&pmodel.ChannelWelcome{Message: "Hello, friend"}, nil, nil)
		wr.On("DeletePersonalChanelWelcome", channelID).Return(nil)

		DeletePersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome message has been deleted")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("no message", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(nil, nil)

		DeletePersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome message has not been set yet")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("error while deleting from store", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(&pmodel.ChannelWelcome{Message: "Hello, friend"}, nil)
		wr.On("DeletePersonalChanelWelcome", channelID).Return(&mmodel.AppError{Message: "DAMN"})

		DeletePersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while deleting the welcome message for the chanel: `DAMN`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("error while retreiving from store", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPersonalChanelWelcome", channelID).Return(nil, &mmodel.AppError{Message: "FOO"})

		DeletePersonalChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while retrieving the welcome message for the chanel: `FOO`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})
}
