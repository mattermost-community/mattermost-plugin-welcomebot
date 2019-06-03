package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	var action *Action
	json.NewDecoder(r.Body).Decode(&action)

	if action == nil {
		encodeEphermalMessage(w, "WelcomeBot Error: We could not decode the action")
		return
	}

	data := &MessageTemplate{}
	var err *model.AppError

	if data.User, err = p.API.GetUser(action.Context.UserID); err != nil {
		p.API.LogError("failed to query user", "user_id", action.Context.UserID)
		encodeEphermalMessage(w, "WelcomeBot Error: We could not find the supplied user")
		return
	}

	if data.Team, err = p.API.GetTeam(action.Context.TeamID); err != nil {
		p.API.LogError("failed to query team", "team_id", action.Context.TeamID)
		encodeEphermalMessage(w, "WelcomeBot Error: We could not find the supplied team")
		return
	}

	if data.DirectMessage, err = p.API.GetDirectChannel(action.Context.UserID, p.welcomeBotUserID); err != nil {
		p.API.LogError("failed to query direct message channel", "user_id", action.Context.UserID)
		encodeEphermalMessage(w, "WelcomeBot Error: We could not find the welcome bot direct message channel")
		return
	}

	data.UserDisplayName = data.User.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)

	// Check to make sure you're still in the team
	if teamMember, err := p.API.GetTeamMember(action.Context.TeamID, action.Context.UserID); err != nil || teamMember == nil || teamMember.DeleteAt > 0 {
		p.API.LogError("Didn't have access to team", "user_id", action.Context.UserID, "team_id", action.Context.TeamID)
		encodeEphermalMessage(w, "WelcomeBot Error: You do not appear to have access to this team")
		return
	}

	switch r.URL.Path {
	case "/addchannels":

		for _, wm := range p.getWelcomeMessages() {

			if data.Team.Name == wm.TeamName {
				for _, ac := range wm.Actions {
					if ac.ActionName == action.Context.Action {
						p.processActionMessage(*data, action, *ac)
						encodeEphermalMessage(w, "")
						return
					}
				}
			}
		}

		encodeEphermalMessage(w, "WelcomeBot Error: The action wasn't found for "+action.Context.Action)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleSupport(action *Action, data *MessageTemplate) {
	p.joinChannel(action, "peer-to-peer-help")
	p.joinChannel(action, "bugs")

	template, _ := template.New("Response").Parse(
		`### Awesome, I have added you to the following support related channels!
~peer-to-peer-help - General help and setup
~bugs - To help investigate or report bugs
`)
	var message bytes.Buffer
	template.Execute(&message, data)

	post := &model.Post{
		Message:   message.String(),
		ChannelId: data.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(
			"We could not create the response post",
			"user_id", post.UserId,
			"err", err.Error(),
		)
	}
}

func (p *Plugin) handelDeveloper(action *Action, data *MessageTemplate) {
	p.joinChannel(action, "developers")
	p.joinChannel(action, "developer-toolkit")
	p.joinChannel(action, "developer-performance")
	p.joinChannel(action, "bugs")

	template, _ := template.New("Response").Parse(
		`### Baller, I have added you to the following developer related channels!
~developers - Great for general developer questions
~developer-toolkit - Great questions about plugins or integrations
~developer-meeting - Weekly core staff and community meeting
~developer-performance - Great for questions about performance or load testing
~bugs - To help investigate or report bugs
`)
	var message bytes.Buffer
	template.Execute(&message, data)

	post := &model.Post{
		Message:   message.String(),
		ChannelId: data.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(
			"We could not create the response post",
			"user_id", post.UserId,
			"err", err.Error(),
		)
	}
}

func (p *Plugin) handleDoNotKnow(action *Action, data *MessageTemplate) {
	p.joinChannel(action, "peer-to-peer-help")
	p.joinChannel(action, "feature-ideas")
	p.joinChannel(action, "developers")
	p.joinChannel(action, "bugs")

	template, _ := template.New("Response").Parse(
		`### Great, I have added you to a few channels that might be interesting!
~peer-to-peer-help - General help and setup
~feature-ideas - To discuss potential feature ideas
~developers - Great for general developer questions
~bugs - To help investigate or report bugs
`)
	var message bytes.Buffer
	template.Execute(&message, data)

	post := &model.Post{
		Message:   message.String(),
		ChannelId: data.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(
			"We could not create the response post",
			"user_id", post.UserId,
			"err", err.Error(),
		)
	}
}

func encodeEphermalMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]interface{}{
		"ephemeral_text": message,
	}

	json.NewEncoder(w).Encode(payload)
}
