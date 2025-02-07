package repo

import (
	"fmt"
	"strings"

	pmodel "github.com/evrone-erp/mattermost-plugin-welcomebot/server/internal/model"
	mmodel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type ChannelWelcome struct {
	api plugin.API
}

func NewChannelWelcomeRepo(p BotAPIProvider) *ChannelWelcome {
	return &ChannelWelcome{
		api: p.APIHandle(),
	}
}

const (
	generalPrefix   = "welcomemsg"
	personalPrefix  = "personal"
	publishedPrefix = "published"
)

func (r *ChannelWelcome) GetPersonalChanelWelcome(channelID string) (*pmodel.ChannelWelcome, *mmodel.AppError) {
	key := personalChannelKey(channelID)

	data, err := r.api.KVGet(key)

	if err != nil {
		return nil, err
	}

	msg := string(data)

	if isEmptyMsg(msg) {
		return nil, nil
	}

	return &pmodel.ChannelWelcome{
		ID:        key,
		Message:   msg,
		Type:      pmodel.WelcomeMessageTypePersonal,
		ChannelID: channelID,
	}, nil
}

func (r *ChannelWelcome) GetPublishedChanelWelcome(channelID string) (*pmodel.ChannelWelcome, *mmodel.AppError) {
	key := publishedChannelKey(channelID)

	data, err := r.api.KVGet(key)

	if err != nil {
		return nil, err
	}

	msg := string(data)

	if isEmptyMsg(msg) {
		return nil, nil
	}

	return &pmodel.ChannelWelcome{
		ID:        key,
		Message:   msg,
		Type:      pmodel.WelcomeMessageTypePublished,
		ChannelID: channelID,
	}, nil
}

func (r *ChannelWelcome) DeletePersonalChanelWelcome(channelID string) *mmodel.AppError {
	key := personalChannelKey(channelID)

	return r.api.KVDelete(key)
}

func (r *ChannelWelcome) DeletePublishedChanelWelcome(channelID string) *mmodel.AppError {
	key := publishedChannelKey(channelID)

	return r.api.KVDelete(key)
}

func (r *ChannelWelcome) SetPersonalChanelWelcome(channelID string, message string) *mmodel.AppError {
	if isEmptyMsg(message) {
		return &mmodel.AppError{Message: "trying to store empty message"}
	}

	key := personalChannelKey(channelID)

	return r.api.KVSet(key, []byte(message))
}

func (r *ChannelWelcome) SetPublishedChanelWelcome(channelID string, message string) *mmodel.AppError {
	if isEmptyMsg(message) {
		return &mmodel.AppError{Message: "trying to store empty message"}
	}

	key := publishedChannelKey(channelID)

	return r.api.KVSet(key, []byte(message))
}

func (r *ChannelWelcome) ListChannelsWithWelcome() ([]string, []string, *mmodel.AppError) {
	page := 0
	perPage := 1000
	keys := make([]string, 0)

	for {
		pageKeys, appErr := r.api.KVList(page, perPage)

		if appErr != nil {
			return nil, nil, appErr
		}

		if len(pageKeys) == 0 {
			break
		}

		keys = append(keys, pageKeys...)
		page++
	}

	publishedChannels := make([]string, 0)
	personalChannels := make([]string, 0)

	for _, key := range keys {
		parts := strings.Split(key, ":")

		if len(parts) == 3 && (parts[0] == generalPrefix) && (parts[1] == personalPrefix || parts[1] == publishedPrefix) {
			name := strings.TrimSpace(parts[2])
			if name != "" {
				if parts[1] == publishedPrefix {
					publishedChannels = append(publishedChannels, parts[2])
				} else if parts[1] == personalPrefix {
					personalChannels = append(personalChannels, parts[2])
				}
			}
		}
	}

	return personalChannels, publishedChannels, nil
}

func personalChannelKey(channelID string) string {
	return fmt.Sprintf("%s:%s:%s", generalPrefix, personalPrefix, channelID)
}

func isEmptyMsg(msg string) bool {
	return strings.TrimSpace(msg) == ""
}

func publishedChannelKey(channelID string) string {
	return fmt.Sprintf("%s:%s:%s", generalPrefix, publishedPrefix, channelID)
}
