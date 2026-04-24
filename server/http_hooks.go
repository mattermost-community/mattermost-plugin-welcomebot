package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// Route admin endpoints before decoding the action body — admin requests
	// use a different payload format than interactive button actions.
	if r.URL.Path == "/admin/set_channel_welcome" {
		p.handleAdminSetChannelWelcome(w, r)
		return
	}

	var action *Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil || action == nil {
		// err may be nil when the body decoded successfully but action is nil
		// (e.g. the payload was JSON null) — guard before calling err.Error().
		errMsg := "nil action"
		if err != nil {
			errMsg = err.Error()
		}
		p.API.LogDebug("failed to decode action from request body", "error", errMsg)
		p.encodeEphemeralMessage(w, "WelcomeBot Error: We could not decode the action")
		return
	}

	// Guard against a well-formed JSON body that decodes to a non-nil Action
	// but has a nil Context (e.g. {} or {"context":null}). Dereferencing
	// action.Context.UserID without this check would panic the plugin.
	if action.Context == nil || action.Context.UserID == "" || action.Context.TeamID == "" {
		p.API.LogDebug("action decoded but context is missing or incomplete")
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
	data.SiteURL = p.getSiteURL()

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

// handleAdminSetChannelWelcome handles POST /admin/set_channel_welcome.
//
// Accepts {"channel_id": "...", "message": "..."} and writes the message into
// the plugin KV store using the same key as /welcomebot set_channel_welcome.
// The stored message is sent as an ephemeral post when a user joins the channel.
//
// This endpoint exists so setup scripts can apply channel welcome messages
// from config without requiring a human to run /welcomebot set_channel_welcome
// in every channel individually. Caller must be a system admin.
func (p *Plugin) handleAdminSetChannelWelcome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	isSysadmin, err := p.hasSysadminRole(userID)
	if err != nil {
		p.API.LogError("failed to check sysadmin role", "user_id", userID, "error", err.Error())
		http.Error(w, "authorization check failed", http.StatusInternalServerError)
		return
	}
	if !isSysadmin {
		http.Error(w, "forbidden — system admin required", http.StatusForbidden)
		return
	}

	var req struct {
		ChannelID string `json:"channel_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}
	if req.ChannelID == "" {
		http.Error(w, "channel_id is required", http.StatusBadRequest)
		return
	}

	// Only open channels support join-based welcomes. Reject private, direct,
	// and group channels early so setup scripts get a clear error rather than
	// silently storing a message that will never be delivered.
	channelInfo, appErr := p.API.GetChannel(req.ChannelID)
	if appErr != nil {
		p.API.LogError("failed to look up channel", "channel_id", req.ChannelID, "error", appErr.Error())
		http.Error(w, "channel not found", http.StatusBadRequest)
		return
	}
	if channelInfo.Type == model.ChannelTypePrivate ||
		channelInfo.Type == model.ChannelTypeDirect ||
		channelInfo.Type == model.ChannelTypeGroup {
		http.Error(w, "welcome messages are only supported for open channels", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, req.ChannelID)
	if appErr := p.API.KVSet(key, []byte(req.Message)); appErr != nil {
		p.API.LogError("failed to store channel welcome message",
			"channel_id", req.ChannelID,
			"error", appErr.Error(),
		)
		http.Error(w, "failed to store welcome message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","channel_id":%q}`, req.ChannelID)
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
