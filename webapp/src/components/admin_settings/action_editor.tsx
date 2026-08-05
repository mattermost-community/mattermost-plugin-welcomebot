import React, {useId} from 'react';

import ChannelMultiSelect from './channel_multiselect';
import {textToLines, linesToText} from './text_lines';
import {ACTION_TYPE_AUTOMATIC, ACTION_TYPE_BUTTON} from './types';
import type {ConfigMessageAction} from './types';
import type {ActionErrors} from './validation';

type Props = {
    value: ConfigMessageAction;
    teamName: string;
    onChange: (value: ConfigMessageAction) => void;
    onRemove: () => void;
    errors: ActionErrors;
    disabled?: boolean;
};

// One ConfigMessageAction. Controlled - every field change is merged into
// props.value and bubbled up via a single props.onChange call, per the
// one-onChange-per-edit contract the whole component tree follows.
export default function ActionEditor(props: Props) {
    const {value, errors} = props;
    const uid = useId();

    return (
        <div className='welcomebot-action-editor'>
            <div className='form-group'>
                <label
                    id={`${uid}-type-label`}
                    title='Automatic actions run immediately when the welcome message is sent. Button actions show a clickable button that the user must click to run the action.'
                >
                    {'Action type'}
                </label>
                <div
                    role='group'
                    aria-labelledby={`${uid}-type-label`}
                >
                    <label
                        className='radio-inline'
                        htmlFor={`${uid}-type-automatic`}
                        title='Runs immediately, with no button click needed - e.g. auto-add the user to a channel.'
                    >
                        <input
                            id={`${uid}-type-automatic`}
                            type='radio'
                            name={`${uid}-type`}
                            checked={value.ActionType === ACTION_TYPE_AUTOMATIC}
                            disabled={props.disabled}
                            onChange={() => props.onChange({...value, ActionType: ACTION_TYPE_AUTOMATIC})}
                        />
                        {'Automatic'}
                    </label>
                    <label
                        className='radio-inline'
                        htmlFor={`${uid}-type-button`}
                        title='Shows a clickable button in the welcome message; the action only runs once the user clicks it.'
                    >
                        <input
                            id={`${uid}-type-button`}
                            type='radio'
                            name={`${uid}-type`}
                            checked={value.ActionType === ACTION_TYPE_BUTTON}
                            disabled={props.disabled}
                            onChange={() => props.onChange({...value, ActionType: ACTION_TYPE_BUTTON})}
                        />
                        {'Button'}
                    </label>
                </div>
                {errors.actionType && <div className='has-error control-label'>{errors.actionType}</div>}
            </div>

            {value.ActionType === ACTION_TYPE_BUTTON && (
                <div className='form-group'>
                    <label
                        htmlFor={`${uid}-display-name`}
                        title='The text shown on the clickable button. Only used for button actions.'
                    >
                        {'Button text'}
                    </label>
                    <input
                        id={`${uid}-display-name`}
                        type='text'
                        className='form-control'
                        title='The text shown on the clickable button.'
                        value={value.ActionDisplayName}
                        disabled={props.disabled}
                        onChange={(e) => props.onChange({...value, ActionDisplayName: e.target.value})}
                    />
                    {errors.actionDisplayName && <div className='has-error control-label'>{errors.actionDisplayName}</div>}
                </div>
            )}

            <div className='form-group'>
                <label
                    htmlFor={`${uid}-name`}
                    title='A short, URL-safe identifier for this action (letters, numbers, hyphens, underscores). Must be unique within this welcome message - not shown to users.'
                >
                    {'Action name'}
                </label>
                <input
                    id={`${uid}-name`}
                    type='text'
                    className='form-control'
                    title='Internal identifier - letters, numbers, hyphens, underscores. Must be unique within this welcome message.'
                    value={value.ActionName}
                    disabled={props.disabled}
                    onChange={(e) => props.onChange({...value, ActionName: e.target.value})}
                />
                {errors.actionName && <div className='has-error control-label'>{errors.actionName}</div>}
            </div>

            <div className='form-group'>
                <label
                    htmlFor={`${uid}-success-message`}
                    title='Shown to the user after this action completes successfully (e.g. after they click the button, or right after an automatic action runs).'
                >
                    {'Success message'}
                </label>
                <textarea
                    id={`${uid}-success-message`}
                    className='form-control'
                    title='Shown to the user after this action completes successfully.'
                    value={linesToText(value.ActionSuccessfulMessage)}
                    disabled={props.disabled}
                    onChange={(e) => props.onChange({...value, ActionSuccessfulMessage: textToLines(e.target.value)})}
                />
                {errors.actionSuccessfulMessage && <div className='has-error control-label'>{errors.actionSuccessfulMessage}</div>}
            </div>

            <div className='form-group'>
                <label
                    id={`${uid}-channels-label`}
                    title='The channels the user will be added to when this action runs. Only channels on the team selected above are shown.'
                >
                    {'Channels to add the user to'}
                </label>
                <ChannelMultiSelect
                    id={`${uid}-channels`}
                    teamName={props.teamName}
                    value={value.ChannelsAddedTo}
                    disabled={props.disabled}
                    onChange={(channels) => props.onChange({...value, ChannelsAddedTo: channels})}
                />
                {errors.channelsAddedTo && <div className='has-error control-label'>{errors.channelsAddedTo}</div>}
            </div>

            <button
                type='button'
                className='btn btn-link welcomebot-remove-action'
                title='Delete this action.'
                disabled={props.disabled}
                onClick={props.onRemove}
            >
                {'Remove action'}
            </button>
        </div>
    );
}
