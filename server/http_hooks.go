package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

var (
	// sendMessageToTeamDelay is the pause between each direct message sent by the broadcast,
	// to avoid overwhelming the server when a team has many users. It is a var (not a const) so
	// tests can override it.
	sendMessageToTeamDelay = 50 * time.Millisecond

	// sendMessageToTeamPageSize is the page size used when paginating over the team's users.
	sendMessageToTeamPageSize = 200
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// /sendmessagetoteam has its own request shape (raw message body + team parameter),
	// so it is handled before the action-based decoding below.
	if r.URL.Path == "/sendmessagetoteam" {
		p.handleSendMessageToTeam(w, r)
		return
	}

	var action *Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil || action == nil {
		p.API.LogDebug("failed to decode action from request body", "error", err.Error())
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

// handleSendMessageToTeam sends the message provided in the request body to every user of the
// team passed as the "team" parameter. The team can be referenced either by its name or by its ID.
func (p *Plugin) handleSendMessageToTeam(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		p.API.LogError("http request not authenticated: no Mattermost-User-Id")
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	teamRef := r.URL.Query().Get("team")
	if teamRef == "" {
		http.Error(w, "missing required parameter: team", http.StatusBadRequest)
		return
	}

	// Resolve the team either by ID or by name.
	team, appErr := p.API.GetTeam(teamRef)
	if appErr != nil {
		if team, appErr = p.API.GetTeamByName(teamRef); appErr != nil {
			p.API.LogError("failed to query team", "team", teamRef, "error", appErr.Error())
			http.Error(w, "could not find the supplied team", http.StatusNotFound)
			return
		}
	}

	// Only a system administrator or an administrator of this team may broadcast a message.
	if !p.userCanSendMessageToTeam(mattermostUserID, team.Id) {
		p.API.LogError("http request not authorized to send message to team", "user_id", mattermostUserID, "team_id", team.Id)
		http.Error(w, "not authorized", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.API.LogError("failed to read request body", "error", err.Error())
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	message := string(body)
	if message == "" {
		http.Error(w, "missing message in request body", http.StatusBadRequest)
		return
	}

	// A team may contain thousands of users, so the broadcast can take many minutes. Run it in the
	// background and answer immediately so the HTTP request doesn't block (and time out). The
	// initiator is notified via DM once the broadcast finishes.
	go p.broadcastMessageToTeam(mattermostUserID, team, message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "accepted"}); err != nil {
		p.API.LogWarn("failed to write sendmessagetoteam response", "error", err.Error())
	}
}

// broadcastMessageToTeam sends message as a direct message from the bot to every user of the team.
// It is meant to be run in a background goroutine; progress and errors are reported via the log, and
// the initiator (initiatorUserID) receives a DM summary once the broadcast finishes.
func (p *Plugin) broadcastMessageToTeam(initiatorUserID string, team *model.Team, message string) {
	teamID := team.Id
	sent := 0
	failed := 0
	for page := 0; ; page++ {
		users, appErr := p.API.GetUsersInTeam(teamID, page, sendMessageToTeamPageSize)
		if appErr != nil {
			p.API.LogError("failed to query users in team", "team_id", teamID, "error", appErr.Error())
			p.notifyInitiator(initiatorUserID, fmt.Sprintf(
				"Рассылка по команде **%s** прервана из-за ошибки при получении списка пользователей. Отправлено: %d, ошибок: %d.",
				team.DisplayName, sent, failed))
			return
		}

		if len(users) == 0 {
			break
		}

		for _, user := range users {
			dmChannel, appErr := p.API.GetDirectChannel(user.Id, p.botUserID)
			if appErr != nil {
				p.API.LogError("failed to get direct channel", "user_id", user.Id, "error", appErr.Error())
				failed++
				continue
			}

			post := &model.Post{
				UserId:    p.botUserID,
				ChannelId: dmChannel.Id,
				Message:   message,
			}

			if _, appErr := p.API.CreatePost(post); appErr != nil {
				p.API.LogError("failed to post message to user", "user_id", user.Id, "error", appErr.Error())
				failed++
				continue
			}

			sent++

			// Throttle sending so that teams with many users don't overwhelm the server.
			time.Sleep(sendMessageToTeamDelay)
		}

		if len(users) < sendMessageToTeamPageSize {
			break
		}
	}

	p.API.LogInfo("finished broadcasting message to team", "team_id", teamID, "sent", sent, "failed", failed)
	p.notifyInitiator(initiatorUserID, fmt.Sprintf(
		"Рассылка по команде **%s** завершена. Отправлено: %d, ошибок: %d.",
		team.DisplayName, sent, failed))
}

// notifyInitiator sends a direct message from the bot to the given user.
func (p *Plugin) notifyInitiator(userID, message string) {
	dmChannel, appErr := p.API.GetDirectChannel(userID, p.botUserID)
	if appErr != nil {
		p.API.LogError("failed to get direct channel for initiator", "user_id", userID, "error", appErr.Error())
		return
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: dmChannel.Id,
		Message:   message,
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogError("failed to notify initiator", "user_id", userID, "error", appErr.Error())
	}
}

// userCanSendMessageToTeam reports whether the user is allowed to broadcast a message to the team:
// either a system administrator, or an administrator of the given team.
func (p *Plugin) userCanSendMessageToTeam(userID, teamID string) bool {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.API.LogError("failed to query user", "user_id", userID, "error", appErr.Error())
		return false
	}

	if user.IsSystemAdmin() {
		return true
	}

	teamMember, appErr := p.API.GetTeamMember(teamID, userID)
	if appErr != nil || teamMember == nil || teamMember.DeleteAt > 0 {
		return false
	}

	return teamMember.SchemeAdmin
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
