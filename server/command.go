package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const commandHelp = `* |/welcomebot preview [team-name] | - preview the welcome message for the given team name. The current user's username will be used to render the template.
* |/welcomebot list| - list the teams for which welcome messages were defined
* |/welcomebot set_channel_welcome [welcome-message]| - set the welcome message for the given channel. Direct channels are not supported.
* |/welcomebot get_channel_welcome| - print the welcome message set for the given channel (if any)
* |/welcomebot delete_channel_welcome| - delete the welcome message for the given channel (if any)
* |/welcomebot set_team_welcome [welcome-message]| - set a brief text welcome message for your given team.
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
	unsetMessageError = "welcome message has not been set"
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
	if !strings.Contains(user.Roles, model.PERMISSIONS_SYSTEM_ADMIN) {
		return false, nil
	}
	return true, nil
}

func (p *Plugin) hasTeamAdminRole(userID string, teamID string) (bool, error) {
	teamMember, appErr := p.API.GetTeamMember(teamID, userID)
	if appErr != nil {
		return false, appErr
	}
	if !strings.Contains(teamMember.Roles, model.PERMISSIONS_TEAM_ADMIN) {
		return false, nil
	}
	return true, nil
}

func (p *Plugin) hasChannelAdminRole(userID string, channelID string) (bool, error) {
	channelMember, appErr := p.API.GetChannelMember(channelID, userID)
	if appErr != nil {
		return false, appErr
	}
	if !strings.Contains(channelMember.Roles, model.PERMISSIONS_CHANNEL_ADMIN) {
		return false, nil
	}
	return true, nil
}

// Commented out for simplicity of welcome bot. Once restriction is wanted, this can be uncommented.
// func (p *Plugin) checkIfTownSquare(channelID string) (bool, error) {
// 	channel, channelErr := p.API.GetChannel(channelID)
// 	if channelErr != nil {
// 		return false, channelErr
// 	}
// 	if channel.Name != model.DEFAULT_CHANNEL {
// 		return false, nil
// 	}
// 	return true, nil
// }

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
	_, teamMemberErr := p.API.GetTeamMember(teamID, args.UserId)
	if teamMemberErr != nil {
		if teamMemberErr.StatusCode == 404 {
			p.postCommandResponse(args, "You are not a member of that team.")
			return false, teamMemberErr
		}
		p.postCommandResponse(args, "error occurred while getting the Team Admin Role `%s`: `%s`", teamID, teamMemberErr)
		return false, teamMemberErr
	}
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, teamID)

	return doesUserHavePrivileges, nil
}

// checks if the user has System or Team Admin access to the given team
func (p *Plugin) isSystemOrTeamAdmin(args *model.CommandArgs, userID string, teamID string) bool {
	isSysadmin, sysAdminError := p.hasSysadminRole(userID)
	isTeamAdmin, teamAdminError := p.hasTeamAdminRole(userID, teamID)

	if sysAdminError != nil {
		p.postCommandResponse(args, "error occurred while getting the System Admin Role: `%s`", sysAdminError)
		return false
	}
	if teamAdminError != nil {
		p.postCommandResponse(args, "error occurred while getting the Team Admin Role `%s`: `%s`", teamID, teamAdminError)
		return false
	}
	if !isSysadmin && !isTeamAdmin {
		p.postCommandResponse(args, "You do not have the proper privileges to control this Team's welcome messages.")
		return false
	}
	return true
}

// This retrieves a map of team Ids with their respective welcome message
func (p *Plugin) getTeamKVWelcomeMessagesMap(args *model.CommandArgs) map[string]string {
	teamsList, teamErr := p.API.GetTeams()
	if teamErr != nil {
		p.postCommandResponse(args, "Error occurred while getting list of teams: %s", teamErr)
		return make(map[string]string)
	}

	teamsAndKVWelcomeMessagesMap := make(map[string]string)

	for _, team := range teamsList {
		key := fmt.Sprintf("%s%s", welcomebotTeamWelcomeKey, team.Id)
		teamMessage, appErr := p.API.KVGet(key)
		if appErr != nil {
			p.postCommandResponse(args, "Error occurred while retrieving the welcome messages: %s", appErr)
			return make(map[string]string)
		}
		if teamMessage != nil {
			teamsAndKVWelcomeMessagesMap[team.Id] = string(teamMessage)
		}
	}
	return teamsAndKVWelcomeMessagesMap
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) {
	// Retrieve Team to check if a message already exists within the KV pair set
	team, err := p.API.GetTeamByName(teamName)
	if err != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the team: `%s`", err)
		return
	}
	teamID := team.Id

	validPrivileges, _ := p.validatePreviewPrivileges(teamID, args)
	if !validPrivileges {
		return
	}

	key := fmt.Sprintf("%s%s", welcomebotTeamWelcomeKey, teamID)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the team: `%s`", appErr)
		return
	}

	if len(data) == 0 {
		// no dynamic message is set so we check the config for a message
		found := false
		for _, message := range p.getWelcomeMessages() {
			if message.TeamName == teamName {
				if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
					p.postCommandResponse(args, "error occurred while processing greeting for team `%s`: `%s`", teamName, err)
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
	// Create ephemeral team welcome message
	p.postCommandResponse(args, string(data))
}

func (p *Plugin) executeCommandList(args *model.CommandArgs) {
	isSysadmin, sysAdminError := p.hasSysadminRole(args.UserId)
	if sysAdminError != nil {
		p.postCommandResponse(args, "error occurred while getting the System Admin Role `%s`: `%s`", args.TeamId, sysAdminError)
		return
	}
	if !isSysadmin {
		p.postCommandResponse(args, "Only a System Admin can view all teams with welcome messages.")
		return
	}

	welcomeMessagesFromConfig := p.getWelcomeMessages()
	welcomeMessagesFromKVsMap := p.getTeamKVWelcomeMessagesMap(args)
	if len(welcomeMessagesFromConfig) == 0 && len(welcomeMessagesFromKVsMap) == 0 {
		p.postCommandResponse(args, "There are no welcome messages defined")
		return
	}

	// Deduplicate entries
	teams := make(map[string]struct{})
	for _, message := range welcomeMessagesFromConfig {
		teams[message.TeamName] = struct{}{}
	}

	var str strings.Builder
	teamsWithWelcomeMessages := p.getUniqueTeamsWithWelcomeMsgSlice(teams, welcomeMessagesFromKVsMap)
	for _, team := range teamsWithWelcomeMessages {
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

	if channelInfo.Type == model.CHANNEL_PRIVATE {
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

	p.postCommandResponse(args, "stored the channel welcome message:\n%s", message)
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

func (p *Plugin) executeCommandSetTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	// Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_team_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := makeTeamWelcomeMessageKey(args.TeamId)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		p.postCommandResponse(args, "error occurred while storing the welcome message for the team: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "stored the team welcome message:\n%s", message)
}

func (p *Plugin) executeCommandDeleteTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	key := makeTeamWelcomeMessageKey(args.TeamId)
	_, appErr := p.GetTeamWelcomeMessageFromKV(args.TeamId)

	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the team: `%s`", appErr)
		return
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		p.postCommandResponse(args, "error occurred while deleting the welcome message for the team: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "team welcome message has been deleted")
}

func (p *Plugin) executeCommandGetTeamWelcome(args *model.CommandArgs) {
	doesUserHavePrivileges := p.isSystemOrTeamAdmin(args, args.UserId, args.TeamId)
	if !doesUserHavePrivileges {
		return
	}

	data, appErr := p.GetTeamWelcomeMessageFromKV(args.TeamId)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the team: `%s`", appErr)
		return
	}

	// retrieve team name through the teamid
	team, err := p.API.GetTeam(args.TeamId)
	if err != nil {
		p.postCommandResponse(args, err.Error())
		return
	}

	if data == nil {
		for _, message := range p.getWelcomeMessages() {
			if message.TeamName == team.Name {
				if err := p.previewWelcomeMessage(team.Name, args, *message); err != nil {
					p.postCommandResponse(args, "error occurred while processing greeting for team `%s`: `%s`", team.Name, err)
				}
				return
			}
		}
		// if KV do not have message, and Config.json does not have message, then there is no message. Display Error case.
		p.postCommandResponse(args, unsetMessageError)
		return
	}
	p.postCommandResponse(args, string(data))
}

func (p *Plugin) checkCommandPermission(args *model.CommandArgs) (bool, error) {
	isSysadmin, err := p.hasSysadminRole(args.UserId)
	if err != nil {
		p.postCommandResponse(args, "authorization failed: %s", err)
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
		errMsg := "/welcomebot commands can only be executed by the user with a system admin role, team admin role, or channel admin role"
		p.postCommandResponse(args, errMsg)
		return true, errors.New(errMsg)
	}

	// Commented Out In case for future iterations of welcome bot we want to restrict channel welcome messages for town square.
	// isTownSquare, channelErr := p.checkIfTownSquare(args.ChannelId)
	// if channelErr != nil {
	// 	p.postCommandResponse(args, "Channel authorization failed: %s", channelAdminErr)
	// 	return true, channelErr
	// }
	// if !isSysadmin && !isTeamAdmin && isChannelAdmin && isTownSquare {
	// 	errMsg := "/welcomebot commands cannot be executed by a channel admin in Town Square"
	// 	p.postCommandResponse(args, errMsg)
	// 	return true, errors.New(errMsg)
	// }
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

	err, _ := p.checkCommandPermission(args)
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

func (p *Plugin) getUniqueTeamsWithWelcomeMsgSlice(teamsWithConfigWelcome map[string]struct{}, teamsWithKVWelcome map[string]string) []string {
	var uniqueTeams []string
	// Place all keys into one list
	teamsWithConfigWelcomeKeys := convertStringMapIntoKeySlice(teamsWithConfigWelcome)
	teamIDsWithKVWelcomeKeys := convertStringMapIntoKeySlice(teamsWithKVWelcome)

	// Convert the ids into team names before combining into 1 large list
	teamsWithKVWelcomeKeys := []string{}
	for _, id := range teamIDsWithKVWelcomeKeys {
		team, err := p.API.GetTeam(id)
		if err != nil {
			continue
		}
		teamsWithKVWelcomeKeys = append(teamsWithKVWelcomeKeys, team.Name)
	}
	allTeamNames := append(teamsWithConfigWelcomeKeys, teamsWithKVWelcomeKeys...)

	// Leverage the unique priniciple of keys in a map to store unique values as they are encountered
	checkMap := make(map[string]int)
	for _, teamName := range allTeamNames {
		checkMap[teamName] = 0
	}

	// Iterate through each pair in the checkMap to create a list of all unique pairs.
	for teamName := range checkMap {
		uniqueTeams = append(uniqueTeams, teamName)
	}
	return uniqueTeams
}

// Takes maps whose keys are strings, and whose values are anything.
func convertStringMapIntoKeySlice(mapInput interface{}) []string {
	// need to check that input is a map or a slice before continuing
	reflectValue := reflect.ValueOf(mapInput)
	switch reflectValue.Kind() {
	case reflect.Map:
	default:
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
	welcomebot.AddCommand(getChannelWelcome)

	deleteChannelWelcome := model.NewAutocompleteData("delete_channel_welcome", "", "Delete the welcome message for the channel")
	welcomebot.AddCommand(deleteChannelWelcome)

	setTeamWelcome := model.NewAutocompleteData(commandTriggerSetTeamWelcome, "[welcome-message]", "Set the welcome message for the team")
	setChannelWelcome.AddTextArgument("Welcome message for the current team", "[welcome-message]", "")
	welcomebot.AddCommand(setTeamWelcome)

	getTeamWelcome := model.NewAutocompleteData(commandTriggerGetTeamWelcome, "", "Print the welcome message for the team")
	welcomebot.AddCommand(getTeamWelcome)

	deleteTeamWelcome := model.NewAutocompleteData(commandTriggerDeleteTeamWelcome, "", "Delete the welcome message for the team. Configuration based messages are not affected by this.")
	welcomebot.AddCommand(deleteTeamWelcome)

	return welcomebot
}
