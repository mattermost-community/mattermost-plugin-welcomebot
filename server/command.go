package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const commandHelp = `* |/welcomebot preview [team-name] | - preview the welcome message for the given team name. The current user's username will be used to render the template.
* |/welcomebot list| - list the teams for which welcome messages were defined.
The following commands will only be allowed to be run by system admins and users with permission to manage channel roles. |set_channel_welcome|, |get_channel_welcome| and |delete_channel_welcome|.
* |/welcomebot set_channel_welcome [welcome-message]| - set the welcome message for the given channel. Direct channels are not supported.
* |/welcomebot get_channel_welcome| - print the welcome message set for the given channel (if any)
* |/welcomebot delete_channel_welcome| - delete the welcome message for the given channel (if any)
`

const (
	commandTriggerPreview              = "preview"
	commandTriggerList                 = "list"
	commandTriggerSetChannelWelcome    = "set_channel_welcome"
	commandTriggerGetChannelWelcome    = "get_channel_welcome"
	commandTriggerDeleteChannelWelcome = "delete_channel_welcome"
	commandTriggerHelp                 = "help"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "welcomebot",
		DisplayName:      "welcomebot",
		Description:      "Welcome Bot helps add new team members to channels.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: preview, help, list, set_channel_welcome, get_channel_welcome, delete_channel_welcome",
		AutoCompleteHint: "[command]",
		AutocompleteData: getAutocompleteData(),
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

func (p *Plugin) hasSysadminRole(userID string) (bool, error) {
	user, appErr := p.API.GetUser(userID)
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
	case commandTriggerPreview:
		if len(parameters) != 1 {
			return "Please specify a team, for which preview should be made."
		}
	case commandTriggerList:
		if len(parameters) > 0 {
			return "List command does not accept any extra parameters"
		}
	case commandTriggerSetChannelWelcome:
		if len(parameters) == 0 {
			return "`set_channel_welcome` command requires the message to be provided"
		}
	case commandTriggerGetChannelWelcome:
		if len(parameters) > 0 {
			return "`get_channel_welcome` command does not accept any extra parameters"
		}
	case commandTriggerDeleteChannelWelcome:
		if len(parameters) > 0 {
			return "`delete_channel_welcome` command does not accept any extra parameters"
		}
	}

	return ""
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) {
	found := false
	for _, message := range p.getWelcomeMessages() {
		var teamNamesArr = strings.Split(message.TeamName, ",")
		for _, name := range teamNamesArr {
			tn := strings.TrimSpace(name)
			if tn == teamName {
				p.postCommandResponse(args, "%s", teamName)
				if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
					p.postCommandResponse(args, "error occurred while processing greeting for team `%s`: `%s`", teamName, err)
					return
				}
				found = true
			}
		}
	}

	if !found {
		p.postCommandResponse(args, "team `%s` has not been found", teamName)
	}
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
}

func (p *Plugin) executeCommandSetWelcome(args *model.CommandArgs) {
	channelInfo, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while checking the type of the chanelId `%s`: `%s`", args.ChannelId, appErr)
		return
	}

	if channelInfo.Type == model.ChannelTypeDirect {
		p.postCommandResponse(args, "welcome messages are not supported for direct channels")
		return
	}

	// strings.Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_channel_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		p.postCommandResponse(args, "error occurred while storing the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "stored the welcome message:\n%s", message)
}

func (p *Plugin) executeCommandGetWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	p.postCommandResponse(args, "Welcome message is:\n%s", string(data))
}

func (p *Plugin) executeCommandDeleteWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)

	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		p.postCommandResponse(args, "error occurred while deleting the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "welcome message has been deleted")
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
		if action == commandTriggerSetChannelWelcome || action == commandTriggerGetChannelWelcome || action == commandTriggerDeleteChannelWelcome {
			if hasPermissionTo := p.API.HasPermissionToChannel(args.UserId, args.ChannelId, model.PermissionManageChannelRoles); !hasPermissionTo {
				p.postCommandResponse(args, "The `/welcomebot %s` command can only be executed by system admins and channel admins.", action)
				return &model.CommandResponse{}, nil
			}
		}
	}

	switch action {
	case commandTriggerPreview:
		teamName := parameters[0]
		p.executeCommandPreview(teamName, args)
		return &model.CommandResponse{}, nil
	case commandTriggerList:
		p.executeCommandList(args)
		return &model.CommandResponse{}, nil
	case commandTriggerSetChannelWelcome:
		p.executeCommandSetWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerGetChannelWelcome:
		p.executeCommandGetWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerDeleteChannelWelcome:
		p.executeCommandDeleteWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerHelp:
		fallthrough
	case "":
		text := "###### Mattermost welcomebot Plugin - Slash Command Help\n" + strings.ReplaceAll(commandHelp, "|", "`")
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, "Unknown action %v", action)
	return &model.CommandResponse{}, nil
}

func getAutocompleteData() *model.AutocompleteData {
	welcomebot := model.NewAutocompleteData("welcomebot", "[command]",
		"Available commands: preview, help, list, set_channel_welcome, get_channel_welcome, delete_channel_welcome")

	preview := model.NewAutocompleteData("preview", "[team-name]", "Preview the welcome message for the given team name")
	preview.AddTextArgument("Team name to preview welcome message", "[team-name]", "")
	welcomebot.AddCommand(preview)

	list := model.NewAutocompleteData("list", "", "Lists team welcome messages")
	welcomebot.AddCommand(list)

	setChannelWelcome := model.NewAutocompleteData("set_channel_welcome", "[welcome-message]", "Set the welcome message for the channel")
	setChannelWelcome.AddTextArgument("Welcome message for the channel", "[welcome-message]", "")
	welcomebot.AddCommand(setChannelWelcome)

	getChannelWelcome := model.NewAutocompleteData("get_channel_welcome", "", "Print the welcome message set for the channel")
	welcomebot.AddCommand(getChannelWelcome)

	deleteChannelWelcome := model.NewAutocompleteData("delete_channel_welcome", "", "Delete the welcome message for the channel")
	welcomebot.AddCommand(deleteChannelWelcome)

	return welcomebot
}
