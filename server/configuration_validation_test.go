package main

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func validAction() *ConfigMessageAction {
	return &ConfigMessageAction{
		ActionType:              actionTypeButton,
		ActionDisplayName:       "Join Engineering",
		ActionName:              "join-eng",
		ActionSuccessfulMessage: []string{"You joined!"},
		ChannelsAddedTo:         []string{"town-square"},
	}
}

func validMessage() *ConfigMessage {
	return &ConfigMessage{
		TeamName:          "engineering",
		Actions:           []*ConfigMessageAction{validAction()},
		Message:           []string{"Welcome {{.User.Username}}!"},
		AttachmentMessage: []string{"Glad to have you."},
		DelayInSeconds:    5,
		IncludeGuests:     false,
	}
}

func TestValidate_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config Configuration
	}{
		{
			name:   "empty welcome messages",
			config: Configuration{WelcomeMessages: nil},
		},
		{
			name:   "one message, no actions",
			config: Configuration{WelcomeMessages: []*ConfigMessage{{TeamName: "engineering", DelayInSeconds: 0}}},
		},
		{
			name: "one message with automatic and button actions",
			config: Configuration{WelcomeMessages: []*ConfigMessage{
				{
					TeamName: "engineering",
					Actions: []*ConfigMessageAction{
						{ActionType: actionTypeAutomatic, ActionName: "auto-join", ChannelsAddedTo: []string{"town-square"}},
						validAction(),
					},
				},
			}},
		},
		{
			name:   "message at exactly the delay boundary",
			config: Configuration{WelcomeMessages: []*ConfigMessage{{TeamName: "engineering", DelayInSeconds: maxDelayInSeconds}}},
		},
		{
			name:   "team name at exactly the minimum length boundary",
			config: Configuration{WelcomeMessages: []*ConfigMessage{{TeamName: "ab"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.config.Validate())
		})
	}
}

func TestValidate_OverLimitArrays(t *testing.T) {
	t.Run("too many welcome messages", func(t *testing.T) {
		messages := make([]*ConfigMessage, maxWelcomeMessages+1)
		for i := range messages {
			messages[i] = &ConfigMessage{TeamName: "engineering"}
		}
		err := (&Configuration{WelcomeMessages: messages}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many welcome messages")
	})

	t.Run("too many actions in one message", func(t *testing.T) {
		actions := make([]*ConfigMessageAction, maxActionsPerMessage+1)
		for i := range actions {
			actions[i] = &ConfigMessageAction{ActionType: actionTypeAutomatic, ActionName: "a"}
		}
		msg := &ConfigMessage{TeamName: "engineering", Actions: actions}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many actions")
	})

	t.Run("too many channels on one action", func(t *testing.T) {
		channels := make([]string, maxChannelsPerAction+1)
		for i := range channels {
			channels[i] = "town-square"
		}
		action := &ConfigMessageAction{ActionType: actionTypeAutomatic, ActionName: "auto-join", ChannelsAddedTo: channels}
		msg := &ConfigMessage{TeamName: "engineering", Actions: []*ConfigMessageAction{action}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many channels")
	})
}

func TestValidate_RegexFormats(t *testing.T) {
	tests := []struct {
		name     string
		teamName string
		wantErr  bool
	}{
		{name: "valid lowercase handle", teamName: "engineering", wantErr: false},
		{name: "valid at minimum length boundary", teamName: "ab", wantErr: false},
		{name: "invalid below minimum length", teamName: "a", wantErr: true},
		{name: "invalid uppercase", teamName: "Engineering", wantErr: true},
		{name: "invalid spaces", teamName: "engineering team", wantErr: true},
		{name: "invalid empty", teamName: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ConfigMessage{TeamName: tt.teamName}
			err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("invalid action name characters rejected", func(t *testing.T) {
		action := &ConfigMessageAction{ActionType: actionTypeAutomatic, ActionName: "join eng!"}
		msg := &ConfigMessage{TeamName: "engineering", Actions: []*ConfigMessageAction{action}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid action name")
	})

	t.Run("invalid channel handle rejected", func(t *testing.T) {
		action := &ConfigMessageAction{ActionType: actionTypeAutomatic, ActionName: "auto-join", ChannelsAddedTo: []string{"Town Square"}}
		msg := &ConfigMessage{TeamName: "engineering", Actions: []*ConfigMessageAction{action}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid channel name")
	})
}

func TestValidate_MalformedTemplate(t *testing.T) {
	t.Run("unclosed action rejected", func(t *testing.T) {
		msg := &ConfigMessage{TeamName: "engineering", Message: []string{"Hello {{.User.Username"}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid template syntax")
	})

	t.Run("syntactically valid but unknown field is not rejected", func(t *testing.T) {
		// html/template.Parse only checks syntax, not field existence -
		// that's a runtime Execute-time concern already handled gracefully
		// by welcomebot.go's existing render path, out of scope here.
		msg := &ConfigMessage{TeamName: "engineering", Message: []string{"Hello {{.NotAField}}"}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		assert.NoError(t, err)
	})

	t.Run("malformed AttachmentMessage rejected", func(t *testing.T) {
		msg := &ConfigMessage{TeamName: "engineering", AttachmentMessage: []string{"{{if}}"}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AttachmentMessage")
	})

	t.Run("text field over max length rejected", func(t *testing.T) {
		msg := &ConfigMessage{TeamName: "engineering", Message: []string{strings.Repeat("a", maxTextFieldChars+1)}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
}

func TestValidate_DuplicateActionName(t *testing.T) {
	t.Run("duplicate within one message rejected", func(t *testing.T) {
		msg := &ConfigMessage{
			TeamName: "engineering",
			Actions: []*ConfigMessageAction{
				{ActionType: actionTypeAutomatic, ActionName: "join-eng"},
				{ActionType: actionTypeAutomatic, ActionName: "join-eng"},
			},
		}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate action name")
	})

	t.Run("same name reused across different messages allowed", func(t *testing.T) {
		config := Configuration{WelcomeMessages: []*ConfigMessage{
			{TeamName: "engineering", Actions: []*ConfigMessageAction{{ActionType: actionTypeAutomatic, ActionName: "join"}}},
			{TeamName: "sales", Actions: []*ConfigMessageAction{{ActionType: actionTypeAutomatic, ActionName: "join"}}},
		}}
		assert.NoError(t, config.Validate())
	})
}

func TestValidate_ActionType(t *testing.T) {
	tests := []struct {
		name       string
		actionType string
		wantErr    bool
	}{
		{name: "automatic accepted", actionType: actionTypeAutomatic, wantErr: false},
		{name: "button accepted with display name", actionType: actionTypeButton, wantErr: false},
		{name: "invalid value rejected", actionType: "maybe", wantErr: true},
		{name: "empty value rejected", actionType: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &ConfigMessageAction{ActionType: tt.actionType, ActionName: "join"}
			if tt.actionType == actionTypeButton {
				action.ActionDisplayName = "Join"
			}
			msg := &ConfigMessage{TeamName: "engineering", Actions: []*ConfigMessageAction{action}}
			err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("button action without display name rejected", func(t *testing.T) {
		action := &ConfigMessageAction{ActionType: actionTypeButton, ActionName: "join"}
		msg := &ConfigMessage{TeamName: "engineering", Actions: []*ConfigMessageAction{action}}
		err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ActionDisplayName is required")
	})
}

func TestValidate_DelayInSecondsBounds(t *testing.T) {
	tests := []struct {
		name    string
		delay   int
		wantErr bool
	}{
		{name: "negative rejected", delay: -1, wantErr: true},
		{name: "zero accepted", delay: 0, wantErr: false},
		{name: "at max boundary accepted", delay: maxDelayInSeconds, wantErr: false},
		{name: "over max boundary rejected", delay: maxDelayInSeconds + 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ConfigMessage{TeamName: "engineering", DelayInSeconds: tt.delay}
			err := (&Configuration{WelcomeMessages: []*ConfigMessage{msg}}).Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOnConfigurationChange_RejectsInvalidKeepsLastGood proves the
// defense-in-depth backstop in OnConfigurationChange: an invalid config
// is rejected without overwriting the atomic.Value, so hooks keep reading
// the last-known-good WelcomeMessages rather than regressing.
func TestOnConfigurationChange_RejectsInvalidKeepsLastGood(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := &Plugin{}
	p.SetAPI(api)
	p.welcomeMessages.Store([]*ConfigMessage{})

	goodConfig := Configuration{WelcomeMessages: []*ConfigMessage{validMessage()}}
	api.On("LoadPluginConfiguration", mock.Anything).Once().Run(func(args mock.Arguments) {
		dest := args.Get(0).(*Configuration)
		*dest = goodConfig
	}).Return(nil)
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Return()

	require.NoError(t, p.OnConfigurationChange())
	require.Len(t, p.getWelcomeMessages(), 1)
	require.Equal(t, "engineering", p.getWelcomeMessages()[0].TeamName)

	badConfig := Configuration{WelcomeMessages: []*ConfigMessage{{TeamName: "Not A Valid Handle"}}}
	api.On("LoadPluginConfiguration", mock.Anything).Once().Run(func(args mock.Arguments) {
		dest := args.Get(0).(*Configuration)
		*dest = badConfig
	}).Return(nil)
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	require.Error(t, p.OnConfigurationChange())

	// Still the good config from the first call - not overwritten, and not
	// reverted to empty either.
	require.Len(t, p.getWelcomeMessages(), 1)
	require.Equal(t, "engineering", p.getWelcomeMessages()[0].TeamName)
}
