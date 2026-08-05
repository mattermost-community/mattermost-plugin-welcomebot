import {emptyAction, emptyMessage, ACTION_TYPE_AUTOMATIC, ACTION_TYPE_BUTTON} from './types';
import {
    validateWelcomeMessages,
    validateRow,
    validateAction,
    MAX_WELCOME_MESSAGES,
    MAX_ACTIONS_PER_MESSAGE,
    MAX_CHANNELS_PER_ACTION,
    MAX_DELAY_IN_SECONDS,
} from './validation';

function validMessage() {
    return {
        ...emptyMessage(),
        TeamName: 'engineering',
        DelayInSeconds: 5,
        Actions: [{
            ...emptyAction(),
            ActionType: ACTION_TYPE_BUTTON,
            ActionDisplayName: 'Join',
            ActionName: 'join-eng',
            ChannelsAddedTo: ['town-square'],
        }],
    };
}

describe('validateWelcomeMessages', () => {
    test('empty array is valid', () => {
        expect(validateWelcomeMessages([]).valid).toBe(true);
    });

    test('a well-formed message is valid', () => {
        expect(validateWelcomeMessages([validMessage()]).valid).toBe(true);
    });

    test('rejects more than the max number of messages', () => {
        const messages = Array.from({length: MAX_WELCOME_MESSAGES + 1}, () => ({...emptyMessage(), TeamName: 'engineering'}));
        const result = validateWelcomeMessages(messages);
        expect(result.valid).toBe(false);
        expect(result.summary).toContain('At most');
    });
});

describe('validateRow', () => {
    test('invalid team name is flagged', () => {
        const errors = validateRow({...validMessage(), TeamName: 'Not A Handle'});
        expect(errors.teamName).toBeDefined();
    });

    test('valid team name at minimum length boundary is not flagged', () => {
        const errors = validateRow({...validMessage(), TeamName: 'ab'});
        expect(errors.teamName).toBeUndefined();
    });

    test('delay out of bounds is flagged', () => {
        expect(validateRow({...validMessage(), DelayInSeconds: -1}).delayInSeconds).toBeDefined();
        expect(validateRow({...validMessage(), DelayInSeconds: MAX_DELAY_IN_SECONDS + 1}).delayInSeconds).toBeDefined();
        expect(validateRow({...validMessage(), DelayInSeconds: MAX_DELAY_IN_SECONDS}).delayInSeconds).toBeUndefined();
    });

    test('too many actions is flagged', () => {
        const actions = Array.from({length: MAX_ACTIONS_PER_MESSAGE + 1}, (_, i) => ({...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: `a${i}`}));
        const errors = validateRow({...validMessage(), Actions: actions});
        expect(errors.actionsCount).toBeDefined();
    });

    test('duplicate action names within one message are flagged on both', () => {
        const actions = [
            {...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join'},
            {...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join'},
        ];
        const errors = validateRow({...validMessage(), Actions: actions});
        expect(errors.actions[0].actionName).toBeDefined();
        expect(errors.actions[1].actionName).toBeDefined();
    });
});

describe('validateAction', () => {
    test('invalid action type is flagged', () => {
        const errors = validateAction({...emptyAction(), ActionType: 'maybe', ActionName: 'join'}, false);
        expect(errors.actionType).toBeDefined();
    });

    test('button action without display name is flagged', () => {
        const errors = validateAction({...emptyAction(), ActionType: ACTION_TYPE_BUTTON, ActionName: 'join'}, false);
        expect(errors.actionDisplayName).toBeDefined();
    });

    test('automatic action does not require a display name', () => {
        const errors = validateAction({...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join'}, false);
        expect(errors.actionDisplayName).toBeUndefined();
    });

    test('too many channels is flagged', () => {
        const channels = Array.from({length: MAX_CHANNELS_PER_ACTION + 1}, () => 'town-square');
        const errors = validateAction({...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join', ChannelsAddedTo: channels}, false);
        expect(errors.channelsAddedTo).toBeDefined();
    });

    test('invalid channel handle is flagged', () => {
        const errors = validateAction({...emptyAction(), ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join', ChannelsAddedTo: ['Town Square']}, false);
        expect(errors.channelsAddedTo).toBeDefined();
    });
});
