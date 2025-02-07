package repo

import (
	"github.com/stretchr/testify/assert"

	"testing"

	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	mockAPI "github.com/evrone-erp/mattermost-plugin-welcomebot/server/mocks/mattermost/server/public/plugin/API"
	mmodel "github.com/mattermost/mattermost/server/public/model"
)

func setupChannelWelcome() (*ChannelWelcome, *mockAPI.MockAPI) {
	m := new(mockAPI.MockAPI)
	r := ChannelWelcome{
		api: m,
	}

	return &r, m
}

func TestGetPersonalChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:personal:channel-id").Return([]byte("hello"), nil)
		result, err := r.GetPersonalChanelWelcome("channel-id")

		expected := &pmodel.ChannelWelcome{
			ID:        "welcomemsg:personal:channel-id",
			Type:      pmodel.WelcomeMessageTypePersonal,
			Message:   "hello",
			ChannelID: "channel-id",
		}

		assert.Equal(t, expected, result)
		assert.Nil(t, err)
	})

	t.Run("empty message", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:personal:channel-id").Return([]byte("  "), nil)
		result, err := r.GetPersonalChanelWelcome("channel-id")

		assert.Nil(t, result)
		assert.Nil(t, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:personal:channel-id").Return(nil, &mmodel.AppError{Message: "foo"})
		_, err := r.GetPersonalChanelWelcome("channel-id")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestGetPublishedChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:published:channel-id").Return([]byte("hello"), nil)
		result, err := r.GetPublishedChanelWelcome("channel-id")

		expected := &pmodel.ChannelWelcome{
			ID:        "welcomemsg:published:channel-id",
			Type:      pmodel.WelcomeMessageTypePublished,
			Message:   "hello",
			ChannelID: "channel-id",
		}

		assert.Equal(t, expected, result)
		assert.Nil(t, err)
	})

	t.Run("empty message", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:published:channel-id").Return([]byte("  "), nil)
		result, err := r.GetPublishedChanelWelcome("channel-id")

		assert.Nil(t, result)
		assert.Nil(t, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVGet", "welcomemsg:published:channel-id").Return(nil, &mmodel.AppError{Message: "foo"})
		_, err := r.GetPublishedChanelWelcome("channel-id")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestDeletePersonalChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVDelete", "welcomemsg:personal:channel-id").Return(nil)
		err := r.DeletePersonalChanelWelcome("channel-id")

		assert.Nil(t, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVDelete", "welcomemsg:personal:channel-id").Return(&mmodel.AppError{Message: "foo"})
		err := r.DeletePersonalChanelWelcome("channel-id")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestDeletePublishedChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVDelete", "welcomemsg:published:channel-id").Return(nil)
		err := r.DeletePublishedChanelWelcome("channel-id")

		assert.Nil(t, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVDelete", "welcomemsg:published:channel-id").Return(&mmodel.AppError{Message: "foo"})
		err := r.DeletePublishedChanelWelcome("channel-id")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestSetPersonalChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:personal:channel-id", []byte("newmsg")).Return(nil)
		err := r.SetPersonalChanelWelcome("channel-id", "newmsg")

		assert.Nil(t, err)
	})

	t.Run("empty message", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:personal:channel-id", []byte(" ")).Return(nil)
		err := r.SetPersonalChanelWelcome("channel-id", " ")

		assert.Equal(t, &mmodel.AppError{Message: "trying to store empty message"}, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:personal:channel-id", []byte("newmsg")).Return(&mmodel.AppError{Message: "foo"})
		err := r.SetPersonalChanelWelcome("channel-id", "newmsg")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestSetPublishedChanelWelcome(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:published:channel-id", []byte("newmsg")).Return(nil)
		err := r.SetPublishedChanelWelcome("channel-id", "newmsg")

		assert.Nil(t, err)
	})

	t.Run("empty message", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:published:channel-id", []byte(" ")).Return(nil)
		err := r.SetPublishedChanelWelcome("channel-id", " ")

		assert.Equal(t, &mmodel.AppError{Message: "trying to store empty message"}, err)
	})

	t.Run("error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVSet", "welcomemsg:published:channel-id", []byte("newmsg")).Return(&mmodel.AppError{Message: "foo"})
		err := r.SetPublishedChanelWelcome("channel-id", "newmsg")

		assert.Equal(t, "foo", err.Error())
	})
}

func TestListChannelsWithWelcome(t *testing.T) {
	perPage := 1000

	t.Run("happy path", func(t *testing.T) {
		r, m := setupChannelWelcome()
		firstPage := []string{
			"welcomemsg:published:first",
			"welcomemsg:personal:first",
			"welcomemsg:published:",
			"welcomemsg:published: ",
			"welcomemsg:second ",
			"garbage",
		}

		secondPage := []string{
			"welcomemsg:published:second",
			"welcomemsg:published:tree:times",
			"_",
			" ",
		}

		m.On("KVList", 0, perPage).Return(firstPage, nil)
		m.On("KVList", 1, perPage).Return(secondPage, nil)
		m.On("KVList", 2, perPage).Return([]string{}, nil)
		personal, published, err := r.ListChannelsWithWelcome()

		assert.Equal(t, []string{"first", "second"}, published)
		assert.Equal(t, []string{"first"}, personal)
		assert.Nil(t, err)
	})

	t.Run("low level error", func(t *testing.T) {
		r, m := setupChannelWelcome()
		m.On("KVList", 0, perPage).Return(nil, &mmodel.AppError{Message: "foo"})

		_, _, err := r.ListChannelsWithWelcome()

		assert.Equal(t, "foo", err.Error())
	})
}
