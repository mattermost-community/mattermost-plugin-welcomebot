// Mirrors server/configuration.go's ConfigMessage / ConfigMessageAction.
// No field renaming - these are serialized directly to/from config.json
// with the exact Go field names (no struct tags on the Go side).

export type ConfigMessageAction = {
    ActionType: string;
    ActionDisplayName: string;
    ActionName: string;
    ActionSuccessfulMessage: string[];
    ChannelsAddedTo: string[];
};

export type ConfigMessage = {
    TeamName: string;
    Actions: ConfigMessageAction[];
    Message: string[];
    AttachmentMessage: string[];
    DelayInSeconds: number;
    IncludeGuests: boolean;
};

export const ACTION_TYPE_AUTOMATIC = 'automatic';
export const ACTION_TYPE_BUTTON = 'button';

export function emptyAction(): ConfigMessageAction {
    return {
        ActionType: ACTION_TYPE_AUTOMATIC,
        ActionDisplayName: '',
        ActionName: '',
        ActionSuccessfulMessage: [],
        ChannelsAddedTo: [],
    };
}

export function emptyMessage(): ConfigMessage {
    return {
        TeamName: '',
        Actions: [],
        Message: [],
        AttachmentMessage: [],
        DelayInSeconds: 0,
        IncludeGuests: false,
    };
}
