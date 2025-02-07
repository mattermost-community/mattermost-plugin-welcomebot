package repo

import (
	"testing"

	mockAPI "github.com/evrone-erp/mattermost-plugin-welcomebot/server/mocks/mattermost/server/public/plugin/API"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
)

func setupChannelRepo() (*ChannelRepo, *mockAPI.MockAPI) {
	api := new(mockAPI.MockAPI)
	r := ChannelRepo{
		botUserID: "bot-user-id",
		api:       api,
	}

	return &r, api
}

func TestChannelRepoGet(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, api := setupChannelRepo()

		channel := &model.Channel{Id: "channel-id"}
		api.On("GetChannel", "channel-id").Return(channel, nil)

		result, err := r.Get("channel-id")

		assert.Equal(t, channel, result)
		assert.Nil(t, err)
	})
}

func TestChannelRepoGetDirect(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, api := setupChannelRepo()

		channel := &model.Channel{Id: "channel-id"}
		api.On("GetDirectChannel", "user-id", "bot-user-id").Return(channel, nil)

		result, err := r.GetDirect("user-id")

		assert.Equal(t, channel, result)
		assert.Nil(t, err)
	})
}
