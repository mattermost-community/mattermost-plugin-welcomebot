package hook

import (
	"testing"

	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestNotifyWithPublishedWelcome(t *testing.T) {
	userID := "user-id"

	groupChannel := &mmodel.Channel{
		Id:   "group-channel-id",
		Type: mmodel.ChannelTypeOpen,
	}

	channelMember := &mmodel.ChannelMember{
		ChannelId: "group-channel-id",
		UserId:    userID,
	}

	setupMocks := func() (*usecase.MockChannelWelcomeRepo, *usecase.MockChannelRepo, *usecase.MockMessenger) {
		m := new(usecase.MockMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		cr := new(usecase.MockChannelRepo)

		m.On("Post", mock.Anything, mock.Anything).Return(nil)

		return wr, cr, m
	}

	t.Run("happy path", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		wr.On("GetPublishedChanelWelcome", "group-channel-id").Return(&pmodel.ChannelWelcome{Message: "Hello, friend!"}, nil)

		NotifyWithPublishedWelcome(wr, cr, m, channelMember)

		m.AssertCalled(t, "Post", "group-channel-id", "Hello, friend!")
		m.AssertNumberOfCalls(t, "Post", 1)
	})

	t.Run("no stored message", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		wr.On("GetPublishedChanelWelcome", "group-channel-id").Return(nil, nil)

		NotifyWithPublishedWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "Post", 0)
	})

	t.Run("error while fetching published welcome", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		wr.On("GetPublishedChanelWelcome", "group-channel-id").Return(nil, &mmodel.AppError{Message: "foo"})

		NotifyWithPublishedWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "Post", 0)
	})

	t.Run("errro while fetching channel", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(nil, &mmodel.AppError{Message: "foo"})

		NotifyWithPublishedWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "Post", 0)
	})
}
