package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testBotUserID = "botuserid00000000000000000"
	testTeamID    = "teamid0000000000000000000"
)

// TestMain disables the pending-broadcast expiry timer by default so tests that create a pending
// broadcast don't fire a stray timer (which would hit unmocked API calls in the background). The
// expiry test re-enables it with a short duration.
func TestMain(m *testing.M) {
	broadcastPendingExpiry = time.Hour
	os.Exit(m.Run())
}

// mockLogging registers permissive expectations for the logging methods so tests don't have to
// enumerate every log call.
func mockLogging(api *plugintest.API) {
	for _, method := range []string{"LogError", "LogWarn", "LogInfo", "LogDebug"} {
		for n := 1; n <= 9; n += 2 {
			args := make([]interface{}, n)
			for i := range args {
				args[i] = mock.Anything
			}
			api.On(method, args...).Maybe()
		}
	}
}

func newTestPlugin(api *plugintest.API) *Plugin {
	p := &Plugin{botUserID: testBotUserID}
	p.SetAPI(api)
	return p
}

func newRequest(userID, team, body string) *http.Request {
	url := "/sendmessagetoteam"
	if team != "" {
		url += "?team=" + team
	}
	r := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if userID != "" {
		r.Header.Set("Mattermost-User-Id", userID)
	}
	return r
}

func TestHandleSendMessageToTeam_NotAuthenticated(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("", "myteam", "hello"))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleSendMessageToTeam_MissingTeamParam(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("user1", "", "hello"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSendMessageToTeam_TeamNotFound(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	api.On("GetTeam", "myteam").Return(nil, model.NewAppError("GetTeam", "not found", nil, "", http.StatusNotFound))
	api.On("GetTeamByName", "myteam").Return(nil, model.NewAppError("GetTeamByName", "not found", nil, "", http.StatusNotFound))
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("user1", "myteam", "hello"))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSendMessageToTeam_NotAuthorized(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetTeam", "myteam").Return(team, nil)
	// Not a system admin.
	api.On("GetUser", "user1").Return(&model.User{Id: "user1", Roles: "system_user"}, nil)
	// Not a team admin either.
	api.On("GetTeamMember", testTeamID, "user1").Return(&model.TeamMember{TeamId: testTeamID, UserId: "user1", SchemeAdmin: false}, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("user1", "myteam", "hello"))

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSendMessageToTeam_EmptyBody(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetTeam", "myteam").Return(team, nil)
	api.On("GetUser", "admin1").Return(&model.User{Id: "admin1", Roles: "system_user system_admin"}, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("admin1", "myteam", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSendMessageToTeam_TeamAdminBroadcasts(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	// Resolved by name (GetTeam fails, GetTeamByName succeeds).
	api.On("GetTeam", "myteam").Return(nil, model.NewAppError("GetTeam", "not found", nil, "", http.StatusNotFound))
	api.On("GetTeamByName", "myteam").Return(team, nil)
	// Not a system admin, but an admin of this team.
	api.On("GetUser", "teamadmin1").Return(&model.User{Id: "teamadmin1", Roles: "system_user"}, nil)
	api.On("GetTeamMember", testTeamID, "teamadmin1").Return(&model.TeamMember{TeamId: testTeamID, UserId: "teamadmin1", SchemeAdmin: true}, nil)

	users := []*model.User{{Id: "u1"}}
	api.On("GetUsersInTeam", testTeamID, 0, sendMessageToTeamPageSize).Return(users, nil)
	for _, uid := range []string{"u1", "teamadmin1"} {
		api.On("GetDirectChannel", uid, testBotUserID).Return(&model.Channel{Id: "dm_" + uid}, nil)
	}

	createdPosts := make(chan *model.Post, 2)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post {
			createdPosts <- post
			return post
		},
		func(post *model.Post) *model.AppError { return nil },
	)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("teamadmin1", "myteam", "Hello team"))

	assert.Equal(t, http.StatusAccepted, w.Code)

	channels := map[string]string{}
	for i := 0; i < 2; i++ {
		select {
		case post := <-createdPosts:
			channels[post.ChannelId] = post.Message
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for broadcast posts, got %d of 2", len(channels))
		}
	}

	assert.Equal(t, "Hello team", channels["dm_u1"])
	assert.Contains(t, channels["dm_teamadmin1"], "My Team")

	api.AssertExpectations(t)
}

func TestHandleSendMessageToTeam_Pagination(t *testing.T) {
	// Shrink the page size and drop the delay so the test exercises multiple pages quickly.
	origPageSize, origDelay := sendMessageToTeamPageSize, sendMessageToTeamDelay
	sendMessageToTeamPageSize, sendMessageToTeamDelay = 2, 0
	defer func() {
		sendMessageToTeamPageSize, sendMessageToTeamDelay = origPageSize, origDelay
	}()

	api := &plugintest.API{}
	mockLogging(api)

	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetTeam", "myteam").Return(team, nil)
	api.On("GetUser", "admin1").Return(&model.User{Id: "admin1", Roles: "system_admin"}, nil)

	// Page 0: full page (== page size) forces another lookup. Page 1: partial page ends the loop.
	api.On("GetUsersInTeam", testTeamID, 0, 2).Return([]*model.User{{Id: "u1"}, {Id: "u2"}}, nil)
	api.On("GetUsersInTeam", testTeamID, 1, 2).Return([]*model.User{{Id: "u3"}}, nil)

	for _, uid := range []string{"u1", "u2", "u3", "admin1"} {
		api.On("GetDirectChannel", uid, testBotUserID).Return(&model.Channel{Id: "dm_" + uid}, nil)
	}

	createdPosts := make(chan *model.Post, 4)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post {
			createdPosts <- post
			return post
		},
		func(post *model.Post) *model.AppError { return nil },
	)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("admin1", "myteam", "Hello team"))

	assert.Equal(t, http.StatusAccepted, w.Code)

	// 3 recipients across two pages + 1 initiator notification.
	channels := map[string]string{}
	for i := 0; i < 4; i++ {
		select {
		case post := <-createdPosts:
			channels[post.ChannelId] = post.Message
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for broadcast posts, got %d of 4", len(channels))
		}
	}

	for _, uid := range []string{"u1", "u2", "u3"} {
		assert.Equal(t, "Hello team", channels["dm_"+uid])
	}
	assert.Contains(t, channels["dm_admin1"], "My Team")

	// Both pages must have been requested.
	api.AssertCalled(t, "GetUsersInTeam", testTeamID, 0, 2)
	api.AssertCalled(t, "GetUsersInTeam", testTeamID, 1, 2)
	api.AssertExpectations(t)
}

func TestParseBroadcastCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantTeam    string
		wantMessage string
	}{
		{"team and single-line message", "!broadcast myteam Hello everyone", "myteam", "Hello everyone"},
		{"team and multi-line message", "!broadcast myteam\nHello\neveryone", "myteam", "Hello\neveryone"},
		{"extra spaces", "!broadcast   myteam    Hello", "myteam", "Hello"},
		{"only team, no message", "!broadcast myteam", "myteam", ""},
		{"prefix only", "!broadcast", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team, message := parseBroadcastCommand(tt.command)
			assert.Equal(t, tt.wantTeam, team)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}

func TestMessageHasBeenPosted_IgnoresBotOwnMessage(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: testBotUserID, ChannelId: "dm", Message: "!broadcast myteam hi"})
	// No API calls expected: AssertExpectations covers that nothing unexpected ran.
}

func TestMessageHasBeenPosted_IgnoresNonBroadcast(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "user1", ChannelId: "dm", Message: "just a normal message"})
}

func TestMessageHasBeenPosted_IgnoresNonDMChannel(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)
	// The bot's DM channel with the user differs from the post's channel, so it must be ignored.
	api.On("GetDirectChannel", "user1", testBotUserID).Return(&model.Channel{Id: "dm_user1"}, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "user1", ChannelId: "some_public_channel", Message: "!broadcast myteam hi"})
}

func TestMessageHasBeenPosted_NonAdminIgnoredSilently(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	api.On("GetDirectChannel", "user1", testBotUserID).Return(&model.Channel{Id: "dm_user1"}, nil)
	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetTeam", "myteam").Return(team, nil)
	// Neither a system admin nor an admin of the team.
	api.On("GetUser", "user1").Return(&model.User{Id: "user1", Roles: "system_user"}, nil)
	api.On("GetTeamMember", testTeamID, "user1").Return(&model.TeamMember{TeamId: testTeamID, UserId: "user1", SchemeAdmin: false}, nil)

	// A non-admin must get no reply at all: CreatePost is intentionally not registered, so any call
	// would fail the mock.
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "user1", ChannelId: "dm_user1", Message: "!broadcast myteam hi"})

	// Give any (erroneous) background reply a chance to fire before asserting expectations.
	time.Sleep(50 * time.Millisecond)
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func recvPost(t *testing.T, posts <-chan *model.Post) *model.Post {
	t.Helper()
	select {
	case post := <-posts:
		return post
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a post")
		return nil
	}
}

func TestMessageHasBeenPosted_RequestAsksForConfirmation(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)
	api.On("GetTeam", "myteam").Return(team, nil)
	api.On("GetUser", "admin1").Return(&model.User{Id: "admin1", Roles: "system_admin"}, nil)
	api.On("GetTeamStats", testTeamID).Return(&model.TeamStats{TeamId: testTeamID, ActiveMemberCount: 6220}, nil)
	api.On("KVSetWithExpiry", broadcastPendingKeyPrefix+"admin1", mock.AnythingOfType("[]uint8"), int64(broadcastPendingKVTTLSeconds)).Return(nil)

	posts := make(chan *model.Post, 1)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post { posts <- post; return post },
		func(post *model.Post) *model.AppError { return nil },
	)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: "!broadcast myteam Hello team"})

	prompt := recvPost(t, posts)
	assert.Equal(t, "dm_admin1", prompt.ChannelId)
	assert.Contains(t, prompt.Message, "6220")
	assert.Contains(t, prompt.Message, "My Team")
	assert.Contains(t, prompt.Message, broadcastConfirmWord)

	// Nothing must be broadcast before confirmation.
	api.AssertNotCalled(t, "GetUsersInTeam", mock.Anything, mock.Anything, mock.Anything)
	api.AssertExpectations(t)
}

func TestMessageHasBeenPosted_ConfirmationRunsBroadcast(t *testing.T) {
	sendMessageToTeamDelay = 0
	defer func() { sendMessageToTeamDelay = 50 * time.Millisecond }()

	api := &plugintest.API{}
	mockLogging(api)

	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)
	api.On("GetTeam", "myteam").Return(team, nil)
	api.On("GetUser", "admin1").Return(&model.User{Id: "admin1", Roles: "system_admin"}, nil)
	api.On("GetTeamStats", testTeamID).Return(&model.TeamStats{TeamId: testTeamID, ActiveMemberCount: 1}, nil)

	// KV round-trip: capture what the request stores and hand it back on confirmation.
	var stored []byte
	key := broadcastPendingKeyPrefix + "admin1"
	api.On("KVSetWithExpiry", key, mock.AnythingOfType("[]uint8"), int64(broadcastPendingKVTTLSeconds)).Return(
		func(_ string, value []byte, _ int64) *model.AppError { stored = value; return nil })
	api.On("KVGet", key).Return(
		func(string) []byte { return stored },
		func(string) *model.AppError { return nil },
	)
	api.On("KVDelete", key).Return(nil)

	api.On("GetUsersInTeam", testTeamID, 0, sendMessageToTeamPageSize).Return([]*model.User{{Id: "u1"}}, nil)
	api.On("GetDirectChannel", "u1", testBotUserID).Return(&model.Channel{Id: "dm_u1"}, nil)

	posts := make(chan *model.Post, 4)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post { posts <- post; return post },
		func(post *model.Post) *model.AppError { return nil },
	)

	p := newTestPlugin(api)

	// Step 1: request stores the pending broadcast and prompts for confirmation.
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: "!broadcast myteam Hello team"})
	_ = recvPost(t, posts) // confirmation prompt
	assert.NotNil(t, stored)

	// Step 2: confirmation runs the broadcast.
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: "Да"})

	// Expected posts: ack to initiator, broadcast to u1, completion report to initiator.
	byChannel := map[string][]string{}
	for i := 0; i < 3; i++ {
		post := recvPost(t, posts)
		byChannel[post.ChannelId] = append(byChannel[post.ChannelId], post.Message)
	}

	assert.Equal(t, []string{"Hello team"}, byChannel["dm_u1"])
	assert.Len(t, byChannel["dm_admin1"], 2) // ack + completion report

	api.AssertExpectations(t)
}

func TestMessageHasBeenPosted_CancellationDiscardsPending(t *testing.T) {
	for _, word := range []string{"нет", "Отмена"} {
		t.Run(word, func(t *testing.T) {
			api := &plugintest.API{}
			mockLogging(api)

			pending := pendingBroadcast{TeamID: testTeamID, TeamDisplayName: "My Team", Message: "Hello team"}
			data, _ := json.Marshal(pending)

			key := broadcastPendingKeyPrefix + "admin1"
			api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)
			api.On("KVGet", key).Return(data, nil)
			api.On("KVDelete", key).Return(nil)

			replies := make(chan *model.Post, 1)
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
				func(post *model.Post) *model.Post { replies <- post; return post },
				func(post *model.Post) *model.AppError { return nil },
			)
			defer api.AssertExpectations(t)

			p := newTestPlugin(api)
			p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: word})

			reply := recvPost(t, replies)
			assert.Equal(t, "dm_admin1", reply.ChannelId)
			assert.Contains(t, reply.Message, "отменена")

			// The pending broadcast must have been consumed and nothing sent.
			api.AssertCalled(t, "KVDelete", key)
			api.AssertNotCalled(t, "GetUsersInTeam", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMessageHasBeenPosted_CancellationWithoutPendingIgnored(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)
	api.On("KVGet", broadcastPendingKeyPrefix+"admin1").Return(nil, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: "нет"})

	// No pending broadcast: a stray "нет" must not produce any reply.
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestMessageHasBeenPosted_ConfirmationWithoutPendingIgnored(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)
	api.On("KVGet", broadcastPendingKeyPrefix+"admin1").Return(nil, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.MessageHasBeenPosted(nil, &model.Post{UserId: "admin1", ChannelId: "dm_admin1", Message: "да"})

	// No pending broadcast: a stray "да" must not produce any reply or broadcast.
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
	api.AssertNotCalled(t, "GetUsersInTeam", mock.Anything, mock.Anything, mock.Anything)
}

func TestExpirePendingBroadcast_NotifiesOnTimeout(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	pending := pendingBroadcast{TeamID: testTeamID, TeamDisplayName: "My Team", Message: "Hello team", Token: "tok1"}
	data, _ := json.Marshal(pending)

	key := broadcastPendingKeyPrefix + "admin1"
	api.On("KVGet", key).Return(data, nil)
	api.On("KVDelete", key).Return(nil)
	api.On("GetDirectChannel", "admin1", testBotUserID).Return(&model.Channel{Id: "dm_admin1"}, nil)

	replies := make(chan *model.Post, 1)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post { replies <- post; return post },
		func(post *model.Post) *model.AppError { return nil },
	)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.expirePendingBroadcast("admin1", "tok1")

	reply := recvPost(t, replies)
	assert.Equal(t, "dm_admin1", reply.ChannelId)
	assert.Contains(t, reply.Message, "истекло")
	assert.Contains(t, reply.Message, "My Team")
	api.AssertCalled(t, "KVDelete", key)
}

func TestExpirePendingBroadcast_IgnoresStaleToken(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	// The pending broadcast has a newer token, so a stale timer must not touch it.
	pending := pendingBroadcast{TeamID: testTeamID, TeamDisplayName: "My Team", Token: "new-token"}
	data, _ := json.Marshal(pending)
	api.On("KVGet", broadcastPendingKeyPrefix+"admin1").Return(data, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.expirePendingBroadcast("admin1", "old-token")

	api.AssertNotCalled(t, "KVDelete", mock.Anything)
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestExpirePendingBroadcast_NoPendingIgnored(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	// Already confirmed or cancelled: nothing to expire.
	api.On("KVGet", broadcastPendingKeyPrefix+"admin1").Return(nil, nil)
	defer api.AssertExpectations(t)

	p := newTestPlugin(api)
	p.expirePendingBroadcast("admin1", "tok1")

	api.AssertNotCalled(t, "KVDelete", mock.Anything)
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestHandleSendMessageToTeam_SystemAdminBroadcasts(t *testing.T) {
	api := &plugintest.API{}
	mockLogging(api)

	team := &model.Team{Id: testTeamID, Name: "myteam", DisplayName: "My Team"}
	api.On("GetTeam", "myteam").Return(team, nil)
	api.On("GetUser", "admin1").Return(&model.User{Id: "admin1", Roles: "system_user system_admin"}, nil)

	// Background broadcast: one page with two users, fewer than the page size, so the loop stops.
	users := []*model.User{{Id: "u1"}, {Id: "u2"}}
	api.On("GetUsersInTeam", testTeamID, 0, sendMessageToTeamPageSize).Return(users, nil)

	// Each recipient plus the initiator gets a direct channel and a post.
	for _, uid := range []string{"u1", "u2", "admin1"} {
		api.On("GetDirectChannel", uid, testBotUserID).Return(&model.Channel{Id: "dm_" + uid}, nil)
	}

	createdPosts := make(chan *model.Post, 3)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(
		func(post *model.Post) *model.Post {
			createdPosts <- post
			return post
		},
		func(post *model.Post) *model.AppError { return nil },
	)

	p := newTestPlugin(api)
	w := httptest.NewRecorder()

	p.handleSendMessageToTeam(w, newRequest("admin1", "myteam", "Hello team"))

	// The handler must answer immediately without waiting for the broadcast.
	assert.Equal(t, http.StatusAccepted, w.Code)

	// The background broadcast should send to both users and notify the initiator.
	posts := make([]*model.Post, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case post := <-createdPosts:
			posts = append(posts, post)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for broadcast posts, got %d of 3", len(posts))
		}
	}

	channels := map[string]string{}
	for _, post := range posts {
		assert.Equal(t, testBotUserID, post.UserId)
		channels[post.ChannelId] = post.Message
	}

	assert.Equal(t, "Hello team", channels["dm_u1"])
	assert.Equal(t, "Hello team", channels["dm_u2"])
	// The initiator receives a summary rather than the broadcast body.
	assert.Contains(t, channels["dm_admin1"], "My Team")

	api.AssertExpectations(t)
}
