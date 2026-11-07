package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// fakeAPI implements only the plugin.API methods used by the paths under test.
// The embedded interface keeps the type satisfying plugin.API without writing
// stubs for every method; calling an unimplemented method panics, which is the
// desired failure mode for an unexpected dependency.
type fakeAPI struct {
	plugin.API

	users    map[string]*model.User
	teams    map[string]*model.Team
	members  map[string]*model.TeamMember
	channels map[string]*model.Channel

	ephemeral []*model.Post
	joined    []string
	created   []*model.Post
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		users:    map[string]*model.User{},
		teams:    map[string]*model.Team{},
		members:  map[string]*model.TeamMember{},
		channels: map[string]*model.Channel{},
	}
}

func notFound(where string) *model.AppError {
	return model.NewAppError(where, "not_found", nil, "", http.StatusNotFound)
}

func (f *fakeAPI) GetUser(userID string) (*model.User, *model.AppError) {
	if user, ok := f.users[userID]; ok {
		return user, nil
	}
	return nil, notFound("GetUser")
}

func (f *fakeAPI) GetTeam(teamID string) (*model.Team, *model.AppError) {
	for _, team := range f.teams {
		if team.Id == teamID {
			return team, nil
		}
	}
	return nil, notFound("GetTeam")
}

func (f *fakeAPI) GetTeamByName(name string) (*model.Team, *model.AppError) {
	if team, ok := f.teams[name]; ok {
		return team, nil
	}
	return nil, notFound("GetTeamByName")
}

func (f *fakeAPI) GetTeamMember(teamID, userID string) (*model.TeamMember, *model.AppError) {
	if member, ok := f.members[teamID+"/"+userID]; ok {
		return member, nil
	}
	return nil, notFound("GetTeamMember")
}

func (f *fakeAPI) GetChannel(channelID string) (*model.Channel, *model.AppError) {
	if channel, ok := f.channels[channelID]; ok {
		return channel, nil
	}
	return nil, notFound("GetChannel")
}

func (f *fakeAPI) GetChannelByName(teamID, name string, _ bool) (*model.Channel, *model.AppError) {
	for _, channel := range f.channels {
		if channel.TeamId == teamID && channel.Name == name {
			return channel, nil
		}
	}
	return nil, notFound("GetChannelByName")
}

func (f *fakeAPI) GetDirectChannel(userID1, _ string) (*model.Channel, *model.AppError) {
	return &model.Channel{Id: "dm-" + userID1, Type: model.ChannelTypeDirect}, nil
}

func (f *fakeAPI) GetConfig() *model.Config {
	return nil
}

func (f *fakeAPI) KVGet(_ string) ([]byte, *model.AppError) {
	return nil, nil
}

func (f *fakeAPI) HasPermissionTo(_ string, _ *model.Permission) bool {
	return false
}

func (f *fakeAPI) SendEphemeralPost(userID string, post *model.Post) *model.Post {
	f.ephemeral = append(f.ephemeral, post)
	return post
}

func (f *fakeAPI) AddChannelMember(channelID, userID string) (*model.ChannelMember, *model.AppError) {
	f.joined = append(f.joined, channelID+"/"+userID)
	return &model.ChannelMember{ChannelId: channelID, UserId: userID}, nil
}

func (f *fakeAPI) CreatePost(post *model.Post) (*model.Post, *model.AppError) {
	f.created = append(f.created, post)
	return post, nil
}

func (f *fakeAPI) LogError(string, ...any) {}
func (f *fakeAPI) LogWarn(string, ...any)  {}
func (f *fakeAPI) LogDebug(string, ...any) {}

func (f *fakeAPI) ephemeralMessages() []string {
	messages := make([]string, 0, len(f.ephemeral))
	for _, post := range f.ephemeral {
		messages = append(messages, post.Message)
	}
	return messages
}

func newTestPlugin(f *fakeAPI, messages ...*ConfigMessage) *Plugin {
	p := &Plugin{botUserID: "bot"}
	p.API = f
	p.welcomeMessages.Store(messages)
	return p
}

func seedTeam(f *fakeAPI) {
	team := &model.Team{Id: "teamA", Name: "team-a", DisplayName: "Team A"}
	f.teams[team.Name] = team
	f.channels["chanTown"] = &model.Channel{
		Id: "chanTown", TeamId: team.Id, Name: "town-square",
		DisplayName: "Town Square", Type: model.ChannelTypeOpen,
	}
	f.channels["chanSecret"] = &model.Channel{
		Id: "chanSecret", TeamId: team.Id, Name: "secret-onboarding",
		DisplayName: "Secret Onboarding", Type: model.ChannelTypeOpen,
	}
}

func seedUser(f *fakeAPI, user *model.User, member bool) {
	f.users[user.Id] = user
	if member {
		f.members["teamA/"+user.Id] = &model.TeamMember{
			TeamId: "teamA", UserId: user.Id,
		}
	}
}

func TestPreviewRequiresTeamMembership(t *testing.T) {
	f := newFakeAPI()
	seedTeam(f)
	seedUser(f, &model.User{Id: "stranger1", Username: "stranger1"}, false)
	seedUser(f, &model.User{Id: "member1", Username: "member1"}, true)

	message := &ConfigMessage{
		TeamName:      "team-a",
		Message:       []string{"Hello {{.UserDisplayName}}"},
		IncludeGuests: true,
		Actions: []*ConfigMessageAction{{
			ActionType:      actionTypeAutomatic,
			ChannelsAddedTo: []string{"secret-onboarding"},
		}},
	}
	p := newTestPlugin(f, message)

	args := &model.CommandArgs{Command: "/welcomebot preview team-a", UserId: "stranger1", ChannelId: "chanTown"}
	if _, appErr := p.ExecuteCommand(nil, args); appErr != nil {
		t.Fatalf("ExecuteCommand returned an error: %v", appErr)
	}

	for _, text := range f.ephemeralMessages() {
		if strings.Contains(text, "Hello") {
			t.Fatalf("non-member received welcome content: %q", text)
		}
	}
	if len(f.joined) != 0 {
		t.Fatalf("non-member was added to channels: %v", f.joined)
	}
	if len(f.ephemeral) == 0 {
		t.Fatal("non-member received no response at all")
	}
}

func TestPreviewAppliesGuestPolicy(t *testing.T) {
	f := newFakeAPI()
	seedTeam(f)
	seedUser(f, &model.User{Id: "guest1", Username: "guest1", Roles: model.SystemGuestRoleId}, true)

	message := &ConfigMessage{
		TeamName:      "team-a",
		Message:       []string{"Hello {{.UserDisplayName}}"},
		IncludeGuests: false,
		Actions: []*ConfigMessageAction{{
			ActionType:      actionTypeAutomatic,
			ChannelsAddedTo: []string{"secret-onboarding"},
		}},
	}
	p := newTestPlugin(f, message)

	args := &model.CommandArgs{Command: "/welcomebot preview team-a", UserId: "guest1", ChannelId: "chanTown"}
	if _, appErr := p.ExecuteCommand(nil, args); appErr != nil {
		t.Fatalf("ExecuteCommand returned an error: %v", appErr)
	}

	for _, text := range f.ephemeralMessages() {
		if strings.Contains(text, "Hello") {
			t.Fatalf("excluded guest received welcome content: %q", text)
		}
	}
	if len(f.joined) != 0 {
		t.Fatalf("excluded guest was added to channels: %v", f.joined)
	}
}

func TestPreviewNeverPerformsAutomaticActions(t *testing.T) {
	f := newFakeAPI()
	seedTeam(f)
	seedUser(f, &model.User{Id: "member1", Username: "member1"}, true)

	message := &ConfigMessage{
		TeamName:      "team-a",
		Message:       []string{"Hello {{.UserDisplayName}}"},
		IncludeGuests: true,
		Actions: []*ConfigMessageAction{{
			ActionType:      actionTypeAutomatic,
			ChannelsAddedTo: []string{"secret-onboarding"},
		}},
	}
	p := newTestPlugin(f, message)

	args := &model.CommandArgs{Command: "/welcomebot preview team-a", UserId: "member1", ChannelId: "chanTown"}
	if _, appErr := p.ExecuteCommand(nil, args); appErr != nil {
		t.Fatalf("ExecuteCommand returned an error: %v", appErr)
	}

	found := false
	for _, text := range f.ephemeralMessages() {
		if strings.Contains(text, "Hello member1") {
			found = true
		}
	}
	if !found {
		t.Fatalf("member did not receive the preview: %v", f.ephemeralMessages())
	}
	if len(f.joined) != 0 {
		t.Fatalf("preview performed automatic joins: %v", f.joined)
	}
}

func TestAddChannelsAppliesGuestPolicy(t *testing.T) {
	f := newFakeAPI()
	seedTeam(f)
	seedUser(f, &model.User{Id: "guest1", Username: "guest1", Roles: model.SystemGuestRoleId}, true)
	seedUser(f, &model.User{Id: "member1", Username: "member1"}, true)

	message := &ConfigMessage{
		TeamName:      "team-a",
		IncludeGuests: false,
		Actions: []*ConfigMessageAction{{
			ActionType:              actionTypeAutomatic,
			ChannelsAddedTo:         []string{"secret-onboarding"},
			ActionSuccessfulMessage: []string{"You joined"},
		}},
	}
	p := newTestPlugin(f, message)

	post := func(userID string) *httptest.ResponseRecorder {
		body := `{"context":{"team_id":"teamA","user_id":"` + userID + `","action":""}}`
		req := httptest.NewRequest(http.MethodPost, "/addchannels", strings.NewReader(body))
		req.Header.Set("Mattermost-User-Id", userID)
		w := httptest.NewRecorder()
		p.ServeHTTP(nil, w, req)
		return w
	}

	if w := post("guest1"); w.Code != http.StatusOK {
		t.Fatalf("guest request returned %d", w.Code)
	}
	if len(f.joined) != 0 {
		t.Fatalf("excluded guest was added to channels: %v", f.joined)
	}

	if w := post("member1"); w.Code != http.StatusOK {
		t.Fatalf("member request returned %d", w.Code)
	}
	if len(f.joined) != 1 || f.joined[0] != "chanSecret/member1" {
		t.Fatalf("member was not added as expected: %v", f.joined)
	}
}

func TestEscapeMarkdownNeutralizesLinks(t *testing.T) {
	input := "[Verify](https://evil.example)"
	want := `\[Verify\]\(https://evil\.example\)`
	if got := escapeMarkdown(input); got != want {
		t.Fatalf("escapeMarkdown(%q) = %q, want %q", input, got, want)
	}
}

func TestSanitizedTemplateDataCannotCreateMarkdownLinks(t *testing.T) {
	team := sanitizeTeam(&model.Team{
		Id:          "teamA",
		Name:        "team-a",
		DisplayName: "[Verify](https://evil.example)",
	})

	rendered, err := renderTemplate("test", []string{"Welcome to {{.Team.DisplayName}}!"}, MessageTemplate{Team: team})
	if err != nil {
		t.Fatalf("renderTemplate returned an error: %v", err)
	}
	if strings.Contains(rendered, "[Verify](") {
		t.Fatalf("rendered message still contains an active Markdown link: %q", rendered)
	}
	if !strings.Contains(rendered, `\[Verify\]`) {
		t.Fatalf("rendered message did not contain the escaped value: %q", rendered)
	}
}

func TestProcessActionMessageDoesNotPanicOnInvalidTemplate(t *testing.T) {
	f := newFakeAPI()
	p := newTestPlugin(f)

	action := &Action{Context: &ActionContext{TeamID: "teamA", UserID: "member1"}}
	configAction := &ConfigMessageAction{
		ActionSuccessfulMessage: []string{"{{"},
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("processActionMessage panicked on an invalid template: %v", recovered)
		}
	}()

	p.processActionMessage(MessageTemplate{User: &model.User{Id: "member1"}}, action, *configAction)
}