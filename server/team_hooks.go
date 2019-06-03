package main

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// UserHasJoinedTeam is invoked after the membership has been committed to the database. If
// actor is not nil, the user was added to the team by the actor.
func (p *Plugin) UserHasJoinedTeam(c *plugin.Context, teamMember *model.TeamMember, actor *model.User) {
	data := p.constructMessageTemplate(teamMember.UserId, teamMember.TeamId)
	if data == nil {
		return
	}

	for _, message := range p.getWelcomeMessages() {
		if message.TeamName == data.Team.Name {
			go p.processWelcomeMessage(*data, *message)
		}
	}
}

func (p *Plugin) contributorsWelcome(c *plugin.Context, data *MessageTemplate) {
	// We wait for a bit so it's obvious in the UI that you've been mentioned
	time.Sleep(3 * time.Second)

	welcomeTemplate, _ := template.New("Welcome").Parse(
		`# Welcome {{.UserDisplayName}} to the Mattermost {{.Team.DisplayName}} nightly build server!

### Please review our Code of Conduct 

Thank you for visiting the virtual office of the Mattermost core team, and for your interest in the [Mattermost open source project](http://www.mattermost.org/vision/#mattermost-teams-v1). 

This is the virtual office of for the Mattermost core team, project contributors, testers and invited guests. Please behave here as you would as a guest in an office space where people are working. 

Important notes:

##### 1) This is a virtual office, not a demo server

You can use [Mattermost install guides](http://www.mattermost.org/installation/) to install your own version of Matterrmost if you want to experiment with features. 

##### 2) This is a virtual office, not the Mattermost public forum (the [forum](https://forum.mattermost.org/) is here)

Please do not create public channels or private groups. If you want to discuss something that’s not a topic started by the core team, please start a topic on the [public forum](https://forum.mattermost.org/), or discuss in the [IRC Public Discussion channel](https://pre-release.mattermost.com/core/channels/irc#). 

If people are responding to your @ mentions, it’s fine to continue a conversation. If your mentions have been repeatedly been ignored, please stop. 

Please move conversations to the appropriate channel by topic and do not post long threads in the Reception area. Moderators delete or move your messages if the topic isn't relevant to 300+ members viewing the Reception area. 

##### 3) This is a virtual office, not a substitute for community systems

There are standard community systems for [bugs, feature ideas, and troubleshooting help](http://docs.mattermost.com/process/community-systems.html). 

If you receive a request from the core team, or matterbot, to change your behavior in the office, please follow the request or leave the office. 

In addition to the basics discussed above the [Contributor Code of Conduct](http://contributor-covenant.org/version/1/3/0/code_of_conduct.txt) also applies.

Updated March 5, 2016
		`)
	var welcomeMessage bytes.Buffer
	welcomeTemplate.Execute(&welcomeMessage, data)

	welcomePost := &model.Post{
		Message:   welcomeMessage.String(),
		ChannelId: data.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
	}

	if _, err := p.API.CreatePost(welcomePost); err != nil {
		p.API.LogError(
			"We could not create the welcome post",
			"user_id", welcomePost.UserId,
			"err", err.Error(),
		)
	} else {
		p.API.LogDebug(
			"Posted welcome message",
			"username", data.User.Username,
			"team", data.Team.Name,
		)
	}
}

func (p *Plugin) contributorsHelpWithChannels(c *plugin.Context, data *MessageTemplate) {
	time.Sleep(1 * time.Second)

	//siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	siteURL := "http://localhost:8065"
	welcomeTemplate, _ := template.New("Welcome").Parse("# Need help with Channels?")
	var welcomeMessage bytes.Buffer
	welcomeTemplate.Execute(&welcomeMessage, data)

	action1 := &model.PostAction{
		Name: "I'm interested in Support",
		Integration: &model.PostActionIntegration{
			Context: map[string]interface{}{
				"action":  "support",
				"team_id": data.Team.Id,
				"user_id": data.User.Id,
			},
			URL: fmt.Sprintf("%v/plugins/%v/addchannels", siteURL, PluginId),
		},
	}

	action2 := &model.PostAction{
		Name: "Developing on Mattermost",
		Integration: &model.PostActionIntegration{
			Context: map[string]interface{}{
				"action":  "developer",
				"team_id": data.Team.Id,
				"user_id": data.User.Id,
			},
			URL: fmt.Sprintf("%v/plugins/%v/addchannels", siteURL, PluginId),
		},
	}

	action3 := &model.PostAction{
		Name: "I don't Know?",
		Integration: &model.PostActionIntegration{
			Context: map[string]interface{}{
				"action":  "do-not-know",
				"team_id": data.Team.Id,
				"user_id": data.User.Id,
			},
			URL: fmt.Sprintf("%v/plugins/%v/addchannels", siteURL, PluginId),
		},
	}

	sa1 := &model.SlackAttachment{
		Text: "I can help you get started by joining you to a bunch of existing channels! Which types of channels would you like to join?",
		Actions: []*model.PostAction{
			action1,
			action2,
			action3,
		},
	}
	attachments := make([]*model.SlackAttachment, 0)
	attachments = append(attachments, sa1)

	welcomePost := &model.Post{
		Message:   welcomeMessage.String(),
		ChannelId: data.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}

	if _, err := p.API.CreatePost(welcomePost); err != nil {
		p.API.LogError(
			"We could not create the welcome post",
			"user_id", welcomePost.UserId,
			"err", err.Error(),
		)
	} else {
		p.API.LogDebug(
			"Posted welcome message",
			"username", data.User.Username,
			"team", data.Team.Name,
		)
	}
}
