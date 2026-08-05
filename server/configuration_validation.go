package main

import (
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

const (
	maxWelcomeMessages   = 100
	maxActionsPerMessage = 25
	maxChannelsPerAction = 50
	maxDelayInSeconds    = 3600
	maxTextFieldChars    = 10000
	maxActionDisplayName = 128
)

var (
	teamNameRE    = regexp.MustCompile(`^[a-z0-9-]{2,64}$`)
	actionNameRE  = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	channelNameRE = regexp.MustCompile(`^[a-z0-9-]{2,64}$`)
)

// Validate checks a Configuration for structural and format correctness. It
// does not check that referenced teams/channels actually exist on the
// server - that is intentionally left to the runtime hooks (which already
// fail closed via their existing exact-match lookups), keeping Validate a
// pure, side-effect-free function safe to call from tests and from
// OnConfigurationChange alike.
func (c *Configuration) Validate() error {
	if len(c.WelcomeMessages) > maxWelcomeMessages {
		return fmt.Errorf("too many welcome messages: got %d, max %d", len(c.WelcomeMessages), maxWelcomeMessages)
	}

	var errs []error
	for i, msg := range c.WelcomeMessages {
		if err := validateConfigMessage(msg); err != nil {
			errs = append(errs, fmt.Errorf("welcome message #%d: %w", i+1, err))
		}
	}
	return errors.Join(errs...)
}

func validateConfigMessage(msg *ConfigMessage) error {
	if !teamNameRE.MatchString(msg.TeamName) {
		return fmt.Errorf("invalid team name %q: must match %s", msg.TeamName, teamNameRE.String())
	}

	if msg.DelayInSeconds < 0 || msg.DelayInSeconds > maxDelayInSeconds {
		return fmt.Errorf("DelayInSeconds must be between 0 and %d, got %d", maxDelayInSeconds, msg.DelayInSeconds)
	}

	var errs []error

	if err := validateTextField("Message", msg.Message); err != nil {
		errs = append(errs, err)
	}
	if err := validateTextField("AttachmentMessage", msg.AttachmentMessage); err != nil {
		errs = append(errs, err)
	}

	if len(msg.Actions) > maxActionsPerMessage {
		errs = append(errs, fmt.Errorf("too many actions: got %d, max %d", len(msg.Actions), maxActionsPerMessage))
	} else {
		seenActionNames := make(map[string]bool, len(msg.Actions))
		for i, action := range msg.Actions {
			if err := validateConfigMessageAction(action); err != nil {
				errs = append(errs, fmt.Errorf("action #%d: %w", i+1, err))
				continue
			}
			if seenActionNames[action.ActionName] {
				errs = append(errs, fmt.Errorf("action #%d: duplicate action name %q within this message", i+1, action.ActionName))
				continue
			}
			seenActionNames[action.ActionName] = true
		}
	}

	return errors.Join(errs...)
}

func validateConfigMessageAction(action *ConfigMessageAction) error {
	var errs []error

	switch action.ActionType {
	case actionTypeAutomatic, actionTypeButton:
	default:
		errs = append(errs, fmt.Errorf("invalid ActionType %q: must be %q or %q", action.ActionType, actionTypeAutomatic, actionTypeButton))
	}

	if !actionNameRE.MatchString(action.ActionName) {
		errs = append(errs, fmt.Errorf("invalid action name %q: must match %s", action.ActionName, actionNameRE.String()))
	}

	if action.ActionType == actionTypeButton {
		if action.ActionDisplayName == "" {
			errs = append(errs, fmt.Errorf("ActionDisplayName is required when ActionType is %q", actionTypeButton))
		} else if len(action.ActionDisplayName) > maxActionDisplayName {
			errs = append(errs, fmt.Errorf("ActionDisplayName is too long: %d characters, max %d", len(action.ActionDisplayName), maxActionDisplayName))
		}
	}

	if err := validateTextField("ActionSuccessfulMessage", action.ActionSuccessfulMessage); err != nil {
		errs = append(errs, err)
	}

	if len(action.ChannelsAddedTo) > maxChannelsPerAction {
		errs = append(errs, fmt.Errorf("too many channels: got %d, max %d", len(action.ChannelsAddedTo), maxChannelsPerAction))
	} else {
		for _, ch := range action.ChannelsAddedTo {
			if !channelNameRE.MatchString(ch) {
				errs = append(errs, fmt.Errorf("invalid channel name %q: must match %s", ch, channelNameRE.String()))
				break
			}
		}
	}

	return errors.Join(errs...)
}

// validateTextField checks a joined-lines text field (the same "\n"-joined
// shape welcomebot.go uses at render time) for length and, since these
// fields are rendered as Go templates, template syntax validity.
func validateTextField(name string, lines []string) error {
	joined := strings.Join(lines, "\n")

	if len(joined) > maxTextFieldChars {
		return fmt.Errorf("%s is too long: %d characters, max %d", name, len(joined), maxTextFieldChars)
	}

	if _, err := template.New(name).Parse(joined); err != nil {
		return fmt.Errorf("%s contains invalid template syntax: %w", name, err)
	}

	return nil
}
