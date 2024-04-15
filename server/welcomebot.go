package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
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
		if data.DirectMessage, err = p.API.GetDirectChannel(userID, p.botUserID); err != nil {
			p.API.LogError("failed to query direct message channel", "user_id", userID)
			return nil
		}
	}

	data.UserDisplayName = data.User.GetDisplayName(model.ShowNicknameFullName)

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

func (p *Plugin) newSampleMessageTemplate(teamName string, userID string) (*MessageTemplate, error) {
	data := &MessageTemplate{}
	var err *model.AppError

	if data.User, err = p.API.GetUser(userID); err != nil {
		p.API.LogError("failed to query user", "user_id", userID, "err", err)
		return nil, fmt.Errorf("failed to query user %s: %w", userID, err)
	}

	if data.Team, err = p.API.GetTeamByName(strings.ToLower(teamName)); err != nil {
		p.API.LogError("failed to query team", "team_name", teamName, "err", err)
		return nil, fmt.Errorf("failed to query team %s: %w", teamName, err)
	}

	if data.Townsquare, err = p.API.GetChannelByName(data.Team.Id, "town-square", false); err != nil {
		p.API.LogError("failed to query town-square", "team_name", data.Team.Name)
		return nil, fmt.Errorf("failed to query town-square %s: %w", data.Team.Name, err)
	}

	if data.DirectMessage, err = p.API.GetDirectChannel(data.User.Id, p.botUserID); err != nil {
		p.API.LogError("failed to query direct message channel", "user_name", data.User.Username)
		return nil, fmt.Errorf("failed to query direct message channel %s: %w", data.User.Id, err)
	}

	data.UserDisplayName = data.User.GetDisplayName(model.ShowNicknameFullName)

	return data, nil
}

func (p *Plugin) previewWelcomeMessage(teamName string, args *model.CommandArgs, configMessage ConfigMessage) error {
	messageTemplate, err := p.newSampleMessageTemplate(teamName, args.UserId)
	if err != nil {
		return err
	}

	post := p.renderWelcomeMessage(*messageTemplate, configMessage)
	post.ChannelId = args.ChannelId
	_ = p.API.SendEphemeralPost(args.UserId, post)

	return nil
}

func (p *Plugin) renderWelcomeMessage(messageTemplate MessageTemplate, configMessage ConfigMessage) *model.Post {
	actionButtons := make([]*model.PostAction, 0)

	for _, configAction := range configMessage.Actions {
		if configAction.ActionType == actionTypeAutomatic {
			action := &Action{}
			action.UserID = messageTemplate.User.Id
			action.Context = &ActionContext{}
			action.Context.TeamID = messageTemplate.Team.Id
			action.Context.UserID = messageTemplate.User.Id
			action.Context.DirectMessagePost = configAction.ActionDirectMessagePost
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
						"action":              configAction.ActionName,
						"team_id":             messageTemplate.Team.Id,
						"user_id":             messageTemplate.User.Id,
						"direct_message_post": configAction.ActionDirectMessagePost,
					},
					URL: fmt.Sprintf("%v/plugins/%v/addchannels", p.getSiteURL(), manifest.ID),
				},
			}

			actionButtons = append(actionButtons, actionButton)
		}
	}

	tmpMsg, _ := template.New("Response").Parse(strings.Join(configMessage.Message, "\n"))
	var message bytes.Buffer
	err := tmpMsg.Execute(&message, messageTemplate)
	if err != nil {
		p.API.LogError(
			"Failed to execute message template",
			"err", err.Error(),
		)
	}

	post := &model.Post{
		Message: message.String(),
		UserId:  p.botUserID,
	}

	if len(configMessage.AttachmentMessage) > 0 || len(actionButtons) > 0 {
		tmpAtch, _ := template.New("AttachmentResponse").Parse(strings.Join(configMessage.AttachmentMessage, "\n"))
		var attachMessage bytes.Buffer
		err := tmpAtch.Execute(&attachMessage, messageTemplate)
		if err != nil {
			p.API.LogError(
				"Failed to execute message template",
				"err", err.Error(),
			)
		}

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

	return post
}

func (p *Plugin) processWelcomeMessage(messageTemplate MessageTemplate, configMessage ConfigMessage) {
	time.Sleep(time.Second * time.Duration(configMessage.DelayInSeconds))

	siteURL := p.getSiteURL()
	if strings.Contains(siteURL, "localhost") || strings.Contains(siteURL, "127.0.0.1") {
		p.API.LogWarn(`Site url is set to localhost or 127.0.0.1.  For this to work properly you must also set "AllowedUntrustedInternalConnections": "127.0.0.1" in config.json`)
	}

	post := p.renderWelcomeMessage(messageTemplate, configMessage)
	post.ChannelId = messageTemplate.DirectMessage.Id

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
	err := tmpMsg.Execute(&message, messageTemplate)
	if err != nil {
		p.API.LogError(
			"Failed to execute message template",
			"err", err.Error(),
		)
	}

	post := &model.Post{
		Message:   message.String(),
		ChannelId: messageTemplate.DirectMessage.Id,
		UserId:    p.botUserID,
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
	// If it begins with @ create a DM channel
	if strings.HasPrefix(channelName, "@") {
		if err := p.handleDMs(action,channelName); err != nil {
			p.API.LogError("failed to handle DM channel, continuing to next channel. " + err.Error())
		}

	} else { // Otherwise treat it like a normal channel
		if channel, err := p.API.GetChannelByName(action.Context.TeamID, channelName, false); err == nil {
			if _, err := p.API.AddChannelMember(channel.Id, action.Context.UserID); err != nil {
				p.API.LogError("Couldn't add user to the channel, continuing to next channel", "channel_name", channelName, "user_id", action.Context.UserID, channel.Id)
				return
			}
		} else {
			p.API.LogError("failed to get channel, continuing to the next channel", "channel_name", channelName, "user_id", action.Context.UserID)
			return
		}
	}
}

func (p *Plugin) handleDMs(action *Action, channelName string) error {
	username := channelName[1:]
	dmUser, userErr := p.API.GetUserByUsername(username)
	if userErr != nil {
		return errors.Wrapf(userErr, "couldn't find DM channel for username %s", username)
	}

	if !dmUser.IsBot {
		return errors.Wrapf(userErr, "Specified DM user is not a bot for username %s", username)
	}

	dmChannel, err := p.API.GetDirectChannel(dmUser.Id, action.Context.UserID)
	if err != nil {
		return errors.Wrapf(err, "Couldn't create or get DM channel for user_id %s and channel_id %s" , action.Context.UserID, dmChannel.Id)
	}

	dmMessage := "Welcome to the team!"
	if len(action.Context.DirectMessagePost) != 0 {
		dmMessage = action.Context.DirectMessagePost
	}

	post := &model.Post{
		Message:   dmMessage,
		ChannelId: dmChannel.Id,
		UserId:    dmUser.Id,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		return errors.Wrapf(err, "Could not create direct message for user_id %s", post.UserId)
	}

	return nil
}
