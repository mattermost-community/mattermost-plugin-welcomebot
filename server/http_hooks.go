package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// broadcastCommandPrefix is the marker that turns a direct message to the bot into a team broadcast.
// The expected format is: "!broadcast <team> <message...>", where <team> is a team name or ID and
// the message may span multiple lines.
const broadcastCommandPrefix = "!broadcast"

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
	team := p.resolveTeam(teamRef)
	if team == nil {
		http.Error(w, "could not find the supplied team", http.StatusNotFound)
		return
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

// MessageHasBeenPosted lets an admin trigger a broadcast by writing a direct message to the bot in
// the form "!broadcast <team> <message...>". The message is sent to every user of the given team,
// provided the author is a system administrator or an administrator of that team.
func (p *Plugin) MessageHasBeenPosted(_ *plugin.Context, post *model.Post) {
	// Ignore the bot's own posts to avoid loops (the broadcast itself posts to DM channels).
	if post.UserId == p.botUserID {
		return
	}

	message := strings.TrimSpace(post.Message)
	if !strings.HasPrefix(message, broadcastCommandPrefix) {
		return
	}

	// Only react to direct messages between the author and the bot.
	dmChannel, appErr := p.API.GetDirectChannel(post.UserId, p.botUserID)
	if appErr != nil {
		p.API.LogError("failed to get direct channel", "user_id", post.UserId, "error", appErr.Error())
		return
	}
	if dmChannel.Id != post.ChannelId {
		return
	}

	teamRef, broadcastMessage := parseBroadcastCommand(message)

	// The broadcast is restricted to system administrators and team administrators. Resolve the
	// target team up-front so team-admin rights can be checked against it.
	isSysadmin := p.isSystemAdmin(post.UserId)

	var team *model.Team
	if teamRef != "" {
		team = p.resolveTeam(teamRef)
	}

	authorized := isSysadmin || (team != nil && p.isTeamAdmin(post.UserId, team.Id))
	if !authorized {
		// Stay silent for non-admins so the feature is neither discoverable nor usable by them,
		// and they can't probe team existence through the responses.
		p.API.LogDebug("ignoring !broadcast from non-admin user", "user_id", post.UserId)
		return
	}

	// From here the author is an admin, so give helpful feedback on malformed input.
	if teamRef == "" || broadcastMessage == "" {
		p.notifyInitiator(post.UserId, fmt.Sprintf(
			"Не удалось разобрать команду. Формат: `%s <команда> <сообщение>`.", broadcastCommandPrefix))
		return
	}

	if team == nil {
		p.notifyInitiator(post.UserId, fmt.Sprintf("Команда `%s` не найдена.", teamRef))
		return
	}

	p.notifyInitiator(post.UserId, fmt.Sprintf(
		"Рассылка по команде **%s** запущена. По завершении придёт отчёт.", team.DisplayName))

	go p.broadcastMessageToTeam(post.UserId, team, broadcastMessage)
}

// parseBroadcastCommand extracts the team reference and message body from a broadcast command. The
// first whitespace-delimited token after the prefix is the team; everything after it (including
// newlines) is the message.
func parseBroadcastCommand(command string) (teamRef, message string) {
	rest := strings.TrimLeftFunc(strings.TrimPrefix(command, broadcastCommandPrefix), unicode.IsSpace)

	idx := strings.IndexFunc(rest, unicode.IsSpace)
	if idx == -1 {
		// Only a team was provided, no message.
		return rest, ""
	}

	teamRef = rest[:idx]
	message = strings.TrimSpace(rest[idx:])
	return teamRef, message
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

// resolveTeam resolves a team referenced either by its ID or by its name, returning nil if neither
// lookup succeeds.
func (p *Plugin) resolveTeam(teamRef string) *model.Team {
	if team, appErr := p.API.GetTeam(teamRef); appErr == nil {
		return team
	}

	team, appErr := p.API.GetTeamByName(teamRef)
	if appErr != nil {
		p.API.LogError("failed to query team", "team", teamRef, "error", appErr.Error())
		return nil
	}

	return team
}

// isSystemAdmin reports whether the user has the system administrator role.
func (p *Plugin) isSystemAdmin(userID string) bool {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.API.LogError("failed to query user", "user_id", userID, "error", appErr.Error())
		return false
	}

	return user.IsSystemAdmin()
}

// isTeamAdmin reports whether the user is an active administrator of the given team.
func (p *Plugin) isTeamAdmin(userID, teamID string) bool {
	teamMember, appErr := p.API.GetTeamMember(teamID, userID)
	if appErr != nil || teamMember == nil || teamMember.DeleteAt > 0 {
		return false
	}

	return teamMember.SchemeAdmin
}

// userCanSendMessageToTeam reports whether the user is allowed to broadcast a message to the team:
// either a system administrator, or an administrator of the given team.
func (p *Plugin) userCanSendMessageToTeam(userID, teamID string) bool {
	return p.isSystemAdmin(userID) || p.isTeamAdmin(userID, teamID)
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
