import type {ConfigMessage, ConfigMessageAction} from './types';
import {ACTION_TYPE_AUTOMATIC, ACTION_TYPE_BUTTON} from './types';

// These limits mirror server/configuration_validation.go one-for-one so the
// admin gets the same feedback here that Go's Validate() would otherwise
// only surface after a round trip to Save. Keep in sync with that file -
// there is no build-time check tying the two together, so a reviewer needs
// to check both whenever either changes.
export const MAX_WELCOME_MESSAGES = 100;
export const MAX_ACTIONS_PER_MESSAGE = 25;
export const MAX_CHANNELS_PER_ACTION = 50;
export const MAX_DELAY_IN_SECONDS = 3600;
export const MAX_TEXT_FIELD_CHARS = 10000;
export const MAX_ACTION_DISPLAY_NAME = 128;

export const TEAM_NAME_RE = /^[a-z0-9-]{2,64}$/;
export const ACTION_NAME_RE = /^[a-zA-Z0-9_-]{1,64}$/;
export const CHANNEL_NAME_RE = /^[a-z0-9-]{2,64}$/;

export type ActionErrors = {
    actionType?: string;
    actionDisplayName?: string;
    actionName?: string;
    actionSuccessfulMessage?: string;
    channelsAddedTo?: string;
};

export type RowErrors = {
    teamName?: string;
    delayInSeconds?: string;
    message?: string;
    attachmentMessage?: string;
    actionsCount?: string;
    actions: ActionErrors[];
};

// validateTextLength only checks length. Message/AttachmentMessage/
// ActionSuccessfulMessage are also parsed as Go html/template on save -
// there's no JS-side equivalent parser, so template-syntax errors are only
// caught server-side (see the plan's Go-vs-client validation split).
function validateTextLength(name: string, lines: string[]): string | undefined {
    const joined = lines.join('\n');
    if (joined.length > MAX_TEXT_FIELD_CHARS) {
        return `${name} is too long: ${joined.length} characters, max ${MAX_TEXT_FIELD_CHARS}.`;
    }
    return undefined;
}

export function validateAction(action: ConfigMessageAction, isDuplicateName: boolean): ActionErrors {
    const errors: ActionErrors = {};

    if (action.ActionType !== ACTION_TYPE_AUTOMATIC && action.ActionType !== ACTION_TYPE_BUTTON) {
        errors.actionType = `Must be "${ACTION_TYPE_AUTOMATIC}" or "${ACTION_TYPE_BUTTON}".`;
    }

    if (!ACTION_NAME_RE.test(action.ActionName)) {
        errors.actionName = 'Must be 1-64 characters: letters, numbers, hyphens, or underscores.';
    } else if (isDuplicateName) {
        errors.actionName = 'Duplicate action name within this welcome message.';
    }

    if (action.ActionType === ACTION_TYPE_BUTTON) {
        if (!action.ActionDisplayName) {
            errors.actionDisplayName = 'Required for button actions.';
        } else if (action.ActionDisplayName.length > MAX_ACTION_DISPLAY_NAME) {
            errors.actionDisplayName = `Must be ${MAX_ACTION_DISPLAY_NAME} characters or fewer.`;
        }
    }

    const successMessageError = validateTextLength('Success message', action.ActionSuccessfulMessage);
    if (successMessageError) {
        errors.actionSuccessfulMessage = successMessageError;
    }

    if (action.ChannelsAddedTo.length > MAX_CHANNELS_PER_ACTION) {
        errors.channelsAddedTo = `At most ${MAX_CHANNELS_PER_ACTION} channels.`;
    } else if (action.ChannelsAddedTo.some((channel) => !CHANNEL_NAME_RE.test(channel))) {
        errors.channelsAddedTo = 'One or more selected channels have an invalid handle.';
    }

    return errors;
}

export function actionHasErrors(errors: ActionErrors): boolean {
    return Object.keys(errors).length > 0;
}

function findDuplicateActionNames(actions: ConfigMessageAction[]): Set<string> {
    const seen = new Set<string>();
    const dupes = new Set<string>();
    for (const action of actions) {
        if (seen.has(action.ActionName)) {
            dupes.add(action.ActionName);
        }
        seen.add(action.ActionName);
    }
    return dupes;
}

export function validateRow(message: ConfigMessage): RowErrors {
    const errors: RowErrors = {actions: []};

    if (!TEAM_NAME_RE.test(message.TeamName)) {
        errors.teamName = 'Must be a valid team handle: 2-64 lowercase letters, numbers, or hyphens.';
    }

    if (message.DelayInSeconds < 0 || message.DelayInSeconds > MAX_DELAY_IN_SECONDS) {
        errors.delayInSeconds = `Must be between 0 and ${MAX_DELAY_IN_SECONDS} seconds.`;
    }

    const messageError = validateTextLength('Message', message.Message);
    if (messageError) {
        errors.message = messageError;
    }

    const attachmentError = validateTextLength('Attachment message', message.AttachmentMessage);
    if (attachmentError) {
        errors.attachmentMessage = attachmentError;
    }

    if (message.Actions.length > MAX_ACTIONS_PER_MESSAGE) {
        errors.actionsCount = `At most ${MAX_ACTIONS_PER_MESSAGE} actions per welcome message.`;
    }

    const duplicateNames = findDuplicateActionNames(message.Actions);
    errors.actions = message.Actions.map((action) => validateAction(action, duplicateNames.has(action.ActionName)));

    return errors;
}

export function rowHasErrors(errors: RowErrors): boolean {
    if (errors.teamName || errors.delayInSeconds || errors.message || errors.attachmentMessage || errors.actionsCount) {
        return true;
    }
    return errors.actions.some(actionHasErrors);
}

export type ValidationResult = {
    valid: boolean;
    rowErrors: RowErrors[];
    summary?: string;
};

export function validateWelcomeMessages(messages: ConfigMessage[]): ValidationResult {
    const rowErrors = messages.map(validateRow);

    if (messages.length > MAX_WELCOME_MESSAGES) {
        return {
            valid: false,
            rowErrors,
            summary: `At most ${MAX_WELCOME_MESSAGES} welcome messages are allowed (currently ${messages.length}).`,
        };
    }

    return {
        valid: !rowErrors.some(rowHasErrors),
        rowErrors,
    };
}
