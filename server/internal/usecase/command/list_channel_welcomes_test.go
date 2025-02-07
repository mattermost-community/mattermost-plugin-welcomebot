package command

import (
	"testing"

	"github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/usecase"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
)

func TestListChannelWelcomes(t *testing.T) {
	setupMocks := func() (*usecase.MockCommandMessenger, *usecase.MockChannelWelcomeRepo, *usecase.MockChannelRepo) {
		m := new(usecase.MockCommandMessenger)
		wr := new(usecase.MockChannelWelcomeRepo)
		cr := new(usecase.MockChannelRepo)
		m.On("PostCommandResponse", mock.Anything).Return()

		return m, wr, cr
	}

	t.Run("complete happy path", func(t *testing.T) {
		m, wr, cr := setupMocks()

		personalIDs := []string{"personalid-1", "personalid-2"}
		publishedIDs := []string{"published-1", "garbage"}

		wr.On("ListChannelsWithWelcome").Return(personalIDs, publishedIDs, nil)
		cr.On("Get", "personalid-1").Return(&model.Channel{Id: "personalid-1", Name: "pe1"}, nil)
		cr.On("Get", "personalid-2").Return(&model.Channel{Id: "personalid-2", Name: "pe2"}, nil)
		cr.On("Get", "published-1").Return(&model.Channel{Id: "published-1", Name: "pu1"}, nil)
		cr.On("Get", "garbage").Return(nil, &model.AppError{Message: "whatever"})

		ListChannelWelcomes(m, wr, cr)
		expectedMessage := "Channels with published welcome message:\n~pu1\nChannels with personal welcome message:\n~pe1\n~pe2\n"

		m.AssertCalled(t, "PostCommandResponse", expectedMessage)
	})
}
