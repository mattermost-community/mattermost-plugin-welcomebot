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

func (p *Plugin) responsef(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Username:     p.botUserID,
		Text:         fmt.Sprintf(format, args...),
		Type:         model.POST_DEFAULT,
	}
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

func (p *Plugin) validateCommand(action string, parameters []string) *model.CommandResponse {
	switch action {
	case "preview":
		if len(parameters) != 1 {
			return p.responsef("Please specify a team, for which preview should be made.")
		}
	case "list":
		if len(parameters) > 0 {
			return p.responsef("List command does not accept any extra parameters")
		}
	case "set_channel_welcome":
		if len(parameters) == 0 {
			return p.responsef("`set_channel_welcome` command requires the message to be provided")
		}
	case "get_channel_welcome":
		if len(parameters) > 0 {
			return p.responsef("`get_channel_welcome` command does not accept any extra parameters")
		}
	case "delete_channel_welcome":
		if len(parameters) > 0 {
			return p.responsef("`delete_channel_welcome` command does not accept any extra parameters")
		}
	}

	return nil
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) *model.CommandResponse {
	found := false
	for _, message := range p.getWelcomeMessages() {
		if message.TeamName == teamName {
			if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
				return p.responsef("error occured while processing greeting for team `%s`: `%s`", teamName, err)
			}

			found = true
		}
	}

	if !found {
		return p.responsef("team `%s` has not been found", teamName)
	}

	return &model.CommandResponse{}
}

func (p *Plugin) executeCommandList(args *model.CommandArgs) *model.CommandResponse {
	wecomeMessages := p.getWelcomeMessages()

	if len(wecomeMessages) == 0 {
		return p.responsef("There are no welcome messages defined")
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
	return p.responsef(str.String())
}

func (p *Plugin) executeCommandSetWelcome(args *model.CommandArgs) *model.CommandResponse {
	channelInfo, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		return p.responsef("error occured while checking the type of the chanelId `%s`: `%s`", args.ChannelId, appErr)
	}

	if channelInfo.Type == model.CHANNEL_PRIVATE {
		return p.responsef("welcome messages are not supported for direct channels")
	}

	// strings.Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_channel_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := fmt.Sprintf("%s%s", args.ChannelId, WELCOMEBOT_CHANNEL_WELCOME_KEY)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		return p.responsef("error occured while storing the welcome message for the chanel: `%s`", appErr)
	}

	return p.responsef("stored the welcome message:\n%s", message)
}

func (p *Plugin) executeCommandGetWelcome(args *model.CommandArgs) *model.CommandResponse {
	key := fmt.Sprintf("%s%s", args.ChannelId, WELCOMEBOT_CHANNEL_WELCOME_KEY)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return p.responsef("error occured while retrieving the welcome message for the chanel: `%s`", appErr)
	}

	if data == nil {
		return p.responsef("welcome message has not been set yet")
	}

	return p.responsef("Welcome message is:\n%s", string(data))
}

func (p *Plugin) executeCommandDeleteWelcome(args *model.CommandArgs) *model.CommandResponse {
	key := fmt.Sprintf("%s%s", args.ChannelId, WELCOMEBOT_CHANNEL_WELCOME_KEY)
	data, appErr := p.API.KVGet(key)

	if appErr != nil {
		return p.responsef("error occured while retrieving the welcome message for the chanel: `%s`", appErr)
	}

	if data == nil {
		return p.responsef("welcome message has not been set yet")
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		return p.responsef("error occured while deleting the welcome message for the chanel: `%s`", appErr)
	}

	return p.responsef("welcome message has been deleted")
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

	if response := p.validateCommand(action, parameters); response != nil {
		return response, nil
	}

	isSysadmin, err := p.hasSysadminRole(args.UserId)
	if err != nil {
		return p.responsef("authorization failed: %s", err), nil
	}
	if !isSysadmin {
		return p.responsef("/welcomebot commands can only be executed by the user with system admin role"), nil
	}

	switch action {
	case "preview":
		teamName := parameters[0]
		return p.executeCommandPreview(teamName, args), nil
	case "list":
		return p.executeCommandList(args), nil
	case "set_channel_welcome":
		return p.executeCommandSetWelcome(args), nil
	case "get_channel_welcome":
		return p.executeCommandGetWelcome(args), nil
	case "delete_channel_welcome":
		return p.executeCommandDeleteWelcome(args), nil
	case "help":
		fallthrough
	case "":
		text := "###### Mattermost welcomebot Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		return p.responsef(text), nil
	}

	return p.responsef("Unknown action %v", action), nil
}
