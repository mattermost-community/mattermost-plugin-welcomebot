package main

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const commandHelp = `* |/welcomebot preview [team-name] | - preview the welcome message for the given team name. The current user's username will be used to render the template.
* |/welcomebot list| - list the teams for which welcome messages were defined.
The following commands will only be allowed to be run by system admins and users with permission to manage channel roles. |set_channel_welcome|, |get_channel_welcome| and |delete_channel_welcome|.
* |/welcomebot set_channel_welcome [welcome-message]| - set the welcome message for the given channel. Direct channels are not supported.
* |/welcomebot get_channel_welcome| - print the welcome message set for the given channel (if any)
* |/welcomebot delete_channel_welcome| - delete the welcome message for the given channel (if any)
* |/welcomebot set_team_welcome [welcome-message]| - set a brief welcome message for your current team.
* |/welcomebot get_team_welcome| - print the welcome message set for the given team (if any)
* |/welcomebot delete_team_welcome| - delete the dynamic welcome message for the given team (if any)
`

const (
	commandTriggerPreview              = "preview"
	commandTriggerList                 = "list"
	commandTriggerSetChannelWelcome    = "set_channel_welcome"
	commandTriggerGetChannelWelcome    = "get_channel_welcome"
	commandTriggerDeleteChannelWelcome = "delete_channel_welcome"
	commandTriggerHelp                 = "help"
	commandTriggerSetTeamWelcome       = "set_team_welcome"
	commandTriggerGetTeamWelcome       = "get_team_welcome"
	commandTriggerDeleteTeamWelcome    = "delete_team_welcome"

	// Error Message Constants
	unsetMessageError     = "Welcome message has not been set for the team"
	pluginPermissionError = "`/welcomebot %s` commands can only be executed by the user with a system admin role, team admin role, or channel admin role"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "welcomebot",
		DisplayName:      "welcomebot",
		Description:      "Welcome Bot helps add new team members to channels.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: preview, help, list, set_channel_welcome, get_channel_welcome, delete_channel_welcome, set_team_welcome, get_team_welcome, delete_team_welcome",
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
	if !strings.Contains(user.Roles, model.PermissionsSystemAdmin) {
		return false, nil
	}
	return true, nil
}

func (p *Plugin) hasTeamAdminRole(userID string, teamID string) (bool, error) {
	teamMember, err := p.client.Team.GetMember(teamID, userID)
	if err != nil {
		return false, err
	}
	if !strings.Contains(teamMember.Roles, model.PermissionsTeamAdmin) {
		return false, nil
	}
	return true, nil
}

func (p *Plugin) hasChannelAdminRole(userID string, channelID string) (bool, error) {
	channelMember, err := p.client.Channel.GetMember(channelID, userID)
	if err != nil {
		return false, err
	}
	if !strings.Contains(channelMember.Roles, model.PermissionsChannelAdmin) {
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
			return "`" + commandTriggerSetChannelWelcome + "` command requires the message to be provided"
		}
	case commandTriggerGetChannelWelcome:
		if len(parameters) > 0 {
			return "`" + commandTriggerGetChannelWelcome + "` command does not accept any extra parameters"
		}
	case commandTriggerDeleteChannelWelcome:
		if len(parameters) > 0 {
			return "`" + commandTriggerDeleteChannelWelcome + "` command does not accept any extra parameters"
		}
	case commandTriggerSetTeamWelcome:
		if len(parameters) == 0 {
			return "`" + commandTriggerSetTeamWelcome + "` command requires the message to be provided"
		}
	case commandTriggerGetTeamWelcome:
		if len(parameters) > 0 {
			return "`" + commandTriggerGetTeamWelcome + "` command does not accept any extra parameters"
		}
	case commandTriggerDeleteTeamWelcome:
		if len(parameters) > 0 {
			return "`" + commandTriggerDeleteTeamWelcome + "` command does not accept any extra parameters"
		}
	}

	return ""
}

func (p *Plugin) validatePreviewPrivileges(teamID string, args *model.CommandArgs) (bool, error) {
	if _, err := p.API.GetTeamMember(teamID, args.UserId); err != nil {
		if err.StatusCode == http.StatusNotFound {
			p.postCommandResponse(args, "You are not a member of the team.")
			p.client.Log.Info("The user is not a member of the team.", "UserID", args.UserId, "TeamID", teamID)
			return false, err
		}
		p.postCommandResponse(args, "Error occurred while getting the Team member")
		p.client.Log.Error("Error occurred while getting the Team member.", "TeamID", teamID, "Error", err.Error())
		return false, err
	}

	return p.isSystemOrTeamAdmin(args, args.UserId, teamID), nil
}

// checks if the user has System or Team Admin access to the given team
func (p *Plugin) isSystemOrTeamAdmin(args *model.CommandArgs, userID, teamID string) bool {
	isSysadmin, err := p.hasSysadminRole(userID)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while checking the System Admin Role.")
		p.client.Log.Error("Error occurred while checking the System Admin Role.", "UserID", userID, "Error", err.Error())
		return false
	}

	isTeamAdmin, err := p.hasTeamAdminRole(userID, teamID)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while checking the Team Admin Role for the user.")
		p.client.Log.Error("Error occurred while checking the Team Admin Role for the user.", "UserID", userID, "TeamID", teamID, "Error", err.Error())
		return false
	}

	if !isSysadmin && !isTeamAdmin {
		p.postCommandResponse(args, "You do not have the proper privileges to control this Team's welcome messages.")
		p.client.Log.Info("User does not have the proper privileges to control the Team's welcome messages.", "UserID", userID, "TeamID", teamID)
		return false
	}

	return true
}

// This retrieves a map of team IDs with their respective welcome message from the KV store
func (p *Plugin) getTeamKVWelcomeMessagesMap(args *model.CommandArgs) map[string]string {
	teamsList, err := p.client.Team.List()
	if err != nil {
		p.postCommandResponse(args, "Error occurred while getting list of the teams: %s", err.Error())
		p.client.Log.Error("Error occurred while getting list of the teams", "Error", err.Error())
		return make(map[string]string)
	}

	teamsAndKVWelcomeMessagesMap := make(map[string]string)

	for _, team := range teamsList {
		key := fmt.Sprintf("%s%s", welcomebotTeamWelcomeKey, team.Id)
		var teamMessage string
		if err := p.client.KV.Get(key, teamMessage); err != nil {
			p.postCommandResponse(args, "Error occurred while retrieving the welcome messages: %s", err.Error())
			p.client.Log.Error("Error occurred while retrieving the welcome messages from KV store.", "Error", err.Error())
			return make(map[string]string)
		}
		if teamMessage != "" {
			teamsAndKVWelcomeMessagesMap[team.Id] = teamMessage
		}
	}
	return teamsAndKVWelcomeMessagesMap
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) {
	// Retrieve Team to check if a message already exists within the KV pair set
	team, err := p.client.Team.GetByName(teamName)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the team: `%s`", err.Error())
		p.client.Log.Error("Error occurred while retrieving the the team data.", "TeamName", teamName, "Error", err.Error())
		return
	}
	teamID := team.Id

	validPrivileges, err := p.validatePreviewPrivileges(teamID, args)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the team.: `%s`", err.Error())
		p.client.Log.Error("Error occurred while retrieving validating preview privilege for the team.", "TeamID", teamID, "Error", err.Error())
		return
	}
	if !validPrivileges {
		return
	}

	key := fmt.Sprintf("%s%s", welcomebotTeamWelcomeKey, teamID)
	var data string
	if err = p.client.KV.Get(key, data); err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the team: `%s`", err.Error())
		p.client.Log.Error("Error occurred while retrieving team welcome message from KV store.", "TeamID", teamID, "Error", err.Error())
		return
	}

	if data != "" {
		// Create ephemeral team welcome message
		p.postCommandResponse(args, data)
		return
	}

	// no dynamic message is set so we check the config for a message
	for _, message := range p.getWelcomeMessages() {
		if message.TeamName == teamName {
			if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
				p.postCommandResponse(args, "Error occurred while processing the greeting for the team `%s`: `%s`", teamName, err.Error())
				return
			}
			return
		}
	}
	p.postCommandResponse(args, "team `%s` has not been found", teamName)
}

func (p *Plugin) executeCommandList(args *model.CommandArgs) {
	isSysadmin, sysAdminError := p.hasSysadminRole(args.UserId)
	if sysAdminError != nil {
		p.postCommandResponse(args, "Error occurred while getting the System Admin Role `%s`: `%s`", args.TeamId, sysAdminError.Error())
		p.client.Log.Error("Error occurred while getting the System Admin Role", "TeamID", args.TeamId, "Error", sysAdminError.Error())
		return
	}

	if !isSysadmin {
		p.postCommandResponse(args, "Only a System Admin can view all welcome messages of teams.")
		return
	}

	welcomeMessagesFromConfig := p.getWelcomeMessages()
	welcomeMessagesFromKVSMap := p.getTeamKVWelcomeMessagesMap(args)
	if len(welcomeMessagesFromConfig) == 0 && len(welcomeMessagesFromKVSMap) == 0 {
		p.postCommandResponse(args, "There are no welcome messages defined")
		return
	}

	// Deduplicate entries
	teams := make(map[string]struct{})
	for _, message := range welcomeMessagesFromConfig {
		teams[message.TeamName] = struct{}{}
	}

	var str strings.Builder
	teamsWithWelcomeMessages := p.getUniqueTeamsWithWelcomeMsgSlice(teams, welcomeMessagesFromKVSMap)
	for _, team := range teamsWithWelcomeMessages {
		str.WriteString(fmt.Sprintf("\n * %s", team))
	}
	p.postCommandResponse(args, str.String())
}

func (p *Plugin) executeCommandSetWelcome(args *model.CommandArgs) {
	channelInfo, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		p.postCommandResponse(args, "Error occurred while checking the type of the chanelId `%s`: `%s`", args.ChannelId, appErr)
		return
	}

	if channelInfo.Type == model.ChannelTypePrivate {
		p.postCommandResponse(args, "Welcome messages are not supported for direct channels")
		return
	}

	// strings.Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_channel_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		p.postCommandResponse(args, "Error occurred while storing the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "Stored the channel welcome message:\n%s", message)
}

func (p *Plugin) executeCommandGetWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the chanel: `%s`", appErr.Message)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "Welcome message has not been set yet")
		return
	}

	p.postCommandResponse(args, "Welcome message is:\n%s", string(data))
}

func (p *Plugin) executeCommandDeleteWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)

	if appErr != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the chanel: `%s`", appErr.Message)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "Welcome message has not been set yet")
		return
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		p.postCommandResponse(args, "Error occurred while deleting the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "Welcome message has been deleted")
}

func (p *Plugin) executeCommandSetTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	// Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	subCommands := strings.SplitN(args.Command, commandTriggerSetTeamWelcome, 2)
	if len(subCommands) != 2 {
		p.postCommandResponse(args, "Error occurred while extracting the welcome message from the command")
		return
	}

	message := strings.TrimSpace(subCommands[1])

	key := makeTeamWelcomeMessageKey(args.TeamId)
	if isValueSet, err := p.client.KV.Set(key, &message); err != nil || !isValueSet {
		p.postCommandResponse(args, "Error occurred while storing the welcome message for the team: `%s`", err.Error())
		return
	}

	p.postCommandResponse(args, "Stored the team welcome message:\n%s", message)
}

func (p *Plugin) executeCommandDeleteTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	if _, err := p.GetTeamWelcomeMessageFromKV(args.TeamId); err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the team: `%s`", err.Error())
		return
	}

	key := makeTeamWelcomeMessageKey(args.TeamId)
	if err := p.client.KV.Delete(key); err != nil {
		p.postCommandResponse(args, "Error occurred while deleting the welcome message for the team: `%s`", err.Error())
		return
	}

	p.postCommandResponse(args, "Team welcome message has been deleted")
}

func (p *Plugin) executeCommandGetTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	data, err := p.GetTeamWelcomeMessageFromKV(args.TeamId)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the welcome message for the team: `%s`", err.Error())
		return
	}

	// retrieve team name through the teamid
	team, err := p.client.Team.Get(args.TeamId)
	if err != nil {
		p.postCommandResponse(args, "Error occurred while retrieving the team: `%s`", err.Error())
		return
	}

	if data != "" {
		p.postCommandResponse(args, data)
		return
	}

	for _, message := range p.getWelcomeMessages() {
		if message.TeamName == team.Name {
			if err := p.previewWelcomeMessage(team.Name, args, *message); err != nil {
				p.postCommandResponse(args, "Error occurred while processing greeting for team `%s`: `%s`", team.Name, err.Error())
				p.client.Log.Error("Error occurred while processing greeting for team.", "TeamName", team.Name, "Error", err.Error())
			}
			return
		}
	}

	// Show unset error message if no message is stored in KV and config.json file
	p.postCommandResponse(args, unsetMessageError)
}

func (p *Plugin) checkCommandPermission(args *model.CommandArgs, action string) (bool, error) {
	isSysadmin, err := p.hasSysadminRole(args.UserId)
	if err != nil {
		p.postCommandResponse(args, "System admin authorization failed: %s", err.Error())
		return true, err
	}

	isTeamAdmin, teamAdminErr := p.hasTeamAdminRole(args.UserId, args.TeamId)
	if teamAdminErr != nil {
		p.postCommandResponse(args, "Team admin authorization failed: %s", teamAdminErr)
		return true, teamAdminErr
	}

	isChannelAdmin, channelAdminErr := p.hasChannelAdminRole(args.UserId, args.ChannelId)
	if channelAdminErr != nil {
		p.postCommandResponse(args, "Channel admin authorization failed: %s", channelAdminErr)
		return true, channelAdminErr
	}

	if !isSysadmin && !isTeamAdmin && !isChannelAdmin {
		p.postCommandResponse(args, fmt.Sprintf(pluginPermissionError, action))
		return true, errors.New(pluginPermissionError)
	}

	return false, nil
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

	err, _ := p.checkCommandPermission(args, action)
	if err {
		return &model.CommandResponse{}, nil
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
	case commandTriggerSetTeamWelcome:
		p.executeCommandSetTeamWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerGetTeamWelcome:
		p.executeCommandGetTeamWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerDeleteTeamWelcome:
		p.executeCommandDeleteTeamWelcome(args)
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

func (p *Plugin) getUniqueTeamsWithWelcomeMsgSlice(teamsWithConfigWelcomeMsg map[string]struct{}, teamsWithKVWelcomeMsg map[string]string) []string {
	var uniqueTeams []string
	// Place all keys into one list
	teamsWithConfigWelcomeKeys := convertStringMapIntoKeySlice(teamsWithConfigWelcomeMsg)
	teamIDsWithKVWelcomeKeys := convertStringMapIntoKeySlice(teamsWithKVWelcomeMsg)

	// Convert the ids into team names before combining into 1 large list
	teamsWithKVWelcomeKeys := []string{}
	for _, id := range teamIDsWithKVWelcomeKeys {
		team, err := p.client.Team.Get(id)
		if err != nil {
			continue
		}
		teamsWithKVWelcomeKeys = append(teamsWithKVWelcomeKeys, team.Name)
	}

	allTeamNames := teamsWithConfigWelcomeKeys
	allTeamNames = append(allTeamNames, teamsWithKVWelcomeKeys...)

	// Leverage the unique priniciple of keys in a map to store unique values as they are encountered
	teamNameMap := make(map[string]bool)
	for _, teamName := range allTeamNames {
		if !teamNameMap[teamName] {
			uniqueTeams = append(uniqueTeams, teamName)
		}
		teamNameMap[teamName] = true
	}

	return uniqueTeams
}

// Takes maps whose keys are strings, and whose values are anything.
func convertStringMapIntoKeySlice(mapInput interface{}) []string {
	// need to check that input is a map or a slice before continuing
	reflectValue := reflect.ValueOf(mapInput)
	if reflectValue.Kind() != reflect.Map {
		return nil
	}

	// Grabs all keys and makes a list of the keys.
	keys := make([]string, 0, len(reflectValue.MapKeys()))
	for _, keyReflectValue := range reflectValue.MapKeys() {
		key := keyReflectValue.Interface().(string)
		keys = append(keys, key)
	}
	return keys
}

func getAutocompleteData() *model.AutocompleteData {
	welcomebot := model.NewAutocompleteData("welcomebot", "[command]",
		"Available commands: preview, help, list, set_channel_welcome, get_channel_welcome, delete_channel_welcome, "+
			commandTriggerSetTeamWelcome+", "+commandTriggerGetTeamWelcome+", "+commandTriggerDeleteTeamWelcome)

	preview := model.NewAutocompleteData("preview", "[team-name]", "Preview the welcome message for the given team name")
	preview.AddTextArgument("Team name to preview welcome message", "[team-name]", "")
	welcomebot.AddCommand(preview)

	list := model.NewAutocompleteData("list", "", "Lists team welcome messages")
	welcomebot.AddCommand(list)

	setChannelWelcome := model.NewAutocompleteData("set_channel_welcome", "[welcome-message]", "Set the welcome message for the channel")
	setChannelWelcome.AddTextArgument("Welcome message for the channel", "[welcome-message]", "")
	welcomebot.AddCommand(setChannelWelcome)

	getChannelWelcome := model.NewAutocompleteData("get_channel_welcome", "", "Print the welcome message set for the channel")
	getChannelWelcome.AddTextArgument("Channel name to get the welcome message", "[channel-name]", "")
	welcomebot.AddCommand(getChannelWelcome)

	deleteChannelWelcome := model.NewAutocompleteData("delete_channel_welcome", "", "Delete the welcome message for the channel")
	deleteChannelWelcome.AddTextArgument("Channel name to delete the welcome message", "[channel-name]", "")
	welcomebot.AddCommand(deleteChannelWelcome)

	setTeamWelcome := model.NewAutocompleteData(commandTriggerSetTeamWelcome, "[welcome-message]", "Set the welcome message for the team")
	setTeamWelcome.AddTextArgument("Welcome message for the current team", "[welcome-message]", "")
	welcomebot.AddCommand(setTeamWelcome)

	getTeamWelcome := model.NewAutocompleteData(commandTriggerGetTeamWelcome, "", "Print the welcome message for the team")
	welcomebot.AddCommand(getTeamWelcome)

	deleteTeamWelcome := model.NewAutocompleteData(commandTriggerDeleteTeamWelcome, "", "Delete the welcome message for the team. Configuration based messages are not affected by this.")
	welcomebot.AddCommand(deleteTeamWelcome)

	return welcomebot
}
