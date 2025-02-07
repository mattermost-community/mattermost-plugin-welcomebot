package hook

import (
	"testing"

	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestNotifyWithPersonalWelcome(t *testing.T) {
	userID := "user-id"

	groupChannel := &mmodel.Channel{
		Id:   "group-channel-id",
		Type: mmodel.ChannelTypeOpen,
	}

	directChannel := &mmodel.Channel{
		Id:   "direct-channel-id",
		Type: mmodel.ChannelTypeDirect,
	}

	channelMember := &mmodel.ChannelMember{
		ChannelId: "group-channel-id",
		UserId:    userID,
	}

	setupMocks := func() (*usecase.MockChannelWelcomeRepo, *usecase.MockChannelRepo, *usecase.MockMessenger) {
		m := new(usecase.MockMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		cr := new(usecase.MockChannelRepo)

		m.On("PostChannelEphemeral", mock.Anything, mock.Anything, mock.Anything).Return()

		return wr, cr, m
	}

	t.Run("happy path", func(t *testing.T) {
		wr, cr, m := setupMocks()
		m.On("PostDirect", mock.Anything, mock.Anything).Return(nil)
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		cr.On("GetDirect", userID).Return(directChannel, nil)
		wr.On("GetPersonalChanelWelcome", "group-channel-id").Return(&pmodel.ChannelWelcome{Message: "Hello, friend!"}, nil)

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertCalled(t, "PostDirect", "direct-channel-id", "Hello, friend!")
		m.AssertCalled(t, "PostChannelEphemeral", "group-channel-id", "user-id", "Hello, friend!")

		m.AssertNumberOfCalls(t, "PostDirect", 1)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 1)
	})

	t.Run("no stored message", func(t *testing.T) {
		wr, cr, m := setupMocks()
		m.On("PostDirect", mock.Anything, mock.Anything).Return(nil)
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		wr.On("GetPersonalChanelWelcome", "group-channel-id").Return(nil, nil)
		cr.On("GetDirect", userID).Return(directChannel, nil)

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "PostDirect", 0)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 0)
	})

	t.Run("error during direct message", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		cr.On("GetDirect", userID).Return(directChannel, nil)
		wr.On("GetPersonalChanelWelcome", "group-channel-id").Return(&pmodel.ChannelWelcome{Message: "Hello, friend!"}, nil)
		m.On("PostDirect", "direct-channel-id", "Hello, friend!").Return(&mmodel.AppError{Message: "foo"})

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertCalled(t, "PostChannelEphemeral", "group-channel-id", "user-id", "Hello, friend!")

		m.AssertNumberOfCalls(t, "PostDirect", 1)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 1)
	})

	t.Run("error fetching direct", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		cr.On("GetDirect", userID).Return(nil, &mmodel.AppError{Message: "foo"})
		wr.On("GetPersonalChanelWelcome", "group-channel-id").Return(&pmodel.ChannelWelcome{Message: "Hello, friend!"}, nil)

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "PostDirect", 0)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 0)
	})

	t.Run("error while fetching personal welcome", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(groupChannel, nil)
		wr.On("GetPersonalChanelWelcome", "group-channel-id").Return(nil, &mmodel.AppError{Message: "foo"})

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "PostDirect", 0)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 0)
	})

	t.Run("errro while fetching channel", func(t *testing.T) {
		wr, cr, m := setupMocks()
		cr.On("Get", "group-channel-id").Return(nil, &mmodel.AppError{Message: "foo"})

		NotifyWithPersonalWelcome(wr, cr, m, channelMember)

		m.AssertNumberOfCalls(t, "PostDirect", 0)
		m.AssertNumberOfCalls(t, "PostChannelEphemeral", 0)
	})
}
