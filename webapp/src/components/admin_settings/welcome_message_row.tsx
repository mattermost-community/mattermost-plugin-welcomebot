import React, {useEffect, useId, useState} from 'react';

import ActionEditor from './action_editor';
import TeamSelect from './team_select';
import {textToLines, linesToText} from './text_lines';
import {emptyAction} from './types';
import type {ConfigMessage} from './types';
import {MAX_ACTIONS_PER_MESSAGE} from './validation';
import type {RowErrors} from './validation';

const CLEARED_NOTICE_TIMEOUT_MS = 4000;

type Props = {
    value: ConfigMessage;
    onChange: (value: ConfigMessage) => void;
    onRemove: () => void;
    errors: RowErrors;
    disabled?: boolean;
};

// One ConfigMessage. Owns no data state of its own - it's fully controlled,
// merging every leaf edit (including nested action edits) into a single
// props.onChange call, per the bubble-up contract described in the plan.
export default function WelcomeMessageRow(props: Props) {
    const {value, errors} = props;
    const uid = useId();
    const [justClearedChannels, setJustClearedChannels] = useState(false);

    useEffect(() => {
        if (!justClearedChannels) {
            return undefined;
        }
        const handle = setTimeout(() => setJustClearedChannels(false), CLEARED_NOTICE_TIMEOUT_MS);
        return () => clearTimeout(handle);
    }, [justClearedChannels]);

    const handleTeamChange = (newTeamName: string) => {
        const oldTeamName = value.TeamName;

        // Channels are scoped to the message's team - a channel handle
        // selected under the old team has no valid meaning under the new
        // one (and could even collide by name with an unrelated channel),
        // so any actual team change clears every action's ChannelsAddedTo
        // unconditionally rather than trying to carry anything forward.
        const changingTeam = Boolean(oldTeamName) && oldTeamName !== newTeamName;
        const clearedAny = changingTeam && value.Actions.some((action) => action.ChannelsAddedTo.length > 0);
        const updatedActions = changingTeam ?
            value.Actions.map((action) => (action.ChannelsAddedTo.length === 0 ? action : {...action, ChannelsAddedTo: []})) :
            value.Actions;

        setJustClearedChannels(clearedAny);
        props.onChange({...value, TeamName: newTeamName, Actions: updatedActions});
    };

    const handleActionChange = (index: number, updatedAction: typeof value.Actions[number]) => {
        const actions = value.Actions.slice();
        actions[index] = updatedAction;
        props.onChange({...value, Actions: actions});
    };

    const handleActionRemove = (index: number) => {
        const actions = value.Actions.slice();
        actions.splice(index, 1);
        props.onChange({...value, Actions: actions});
    };

    const handleActionAdd = () => {
        props.onChange({...value, Actions: [...value.Actions, emptyAction()]});
    };

    return (
        <div className='welcomebot-message-row'>
            <div className='form-group'>
                <label
                    htmlFor={`${uid}-team`}
                    title='The Mattermost team this welcome message applies to. Each welcome message applies to one team - add another welcome message for a second team.'
                >
                    {'Team'}
                </label>
                <TeamSelect
                    id={`${uid}-team`}
                    value={value.TeamName}
                    disabled={props.disabled}
                    onChange={handleTeamChange}
                />
                {errors.teamName && <div className='has-error control-label'>{errors.teamName}</div>}
                {justClearedChannels && (
                    <div className='help-text'>{'Channel selections were cleared because the team changed.'}</div>
                )}
            </div>

            <div className='form-group'>
                <label
                    htmlFor={`${uid}-delay`}
                    title='How many seconds to wait after the user joins the team before sending this welcome message. 0 sends it immediately.'
                >
                    {'Delay (seconds)'}
                </label>
                <input
                    id={`${uid}-delay`}
                    type='number'
                    className='form-control'
                    title='How many seconds to wait after the user joins the team before sending this welcome message. 0 sends it immediately.'
                    value={value.DelayInSeconds}
                    disabled={props.disabled}
                    onChange={(e) => props.onChange({...value, DelayInSeconds: Number(e.target.value)})}
                />
                {errors.delayInSeconds && <div className='has-error control-label'>{errors.delayInSeconds}</div>}
            </div>

            <div className='form-group'>
                <label
                    className='checkbox-inline'
                    htmlFor={`${uid}-include-guests`}
                    title='If unchecked, guest accounts will not receive this welcome message when they join the team.'
                >
                    <input
                        id={`${uid}-include-guests`}
                        type='checkbox'
                        checked={value.IncludeGuests}
                        disabled={props.disabled}
                        onChange={(e) => props.onChange({...value, IncludeGuests: e.target.checked})}
                    />
                    {'Include guest users'}
                </label>
            </div>

            <div className='form-group'>
                <label
                    htmlFor={`${uid}-message`}
                    title='The welcome message text, sent as a direct message from the bot. Supports Markdown and template variables like {{.UserDisplayName}} and {{.Team.DisplayName}}.'
                >
                    {'Message'}
                </label>
                <textarea
                    id={`${uid}-message`}
                    className='form-control'
                    title='Supports Markdown and template variables like {{.UserDisplayName}} and {{.Team.DisplayName}}.'
                    value={linesToText(value.Message)}
                    disabled={props.disabled}
                    onChange={(e) => props.onChange({...value, Message: textToLines(e.target.value)})}
                />
                {errors.message && <div className='has-error control-label'>{errors.message}</div>}
            </div>

            <div className='form-group'>
                <label
                    htmlFor={`${uid}-attachment`}
                    title="Optional extra text sent as a Slack-style attachment alongside the main message. Leave blank if you don't need it."
                >
                    {'Attachment message'}
                </label>
                <textarea
                    id={`${uid}-attachment`}
                    className='form-control'
                    title='Optional - sent as an attachment alongside the message above. Same Markdown and template variable support.'
                    value={linesToText(value.AttachmentMessage)}
                    disabled={props.disabled}
                    onChange={(e) => props.onChange({...value, AttachmentMessage: textToLines(e.target.value)})}
                />
                {errors.attachmentMessage && <div className='has-error control-label'>{errors.attachmentMessage}</div>}
            </div>

            <div className='welcomebot-actions'>
                <label
                    id={`${uid}-actions-label`}
                    title='Optional follow-up actions the user can take from the welcome message, such as being added to more channels automatically or by clicking a button.'
                >
                    {'Actions'}
                </label>
                {errors.actionsCount && <div className='has-error control-label'>{errors.actionsCount}</div>}
                {value.Actions.map((action, index) => (
                    <ActionEditor
                        // eslint-disable-next-line react/no-array-index-key
                        key={index}
                        value={action}
                        teamName={value.TeamName}
                        disabled={props.disabled}
                        errors={errors.actions[index] ?? {}}
                        onChange={(updated) => handleActionChange(index, updated)}
                        onRemove={() => handleActionRemove(index)}
                    />
                ))}
                <button
                    type='button'
                    className='btn btn-link'
                    title='Add another follow-up action to this welcome message.'
                    disabled={props.disabled || value.Actions.length >= MAX_ACTIONS_PER_MESSAGE}
                    onClick={handleActionAdd}
                >
                    {'+ Add action'}
                </button>
            </div>

            <button
                type='button'
                className='btn btn-link welcomebot-remove-message'
                title='Delete this welcome message entirely.'
                disabled={props.disabled}
                onClick={props.onRemove}
            >
                {'Remove welcome message'}
            </button>
        </div>
    );
}
