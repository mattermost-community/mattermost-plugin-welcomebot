package gateway

import (
	"testing"

	mockAPI "github.com/evrone-erp/mattermost-plugin-welcomebot/server/mocks/mattermost/server/public/plugin/API"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
)

func setupMessenger() (*Messenger, *mockAPI.MockAPI) {
	api := new(mockAPI.MockAPI)
	m := Messenger{
		BotUserID: "bot-user-id",
		api:       api,
	}

	return &m, api
}

func TestMessengerPostDirect(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m, api := setupMessenger()

		post := &model.Post{
			UserId:    "bot-user-id",
			ChannelId: "channel-id",
			Message:   "hello",
		}

		api.On("CreatePost", post).Return(post, nil)
		err := m.PostDirect("channel-id", "hello")

		assert.Nil(t, err)
	})
}

func TestMessengerPostChannelEphemeral(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m, api := setupMessenger()

		post := &model.Post{
			UserId:    "bot-user-id",
			ChannelId: "channel-id",
			Message:   "hello",
		}

		api.On("SendEphemeralPost", "user-id", post).Return(post, nil)
		m.PostChannelEphemeral("channel-id", "user-id", "hello")

		api.AssertNumberOfCalls(t, "SendEphemeralPost", 1)
	})
}

func TestMessengerPost(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m, api := setupMessenger()

		post := &model.Post{
			UserId:    "bot-user-id",
			ChannelId: "channel-id",
			Message:   "hello",
		}

		api.On("CreatePost", post).Return(post, nil)
		err := m.Post("channel-id", "hello")

		api.AssertNumberOfCalls(t, "CreatePost", 1)
		assert.Nil(t, err)
	})
}
