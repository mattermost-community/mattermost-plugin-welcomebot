package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) constructMessageTemplate(userID, teamID string) *MessageTemplate {
	data := &MessageTemplate{}
	var err *model.AppError

	if len(userID) > 0 {
		if data.User, err = p.API.GetUser(userID); err != nil {
			p.API.LogError("failed to query user", "user_id", userID)
			return nil
		}
	}

	if len(teamID) > 0 {
		if data.Team, err = p.API.GetTeam(teamID); err != nil {
			p.API.LogError("failed to query team", "team_id", teamID)
			return nil
		}
	}

	if data.Townsquare, err = p.API.GetChannelByName(teamID, "town-square", false); err != nil {
		p.API.LogError("failed to query town-square", "team_id", teamID)
		return nil
	}

	if data.User != nil {
		if data.DirectMessage, err = p.API.GetDirectChannel(userID, p.welcomeBotUserID); err != nil {
			p.API.LogError("failed to query direct message channel", "user_id", userID)
			return nil
		}
	}

	data.UserDisplayName = data.User.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)

	return data
}

func (p *Plugin) getSiteURL() string {
	siteURL := "http://localhost:8065"

	config := p.API.GetConfig()

	if config == nil || config.ServiceSettings.SiteURL == nil || len(*config.ServiceSettings.SiteURL) == 0 {
		return siteURL
	}

	return *config.ServiceSettings.SiteURL
}

func (p *Plugin) processWelcomeMessage(messageTemplate MessageTemplate, configMessage ConfigMessage) {
	time.Sleep(time.Second * time.Duration(configMessage.DelayInSeconds))

	siteURL := p.getSiteURL()
	if strings.Contains(siteURL, "localhost") || strings.Contains(siteURL, "127.0.0.1") {
		p.API.LogError(`Site url is set to localhost or 127.0.0.1.  For this to work properly you must also set "AllowedUntrustedInternalConnections": "127.0.0.1" in config.json`)
	}

	actionButtons := make([]*model.PostAction, 0)

	for _, configAction := range configMessage.Actions {
		if configAction.ActionType == actionTypeAutomatic {
			action := &Action{}
			action.UserID = messageTemplate.User.Id
			action.Context = &ActionContext{}
			action.Context.TeamID = messageTemplate.Team.Id
			action.Context.UserID = messageTemplate.User.Id
			action.Context.Action = "automatic"

			for _, channelName := range configAction.ChannelsAddedTo {
				p.joinChannel(action, channelName)
			}
		}

		if configAction.ActionType == actionTypeButton {
			actionButton := &model.PostAction{
				Name: configAction.ActionDisplayName,
				Integration: &model.PostActionIntegration{
					Context: map[string]interface{}{
						"action":  configAction.ActionName,
						"team_id": messageTemplate.Team.Id,
						"user_id": messageTemplate.User.Id,
					},
					URL: fmt.Sprintf("%v/plugins/%v/addchannels", siteURL, PluginId),
				},
			}

			actionButtons = append(actionButtons, actionButton)
		}
	}

	tmpMsg, _ := template.New("Response").Parse(strings.Join(configMessage.Message, "\n"))
	var message bytes.Buffer
	tmpMsg.Execute(&message, messageTemplate)

	post := &model.Post{
		Message:   message.String(),
		ChannelId: messageTemplate.DirectMessage.Id,
		UserId:    p.welcomeBotUserID,
	}

	if len(configMessage.AttachmentMessage) > 0 || len(actionButtons) > 0 {
		tmpAtch, _ := template.New("AttachmentResponse").Parse(strings.Join(configMessage.AttachmentMessage, "\n"))
		var attachMessage bytes.Buffer
		tmpAtch.Execute(&attachMessage, messageTemplate)

		sa1 := &model.SlackAttachment{
			Text: attachMessage.String(),
		}

		if len(actionButtons) > 0 {
			sa1.Actions = actionButtons
		}

		attachments := make([]*model.SlackAttachment, 0)
		attachments = append(attachments, sa1)
		post.Props = map[string]interface{}{
			"attachments": attachments,
		}
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(
			"We could not create the response post",
			"user_id", post.UserId,
			"err", err.Error(),
		)
	}
}

func (p *Plugin) processActionMessage(messageTemplate MessageTemplate, action *Action, configMessageAction ConfigMessageAction) {
	for _, channelName := range configMessageAction.ChannelsAddedTo {
		p.joinChannel(action, channelName)
	}

	tmpMsg, _ := template.New("Response").Parse(strings.Join(configMessageAction.ActionSuccessfulMessage, "\n"))
	var message bytes.Buffer
	tmpMsg.Execute(&message, messageTemplate)

	post := &model.Post{
		Message:   message.String(),
		ChannelId: messageTemplate.DirectMessage.Id,
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

func (p *Plugin) joinChannel(action *Action, channelName string) {
	if channel, err := p.API.GetChannelByName(action.Context.TeamID, channelName, false); err == nil {
		if _, err := p.API.AddChannelMember(channel.Id, action.Context.UserID); err != nil {
			p.API.LogError("Couldn't add user to the channel, continuing to nexe channel", "user_id", action.Context.UserID, "channel_id", channel.Id)
			return
		}
	} else {
		p.API.LogError("failed to get channel, continuing to the next channel", "channel_name", channelName, "user_id", action.Context.UserID)
	}
}
