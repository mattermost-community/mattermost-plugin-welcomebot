package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(c *plugin.Context, teamMember *model.TeamMember, actor *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	for _, message := range p.getWelcomeMessages() {
		switch message.TeamName {
		case data.Team.Name:
			var teamNamesArr = strings.Split(message.TeamName, ",")
			for _, name := range teamNamesArr {
				tn := strings.TrimSpace(name)
				if tn == data.Team.Name {
					go p.processWelcomeMessage(*data, *message)
				}
			}
		case "*":
			go p.processWelcomeMessage(*data, *message)
		default:
			p.API.LogError("Couldn't find the message for the team")
		}
	}
}

// UserHasJoinedChannel is invoked after the membership has been committed to
// the database. If actor is not nil, the user was invited to the channel by
// the actor.
func (p *Plugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, _ *model.User) {
	if channelInfo, appErr := p.API.GetChannel(channelMember.ChannelId); appErr != nil {
		mlog.Error(
			"error occurred while checking the type of the chanel",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	} else if channelInfo.Type == model.CHANNEL_PRIVATE {
		return
	}

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, channelMember.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		mlog.Error(
			"error occurred while retrieving the welcome message",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	}

	if data == nil {
		// No welcome message for the given channel
		return
	}

	dmChannel, err := p.API.GetDirectChannel(channelMember.UserId, p.botUserID)
	if err != nil {
		mlog.Error(
			"error occurred while creating direct channel to the user",
			mlog.String("UserId", channelMember.UserId),
			mlog.Err(err),
		)
		return
	}

	// We send a DM and an opportunistic ephemeral message to the channel. See
	// the discussion at the link below for more details:
	// https://github.com/mattermost/mattermost-plugin-welcomebot/pull/31#issuecomment-611691023
	postDM := &model.Post{
		UserId:    p.botUserID,
		ChannelId: dmChannel.Id,
		Message:   string(data),
	}
	if _, appErr := p.API.CreatePost(postDM); appErr != nil {
		mlog.Error("failed to post welcome message to the channel",
			mlog.String("channelId", dmChannel.Id),
			mlog.Err(appErr),
		)
	}

	postChannel := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelMember.ChannelId,
		Message:   string(data),
	}
	time.Sleep(1 * time.Second)
	_ = p.API.SendEphemeralPost(channelMember.UserId, postChannel)
}
