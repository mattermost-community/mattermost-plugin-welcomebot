package command

import (
	"testing"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestDeletePublishedChanelWelcome(t *testing.T) {
	channelID := "test-channel"

	setupMocks := func() (*usecase.MockCommandMessenger, *usecase.MockChannelWelcomeRepo) {
		m := new(usecase.MockCommandMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		m.On("PostCommandResponse", mock.Anything).Return()

		return m, wr
	}

	t.Run("happy path", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPublishedChanelWelcome", channelID).Return(&model.ChannelWelcome{Message: "Hello, friend"}, nil)
		wr.On("DeletePublishedChanelWelcome", channelID).Return(nil)

		DeletePublishedChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome message has been deleted")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("no message", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPublishedChanelWelcome", channelID).Return(nil, nil)

		DeletePublishedChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "welcome message has not been set yet")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("error while deleting from store", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPublishedChanelWelcome", channelID).Return(&model.ChannelWelcome{Message: "useful"}, nil)
		wr.On("DeletePublishedChanelWelcome", channelID).Return(&mmodel.AppError{Message: "DAMN"})

		DeletePublishedChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while deleting the welcome message for the chanel: `DAMN`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})

	t.Run("error while retreiving from store", func(t *testing.T) {
		m, wr := setupMocks()
		wr.On("GetPublishedChanelWelcome", channelID).Return(nil, &mmodel.AppError{Message: "FOO"})

		DeletePublishedChanelWelcome(m, wr, channelID)

		m.AssertCalled(t, "PostCommandResponse", "error occurred while retrieving the welcome message for the chanel: `FOO`")
		m.AssertNumberOfCalls(t, "PostCommandResponse", 1)
	})
}
