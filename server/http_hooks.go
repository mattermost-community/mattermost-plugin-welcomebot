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

const (
	// broadcastCommandPrefix is the marker that turns a direct message to the bot into a team
	// broadcast. The expected format is: "!broadcast <team> <message...>", where <team> is a team
	// name or ID and the message may span multiple lines.
	broadcastCommandPrefix = "!broadcast"

	// broadcastConfirmWord is the reply a user must send to confirm a pending broadcast.
	broadcastConfirmWord = "да"

	// broadcastPendingKeyPrefix namespaces the per-user pending broadcast stored in the KV store.
	broadcastPendingKeyPrefix = "broadcast_pending_"

	// broadcastPendingExpirySeconds is how long a pending broadcast waits for confirmation before it
	// expires and the initiator is notified.
	broadcastPendingExpirySeconds = 300

	// broadcastPendingKVTTLSeconds is the KV entry's time-to-live. It outlives the in-memory expiry
	// timer so the pending broadcast is still readable when the timer fires (and acts as a fallback
	// if the timer is lost, e.g. on a plugin restart).
	broadcastPendingKVTTLSeconds = broadcastPendingExpirySeconds + 30
)

// broadcastPendingExpiry is the delay after which a pending broadcast expires. It is a var (not a
// const) so tests can shorten it.
var broadcastPendingExpiry = broadcastPendingExpirySeconds * time.Second

// broadcastCancelWords are the replies that cancel a pending broadcast.
var broadcastCancelWords = []string{"нет", "отмена"}

// isBroadcastCancelWord reports whether message is one of the cancel words (case-insensitive).
func isBroadcastCancelWord(message string) bool {
	for _, word := range broadcastCancelWords {
		if strings.EqualFold(message, word) {
			return true
		}
	}
	return false
}

// pendingBroadcast holds a broadcast awaiting the initiator's confirmation.
type pendingBroadcast struct {
	TeamID          string `json:"team_id"`
	TeamDisplayName string `json:"team_display_name"`
	Message         string `json:"message"`
	// Token identifies this particular pending broadcast so a stale expiry timer never discards a
	// newer one the same user has since started.
	Token string `json:"token"`
}

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
	isBroadcast := strings.HasPrefix(message, broadcastCommandPrefix)
	isConfirm := strings.EqualFold(message, broadcastConfirmWord)
	isCancel := isBroadcastCancelWord(message)
	if !isBroadcast && !isConfirm && !isCancel {
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

	switch {
	case isBroadcast:
		p.handleBroadcastRequest(post.UserId, message)
	case isConfirm:
		p.handleBroadcastConfirmation(post.UserId)
	case isCancel:
		p.handleBroadcastCancellation(post.UserId)
	}
}

// handleBroadcastCancellation discards the user's pending broadcast (if any) and confirms it.
func (p *Plugin) handleBroadcastCancellation(userID string) {
	pending, ok := p.takePendingBroadcast(userID)
	if !ok {
		// No pending broadcast: the cancel word was an ordinary message, so stay silent.
		return
	}

	p.notifyInitiator(userID, fmt.Sprintf(
		"Рассылка по команде **%s** отменена.", pending.TeamDisplayName))
}

// handleBroadcastRequest validates a "!broadcast" command and, if the author is authorized, stores
// it as a pending broadcast and asks the author to confirm before anything is sent.
func (p *Plugin) handleBroadcastRequest(userID, message string) {
	teamRef, broadcastMessage := parseBroadcastCommand(message)

	// The broadcast is restricted to system administrators and team administrators. Resolve the
	// target team up-front so team-admin rights can be checked against it.
	isSysadmin := p.isSystemAdmin(userID)

	var team *model.Team
	if teamRef != "" {
		team = p.resolveTeam(teamRef)
	}

	authorized := isSysadmin || (team != nil && p.isTeamAdmin(userID, team.Id))
	if !authorized {
		// Stay silent for non-admins so the feature is neither discoverable nor usable by them,
		// and they can't probe team existence through the responses.
		p.API.LogDebug("ignoring !broadcast from non-admin user", "user_id", userID)
		return
	}

	// From here the author is an admin, so give helpful feedback on malformed input.
	if teamRef == "" || broadcastMessage == "" {
		p.notifyInitiator(userID, fmt.Sprintf(
			"Не удалось разобрать команду. Формат: `%s <команда> <сообщение>`.", broadcastCommandPrefix))
		return
	}

	if team == nil {
		p.notifyInitiator(userID, fmt.Sprintf("Команда `%s` не найдена.", teamRef))
		return
	}

	memberCount := p.teamMemberCount(team.Id)

	pending := pendingBroadcast{
		TeamID:          team.Id,
		TeamDisplayName: team.DisplayName,
		Message:         broadcastMessage,
		Token:           model.NewId(),
	}
	if !p.storePendingBroadcast(userID, pending) {
		p.notifyInitiator(userID, "Не удалось подготовить рассылку. Попробуйте ещё раз.")
		return
	}

	p.scheduleBroadcastExpiry(userID, pending.Token)

	p.notifyInitiator(userID, fmt.Sprintf(
		"⚠️ Вы собираетесь отправить сообщение **%d** участникам команды **%s**.\n"+
			"Отправьте \"%s\" для продолжения или \"%s\" для отмены.",
		memberCount, team.DisplayName, broadcastConfirmWord, broadcastCancelWords[0]))
}

// scheduleBroadcastExpiry notifies the initiator (and drops the pending broadcast) if it is not
// confirmed or cancelled within broadcastPendingExpiry.
func (p *Plugin) scheduleBroadcastExpiry(userID, token string) {
	time.AfterFunc(broadcastPendingExpiry, func() {
		p.expirePendingBroadcast(userID, token)
	})
}

// expirePendingBroadcast discards a still-pending broadcast whose confirmation timed out and tells
// the initiator. It does nothing if the broadcast was already handled or replaced by a newer one.
func (p *Plugin) expirePendingBroadcast(userID, token string) {
	pending, ok := p.peekPendingBroadcast(userID)
	if !ok || pending.Token != token {
		return
	}

	p.deletePendingBroadcast(userID)
	p.notifyInitiator(userID, fmt.Sprintf(
		"Время ожидания подтверждения истекло. Рассылка по команде **%s** отменена.",
		pending.TeamDisplayName))
}

// handleBroadcastConfirmation runs the pending broadcast (if any) for the user who confirmed it.
func (p *Plugin) handleBroadcastConfirmation(userID string) {
	pending, ok := p.takePendingBroadcast(userID)
	if !ok {
		// No pending broadcast: the "да" was an ordinary message, so stay silent.
		return
	}

	// Re-check authorization in case the user's roles changed between request and confirmation.
	if !p.userCanSendMessageToTeam(userID, pending.TeamID) {
		p.API.LogError("user no longer authorized to broadcast to team", "user_id", userID, "team_id", pending.TeamID)
		p.notifyInitiator(userID, fmt.Sprintf(
			"У вас больше нет прав на рассылку по команде **%s**.", pending.TeamDisplayName))
		return
	}

	team := &model.Team{Id: pending.TeamID, DisplayName: pending.TeamDisplayName}
	p.notifyInitiator(userID, fmt.Sprintf(
		"Рассылка по команде **%s** запущена. По завершении придёт отчёт.", pending.TeamDisplayName))

	go p.broadcastMessageToTeam(userID, team, pending.Message)
}

// teamMemberCount returns the number of active members in the team, or 0 if it cannot be determined.
func (p *Plugin) teamMemberCount(teamID string) int64 {
	stats, appErr := p.API.GetTeamStats(teamID)
	if appErr != nil {
		p.API.LogError("failed to query team stats", "team_id", teamID, "error", appErr.Error())
		return 0
	}

	return stats.ActiveMemberCount
}

// storePendingBroadcast persists a pending broadcast for the user with an expiry, returning false on
// failure.
func (p *Plugin) storePendingBroadcast(userID string, pending pendingBroadcast) bool {
	data, err := json.Marshal(pending)
	if err != nil {
		p.API.LogError("failed to marshal pending broadcast", "user_id", userID, "error", err.Error())
		return false
	}

	if appErr := p.API.KVSetWithExpiry(broadcastPendingKeyPrefix+userID, data, broadcastPendingKVTTLSeconds); appErr != nil {
		p.API.LogError("failed to store pending broadcast", "user_id", userID, "error", appErr.Error())
		return false
	}

	return true
}

// peekPendingBroadcast loads the user's pending broadcast without removing it, returning ok=false if
// there is none.
func (p *Plugin) peekPendingBroadcast(userID string) (pendingBroadcast, bool) {
	var pending pendingBroadcast

	data, appErr := p.API.KVGet(broadcastPendingKeyPrefix + userID)
	if appErr != nil {
		p.API.LogError("failed to load pending broadcast", "user_id", userID, "error", appErr.Error())
		return pending, false
	}
	if data == nil {
		return pending, false
	}

	if err := json.Unmarshal(data, &pending); err != nil {
		p.API.LogError("failed to unmarshal pending broadcast", "user_id", userID, "error", err.Error())
		return pending, false
	}

	return pending, true
}

// deletePendingBroadcast removes the user's pending broadcast.
func (p *Plugin) deletePendingBroadcast(userID string) {
	if appErr := p.API.KVDelete(broadcastPendingKeyPrefix + userID); appErr != nil {
		p.API.LogError("failed to delete pending broadcast", "user_id", userID, "error", appErr.Error())
	}
}

// takePendingBroadcast loads and deletes the user's pending broadcast, returning ok=false if there
// is none.
func (p *Plugin) takePendingBroadcast(userID string) (pendingBroadcast, bool) {
	var pending pendingBroadcast

	key := broadcastPendingKeyPrefix + userID
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.API.LogError("failed to load pending broadcast", "user_id", userID, "error", appErr.Error())
		return pending, false
	}
	if data == nil {
		return pending, false
	}

	// Consume the pending broadcast so a confirmation can't be replayed.
	if appErr := p.API.KVDelete(key); appErr != nil {
		p.API.LogError("failed to delete pending broadcast", "user_id", userID, "error", appErr.Error())
	}

	if err := json.Unmarshal(data, &pending); err != nil {
		p.API.LogError("failed to unmarshal pending broadcast", "user_id", userID, "error", err.Error())
		return pending, false
	}

	return pending, true
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
	skipped := 0
	failed := 0
	for page := 0; ; page++ {
		// GetUsersInTeam scopes to the team (unlike GetUsers, whose InTeamId option is ignored and
		// which would return every user on the server). Bots and deactivated accounts are excluded
		// client-side below.
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
			// Never message bots (including the welcomebot itself) or deactivated accounts.
			if user.IsBot || user.DeleteAt != 0 {
				skipped++
				continue
			}

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

	p.API.LogInfo("finished broadcasting message to team", "team_id", teamID, "sent", sent, "skipped", skipped, "failed", failed)
	p.notifyInitiator(initiatorUserID, fmt.Sprintf(
		"Рассылка по команде **%s** завершена. Отправлено: %d, пропущено (боты/неактивные): %d, ошибок: %d.",
		team.DisplayName, sent, skipped, failed))
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
