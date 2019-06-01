package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// RegisterCommand to registers welcomebot commands
func (p *Plugin) RegisterCommand(teamID string) error {
	if err := p.API.RegisterCommand(&model.Command{
		TeamId:           teamID,
		Trigger:          welcomeBotUsername,
		AutoComplete:     true,
		AutoCompleteHint: "[team_name]",
		AutoCompleteDesc: "Re-runs the welcomebot for the supplied team",
		DisplayName:      "WelcomeBot Command",
		Description:      "Re-runs the welcomebot with the supplied team",
	}); err != nil {
		p.API.LogError(
			"failed to register command",
			"error", err.Error(),
		)
	}

	return nil
}

// ExecuteCommand executes a command that has been previously registered via the RegisterCommand
// API.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !strings.HasPrefix(args.Command, "/"+welcomeBotUsername) {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown command: " + args.Command),
		}, nil
	}

	team, _ := p.API.GetTeamByName(args.Command)
	if team == nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown team name: " + args.Command),
		}, nil
	}

	teamMember, _ := p.API.GetTeamMember(team.Id, args.UserId)
	if teamMember == nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("You do not appear to be part of the team: " + args.Command),
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("TODO re-run the WelcomeBot"),
	}, nil
}
