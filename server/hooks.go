package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(c *plugin.Context, teamMember *model.TeamMember, actor *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	teamMessage, err := p.GetTeamWelcomeMessageFromKV(teamMember.TeamId)
	if err != nil {
		p.client.Log.Error("Error occurred while retrieving the welcome message", "TeamID", teamMember.TeamId, "Error", err.Error())
		return
	}

	if teamMessage == "" {
		// No dynamic welcome message for the given team, so we check if one has been set in the config.json
		for _, message := range p.getWelcomeMessages() {
			if data.User.IsGuest() && !message.IncludeGuests {
				continue
			}
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
		Message:   teamMessage,
	}
	if err := p.client.Post.CreatePost(postDM); err != nil {
		p.client.Log.Error("failed to post welcome message to the channel", "ChannelID", data.DirectMessage.Id, "Error", err.Error())
	}
}

// UserHasJoinedChannel is invoked after the membership has been committed to
// the database. If actor is not nil, the user was invited to the channel by
// the actor.
func (p *Plugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	channelInfo, err := p.client.Channel.Get(channelMember.ChannelId)
	if err != nil {
		p.client.Log.Error("Error occurred while checking the type of the chanel", "ChannelID", channelMember.ChannelId, "Error", err.Error())
		return
	} else if channelInfo.Type == model.ChannelTypePrivate {
		return
	}

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, channelMember.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.client.Log.Error("Error occurred while retrieving the welcome message", "ChannelID", channelMember.ChannelId, "Error", appErr.Message)
		return
	}

	if data == nil {
		// No welcome message for the given channel
		return
	}

	dmChannel, err := p.client.Channel.GetDirect(channelMember.UserId, p.botUserID)
	if err != nil {
		p.client.Log.Error("Error occurred while creating direct channel to the user", "UserID", channelMember.UserId, "Error", err.Error())
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
	if err := p.client.Post.CreatePost(postDM); err != nil {
		p.client.Log.Error("failed to post welcome message to the channel", "ChannelID", dmChannel.Id, "Error", err.Error())
	}

	postChannel := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelMember.ChannelId,
		Message:   string(data),
	}
	time.Sleep(1 * time.Second)
	_ = p.API.SendEphemeralPost(channelMember.UserId, postChannel)
}
