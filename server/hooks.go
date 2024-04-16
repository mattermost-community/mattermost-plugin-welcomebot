package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(c *plugin.Context, teamMember *model.TeamMember, actor *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	teamMessage, appErr := p.GetTeamWelcomeMessageFromKV(teamMember.TeamId)
	if appErr != nil {
		mlog.Error(
			"error occurred while retrieving the welcome message",
			mlog.String("teamId", teamMember.TeamId),
			mlog.Err(appErr),
		)
		return
	}

	if teamMessage == nil {
		// No dynamic welcome message for the given team, so we check if one has been set in the config.json
		for _, message := range p.getWelcomeMessages() {
			if message.TeamName == data.Team.Name {
				go p.processWelcomeMessage(*data, *message)
			}
		}
		return
	}

	// We send a DM and an opportunistic ephemeral message to the channel. See
	// the discussion at the link below for more details:
	// https://github.com/mattermost/mattermost-plugin-welcomebot/pull/31#issuecomment-611691023
	postDM := &model.Post{
		UserId:    p.botUserID,
		ChannelId: data.DirectMessage.Id,
		Message:   string(teamMessage),
	}
	if _, appErr := p.API.CreatePost(postDM); appErr != nil {
		mlog.Error("failed to post welcome message to the channel",
			mlog.String("channelId", data.DirectMessage.Id),
			mlog.Err(appErr),
		)
	}
}

// UserHasJoinedChannel is invoked after the membership has been committed to
// the database. If actor is not nil, the user was invited to the channel by
// the actor.
func (p *Plugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	channelInfo, appErr := p.API.GetChannel(channelMember.ChannelId)
	if appErr != nil {
		mlog.Error(
			"error occurred while checking the type of the chanel",
			mlog.String("channelId", channelMember.ChannelId),
			mlog.Err(appErr),
		)
		return
	} else if channelInfo.Type == model.ChannelTypePrivate {
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
	channelAddMessage := "# Welcome to the " + channelInfo.DisplayName + " channel. \n\n"

	postDM := &model.Post{
		UserId:    p.botUserID,
		ChannelId: dmChannel.Id,
		Message:   channelAddMessage + string(data),
	}
	if _, appErr := p.API.CreatePost(postDM); appErr != nil {
		mlog.Error("failed to post welcome message to the channel",
			mlog.String("channelId", dmChannel.Id),
			mlog.Err(appErr),
		)
	}

	// Commented out in case we do not want the user immediately greeted with an ephemeral message
	// postChannel := &model.Post{
	// 	UserId:    p.botUserID,
	// 	ChannelId: channelMember.ChannelId,
	// 	Message:   string(data),
	// }

	// if actor.Id == channelMember.UserId {
	// 	time.Sleep(3 * time.Second)
	// } else {
	// 	time.Sleep(15 * time.Second)
	// }

	// _ = p.API.SendEphemeralPost(channelMember.UserId, postChannel)
}
