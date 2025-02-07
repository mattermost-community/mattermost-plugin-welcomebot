package gateway

import (
	"testing"

	mockAPI "github.com/evrone-erp/mattermost-plugin-welcomebot/server/mocks/mattermost/server/public/plugin/API"
	"github.com/mattermost/mattermost/server/public/model"
)

func setupCommandMessenger() (*CommandMessenger, *mockAPI.MockAPI) {
	api := new(mockAPI.MockAPI)
	m := CommandMessenger{
		api:       api,
		BotUserID: "bot-user-id",
		UserID:    "user-id",
		ChannelID: "channel-id",
	}

	return &m, api
}

func TestMessengerPostCommandResponse(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m, api := setupCommandMessenger()

		post := &model.Post{
			UserId:    "bot-user-id",
			ChannelId: "channel-id",
			Message:   "hello",
		}

		api.On("SendEphemeralPost", "user-id", post).Return(post, nil)
		m.PostCommandResponse("hello")

		api.AssertNumberOfCalls(t, "SendEphemeralPost", 1)
	})
}
