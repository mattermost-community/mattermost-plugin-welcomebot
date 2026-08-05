import {ACTION_TYPE_AUTOMATIC, ACTION_TYPE_BUTTON, emptyAction, emptyMessage} from './types';
import type {ConfigMessage} from './types';

export type MessageTemplateKey = 'blank' | 'all-users' | 'specific-team';

export type MessageTemplate = {
    key: MessageTemplateKey;
    label: string;
    description: string;
    build: () => ConfigMessage;
};

// Quick-start presets offered from the "Add a welcome message" row, so a
// first-time admin has a working starting point rather than a blank form.
// Every preset still requires picking a team below (this plugin's
// WelcomeMessages are always team-scoped) - "all new users" means a
// generic, team-agnostic message copy, not a cross-team match.
//
// Template variables used below ({{.UserDisplayName}}, {{.Team.DisplayName}})
// come from server/message_template.go's MessageTemplate struct - see the
// plugin's README for the full list.
export const MESSAGE_TEMPLATES: MessageTemplate[] = [
    {
        key: 'blank',
        label: 'Blank welcome message',
        description: 'Start from scratch with no pre-filled text or actions.',
        build: emptyMessage,
    },
    {
        key: 'all-users',
        label: 'Welcome message for all new users',
        description: 'A generic, friendly welcome with no team-specific wording - a good default for any team. Automatically adds the new user to Town Square.',
        build: () => ({
            ...emptyMessage(),
            Message: ["Welcome, {{.UserDisplayName}}! We're glad you're here."],
            DelayInSeconds: 5,
            Actions: [{
                ...emptyAction(),
                ActionType: ACTION_TYPE_AUTOMATIC,
                ActionName: 'auto-join-town-square',
                ChannelsAddedTo: ['town-square'],
            }],
        }),
    },
    {
        key: 'specific-team',
        label: 'Welcome message for members of a specific team',
        description: 'Greets the new member by team name and offers a button to join a follow-up channel - edit the button text and channel once you\'ve picked a team.',
        build: () => ({
            ...emptyMessage(),
            Message: ["Welcome to {{.Team.DisplayName}}, {{.UserDisplayName}}! We're happy to have you on the team."],
            DelayInSeconds: 5,
            Actions: [{
                ...emptyAction(),
                ActionType: ACTION_TYPE_BUTTON,
                ActionDisplayName: 'Join our announcements channel',
                ActionName: 'join-announcements',
                ActionSuccessfulMessage: ["You're all set - welcome aboard!"],
            }],
        }),
    },
];
