package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(_ *plugin.Context, teamMember *model.TeamMember, _ *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	for _, message := range p.getWelcomeMessages() {
		if data.User.IsGuest() && !message.IncludeGuests {
			continue
		}

		if message.TeamName == data.Team.Name {
			go p.processWelcomeMessage(*data, *message)
		}
	}
}

// UserHasJoinedChannel is invoked after the membership has been committed to
// the database. If actor is not nil, the user was invited to the channel by
// the actor.
func (p *Plugin) UserHasJoinedChannel(_ *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	if channelInfo, appErr := p.API.GetChannel(channelMember.ChannelId); appErr != nil {
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

	// actor == nil means the join was triggered by a plugin API call (AddChannelMember).
	// In that case joinChannel handles the ephemeral directly — skip here to avoid
	// double delivery. Only send from this hook when a real user triggered the join
	// (self-join: actor == user, or admin-add: actor is a different user).
	if actor == nil {
		return
	}

	// Send the welcome as a best-effort ephemeral post after a short delay.
	// Ephemerals are delivered to the active client session and are not persisted
	// server-side, so users can still miss them depending on client state/timing.
	// The delay gives the client a better chance to render the channel before the
	// post is sent.
	delay := p.getChannelWelcomeAutoJoinDelay()
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelMember.ChannelId,
		Message:   string(data),
	}
	go func() {
		time.Sleep(delay)
		_ = p.API.SendEphemeralPost(channelMember.UserId, post)
	}()
}
