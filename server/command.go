package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/mattermost/mattermost-server/v5/model"
)

const COMMAND_HELP = `* |/welcomebot preview [team-name] [user-name]| - preview the welcome message for the given team name. The current user's username will be used to render the template.
* |/welcomebot list| - list the teams for which welcome messages were defined
* |/welcomebot set_channel_welcome [welcome-message]| - set the welcome message for the given channel. Direct channels are not supported.
* |/welcomebot get_channel_welcome| - print the welcome message set for the given channel (if any)
* |/welcomebot delete_channel_welcome| - delete the welcome message for the given channel (if any)
`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "welcomebot",
		DisplayName:      "welcomebot",
		Description:      "Welcome Bot helps add new team members to channels.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: preview, help, list, set_channel_welcome, get_channel_welcome, delete_channel_welcome",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string, textArgs ...interface{}) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   fmt.Sprintf(text, textArgs...),
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) hasSysadminRole(userId string) (bool, error) {
	user, appErr := p.API.GetUser(userId)
	if appErr != nil {
		return false, appErr
	}
	if !strings.Contains(user.Roles, "system_admin") {
		return false, nil
	}
	return true, nil
}

func (p *Plugin) validateCommand(action string, parameters []string) string {
	switch action {
	case "preview":
		if len(parameters) != 1 {
			return "Please specify a team, for which preview should be made."
		}
	case "list":
		if len(parameters) > 0 {
			return "List command does not accept any extra parameters"
		}
	case "set_channel_welcome":
		if len(parameters) == 0 {
			return "`set_channel_welcome` command requires the message to be provided"
		}
	case "get_channel_welcome":
		if len(parameters) > 0 {
			return "`get_channel_welcome` command does not accept any extra parameters"
		}
	case "delete_channel_welcome":
		if len(parameters) > 0 {
			return "`delete_channel_welcome` command does not accept any extra parameters"
		}
	}

	return ""
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) {
	found := false
	for _, message := range p.getWelcomeMessages() {
		if message.TeamName == teamName {
			if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
				p.postCommandResponse(args, "error occured while processing greeting for team `%s`: `%s`", teamName, err)
				return
			}

			found = true
		}
	}

	if !found {
		p.postCommandResponse(args, "team `%s` has not been found", teamName)
	}

	return
}

func (p *Plugin) executeCommandList(args *model.CommandArgs) {
	wecomeMessages := p.getWelcomeMessages()

	if len(wecomeMessages) == 0 {
		p.postCommandResponse(args, "There are no welcome messages defined")
		return
	}

	// Deduplicate entries
	teams := make(map[string]struct{})
	for _, message := range wecomeMessages {
		teams[message.TeamName] = struct{}{}
	}

	var str strings.Builder
	str.WriteString("Teams for which welcome messages are defined:")
	for team := range teams {
		str.WriteString(fmt.Sprintf("\n * %s", team))
	}
	p.postCommandResponse(args, str.String())
	return
}

func (p *Plugin) executeCommandSetWelcome(args *model.CommandArgs) {
	channelInfo, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		p.postCommandResponse(args, "error occured while checking the type of the chanelId `%s`: `%s`", args.ChannelId, appErr)
		return
	}

	if channelInfo.Type == model.CHANNEL_PRIVATE {
		p.postCommandResponse(args, "welcome messages are not supported for direct channels")
		return
	}

	// strings.Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_channel_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := fmt.Sprintf("%s%s", WELCOMEBOT_CHANNEL_WELCOME_KEY, args.ChannelId)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		p.postCommandResponse(args, "error occured while storing the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "stored the welcome message:\n%s", message)
	return
}

func (p *Plugin) executeCommandGetWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", WELCOMEBOT_CHANNEL_WELCOME_KEY, args.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.postCommandResponse(args, "error occured while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	p.postCommandResponse(args, "Welcome message is:\n%s", string(data))
	return
}

func (p *Plugin) executeCommandDeleteWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", WELCOMEBOT_CHANNEL_WELCOME_KEY, args.ChannelId)
	data, appErr := p.API.KVGet(key)

	if appErr != nil {
		p.postCommandResponse(args, "error occured while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		p.postCommandResponse(args, "error occured while deleting the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "welcome message has been deleted")
	return
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	parameters := []string{}
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/welcomebot" {
		return &model.CommandResponse{}, nil
	}

	if response := p.validateCommand(action, parameters); response != "" {
		p.postCommandResponse(args, response)
		return &model.CommandResponse{}, nil
	}

	isSysadmin, err := p.hasSysadminRole(args.UserId)
	if err != nil {
		p.postCommandResponse(args, "authorization failed: %s", err)
		return &model.CommandResponse{}, nil
	}
	if !isSysadmin {
		p.postCommandResponse(args, "/welcomebot commands can only be executed by the user with system admin role")
		return &model.CommandResponse{}, nil
	}

	switch action {
	case "preview":
		teamName := parameters[0]
		p.executeCommandPreview(teamName, args)
		return &model.CommandResponse{}, nil
	case "list":
		p.executeCommandList(args)
		return &model.CommandResponse{}, nil
	case "set_channel_welcome":
		p.executeCommandSetWelcome(args)
		return &model.CommandResponse{}, nil
	case "get_channel_welcome":
		p.executeCommandGetWelcome(args)
		return &model.CommandResponse{}, nil
	case "delete_channel_welcome":
		p.executeCommandDeleteWelcome(args)
		return &model.CommandResponse{}, nil
	case "help":
		fallthrough
	case "":
		text := "###### Mattermost welcomebot Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, "Unknown action %v", action)
	return &model.CommandResponse{}, nil
}
