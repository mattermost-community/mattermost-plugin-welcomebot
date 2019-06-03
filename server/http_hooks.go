package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	var action *Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil || action == nil {
		p.encodeEphermalMessage(w, "WelcomeBot Error: We could not decode the action")
		return
	}

	data := &MessageTemplate{}
	var err *model.AppError

	if data.User, err = p.API.GetUser(action.Context.UserID); err != nil {
		p.API.LogError("failed to query user", "user_id", action.Context.UserID)
		p.encodeEphermalMessage(w, "WelcomeBot Error: We could not find the supplied user")
		return
	}

	if data.Team, err = p.API.GetTeam(action.Context.TeamID); err != nil {
		p.API.LogError("failed to query team", "team_id", action.Context.TeamID)
		p.encodeEphermalMessage(w, "WelcomeBot Error: We could not find the supplied team")
		return
	}

	if data.DirectMessage, err = p.API.GetDirectChannel(action.Context.UserID, p.botUserID); err != nil {
		p.API.LogError("failed to query direct message channel", "user_id", action.Context.UserID)
		p.encodeEphermalMessage(w, "WelcomeBot Error: We could not find the welcome bot direct message channel")
		return
	}

	data.UserDisplayName = data.User.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)

	// Check to make sure you're still in the team
	if teamMember, err := p.API.GetTeamMember(action.Context.TeamID, action.Context.UserID); err != nil || teamMember == nil || teamMember.DeleteAt > 0 {
		p.API.LogError("Didn't have access to team", "user_id", action.Context.UserID, "team_id", action.Context.TeamID)
		p.encodeEphermalMessage(w, "WelcomeBot Error: You do not appear to have access to this team")
		return
	}

	switch r.URL.Path {
	case "/addchannels":
		for _, wm := range p.getWelcomeMessages() {
			if data.Team.Name == wm.TeamName {
				for _, ac := range wm.Actions {
					if ac.ActionName == action.Context.Action {
						p.processActionMessage(*data, action, *ac)
						p.encodeEphermalMessage(w, "")
						return
					}
				}
			}
		}

		p.encodeEphermalMessage(w, "WelcomeBot Error: The action wasn't found for "+action.Context.Action)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) encodeEphermalMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")

	resp := model.PostActionIntegrationResponse{
		EphemeralText: message,
	}

	if _, err := w.Write([]byte(resp.ToJson())); err != nil {
		p.API.LogWarn("failed to write PostActionIntegrationResponse")
	}
}
