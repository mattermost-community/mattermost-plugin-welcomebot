package core

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(_ *plugin.Context, teamMember *model.TeamMember, _ *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	for _, message := range p.GetWelcomeMessages() {
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
func (p *Plugin) UserHasJoinedChannel(_ *plugin.Context, channelMember *model.ChannelMember, _ *model.User) {
	for _, hook := range p.userHasJoinedChannelHooks {
		hook.Execute(p, channelMember)
	}
}
