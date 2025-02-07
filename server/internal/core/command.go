package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	commandTriggerHelp = "help"
)

func (p *Plugin) GetCommand() *model.Command {
	return &model.Command{
		Trigger:          "welcomebot",
		DisplayName:      "welcomebot",
		Description:      "Welcome Bot helps add new team members to channels.",
		AutoComplete:     true,
		AutoCompleteDesc: p.AutoCompleteDesc(),
		AutoCompleteHint: "[command]",
		AutocompleteData: p.GetAutocompleteData(),
	}
}

func (p *Plugin) PostCommandResponse(args *model.CommandArgs, text string, textArgs ...interface{}) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		Message:   fmt.Sprintf(text, textArgs...),
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) AutoCompleteDesc() string {
	triggers := make([]string, len(p.commands)+1)

	for key := range p.commands {
		triggers = append(triggers, key)
	}

	triggers = append(triggers, commandTriggerHelp)
	result := fmt.Sprintf("Available commands: %s", strings.Join(triggers, ", "))

	return result
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

	isSysadmin, err := p.hasSysadminRole(args.UserId)
	if err != nil {
		p.PostCommandResponse(args, "authorization failed: %s", err)
		return &model.CommandResponse{}, nil
	}

	if !p.canExecuteCommands(isSysadmin, args) {
		p.PostCommandResponse(args, "The `/welcomebot %s` command can only be executed by system admins and channel admins.", action)
		return &model.CommandResponse{}, nil
	}

	if action == "" || action == commandTriggerHelp {
		p.printHelp(args)
		return &model.CommandResponse{}, nil
	}

	commandObj, ok := p.commands[action]

	if ok {
		err := commandObj.Validate(parameters)
		if err != nil {
			p.PostCommandResponse(args, err.Error())
			return &model.CommandResponse{}, nil
		}

		commandObj.Execute(p, args)
		return &model.CommandResponse{}, nil
	}

	p.PostCommandResponse(args, "Unknown action %v", action)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) canExecuteCommands(isSysadmin bool, args *model.CommandArgs) bool {
	if !isSysadmin {
		hasPermissionTo := p.API.HasPermissionToChannel(args.UserId, args.ChannelId, model.PermissionManageChannelRoles)
		if !hasPermissionTo {
			return false
		}
	}

	return true
}

func (p *Plugin) printHelp(args *model.CommandArgs) {
	commandsHelp := make([]string, 0, len(p.commands))
	for _, command := range p.commands {
		commandsHelp = append(commandsHelp, command.Help())
	}

	sort.Strings(commandsHelp)

	text := "###### Mattermost welcomebot Plugin - Slash Command Help\n" + strings.Join(commandsHelp, "\n")
	p.PostCommandResponse(args, text)
}

func (p *Plugin) GetAutocompleteData() *model.AutocompleteData {
	triggers := make([]string, len(p.commands))

	for key := range p.commands {
		triggers = append(triggers, key)
	}

	description := fmt.Sprintf("Available commands: %s", strings.Join(triggers, ", "))
	welcomebot := model.NewAutocompleteData("welcomebot", "[command]", description)

	for _, command := range p.commands {
		welcomebot.AddCommand(command.AutocompleteData())
	}

	return welcomebot
}
