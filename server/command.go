package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/mattermost/mattermost-server/v5/model"
)

const COMMAND_HELP = `* |/welcomebot preview [team-name] [user-name]| - preview the welcome message for the given team name. The current user's username will be used to render the template.
* |/welcomebot list| - list the teams for which welcome messages were defined`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "welcomebot",
		DisplayName:      "welcomebot",
		Description:      "Welcome Bot helps add new team members to channels.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: preview, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   text,
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
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
		p.postCommandResponse(args, fmt.Sprintf("authorization failed: %w", err))
		return &model.CommandResponse{}, nil
	}
	if !isSysadmin {
		p.postCommandResponse(args, "/welcomebot commands can only be executed by the user with system admin role")
		return &model.CommandResponse{}, nil
	}

	switch action {
	case "preview":
		if len(parameters) != 1 {
			p.postCommandResponse(args, "Please specify a team, for which preview should be made.")
			return &model.CommandResponse{}, nil
		}

		teamName := parameters[0]

		found := false
		for _, message := range p.getWelcomeMessages() {
			if message.TeamName == teamName {
				if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
					errMsg := fmt.Sprintf("error occured while processing greeting for team `%s`: `%s`", teamName, err)
					p.postCommandResponse(args, errMsg)
					return &model.CommandResponse{}, nil
				}

				found = true
			}
		}

		if !found {
			p.postCommandResponse(args, fmt.Sprintf("team `%s` has not been found", teamName))
		}
		return &model.CommandResponse{}, nil
	case "list":
		if len(parameters) > 0 {
			p.postCommandResponse(args, "List command does not accept any extra parameters")
			return &model.CommandResponse{}, nil
		}

		wecomeMessages := p.getWelcomeMessages()

		if len(wecomeMessages) == 0 {
			p.postCommandResponse(args, "There are no welcome messages defined")
			return &model.CommandResponse{}, nil
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
		return &model.CommandResponse{}, nil
	case "help":
		fallthrough
	case "":
		text := "###### Mattermost welcomebot Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, fmt.Sprintf("Unknown action %v", action))

	return &model.CommandResponse{}, nil
}
