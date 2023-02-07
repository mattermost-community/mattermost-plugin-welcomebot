package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	var action *Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil || action == nil {
		p.API.LogError("failed to decode action from request body", "error", err)
		p.encodeEphemeralMessage(w, "WelcomeBot Error: We could not decode the action")
		return
	}

	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" || mattermostUserID != action.Context.UserID {
		p.API.LogError("http request not authenticated: no Mattermost-User-Id")
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	data := &MessageTemplate{}
	var err *model.AppError

	if data.User, err = p.API.GetUser(action.Context.UserID); err != nil {
		p.API.LogError("failed to query user", "user_id", action.Context.UserID, "error", err.Error())
		p.encodeEphemeralMessage(w, "WelcomeBot Error: We could not find the supplied user")
		return
	}

	if data.Team, err = p.API.GetTeam(action.Context.TeamID); err != nil {
		p.API.LogError("failed to query team", "team_id", action.Context.TeamID, "error", err.Error())
		p.encodeEphemeralMessage(w, "WelcomeBot Error: We could not find the supplied team")
		return
	}

	if data.DirectMessage, err = p.API.GetDirectChannel(action.Context.UserID, p.botUserID); err != nil {
		p.API.LogError("failed to query direct message channel", "user_id", action.Context.UserID, "error", err.Error())
		p.encodeEphemeralMessage(w, "WelcomeBot Error: We could not find the welcome bot direct message channel")
		return
	}

	data.UserDisplayName = data.User.GetDisplayName(model.ShowNicknameFullName)

	// Check to make sure you're still in the team
	if teamMember, err := p.API.GetTeamMember(action.Context.TeamID, action.Context.UserID); err != nil || teamMember == nil || teamMember.DeleteAt > 0 {
		p.API.LogError("Didn't have access to team", "user_id", action.Context.UserID, "team_id", action.Context.TeamID, "error", err.Error())
		p.encodeEphemeralMessage(w, "WelcomeBot Error: You do not appear to have access to this team")
		return
	}

	switch r.URL.Path {
	case "/addchannels":
		for _, wm := range p.getWelcomeMessages() {
			if data.Team.Name == wm.TeamName {
				for _, ac := range wm.Actions {
					if ac.ActionName == action.Context.Action {
						p.processActionMessage(*data, action, *ac)
						p.encodeEphemeralMessage(w, "")
						return
					}
				}
			}
		}

		p.encodeEphemeralMessage(w, "WelcomeBot Error: The action wasn't found for "+action.Context.Action)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) encodeEphemeralMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")

	resp := model.PostActionIntegrationResponse{
		EphemeralText: message,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		p.API.LogWarn("failed to write PostActionIntegrationResponse")
	}
}
