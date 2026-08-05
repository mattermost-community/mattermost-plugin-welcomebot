import React, {useEffect, useRef, useState} from 'react';
import {generateKey} from 'utils/id';

import {MESSAGE_TEMPLATES} from './message_templates';
import type {MessageTemplate} from './message_templates';
import type {ConfigMessage} from './types';
import {validateWelcomeMessages, MAX_WELCOME_MESSAGES} from './validation';
import WelcomeMessageRow from './welcome_message_row';

type SaveResult = {error?: {message?: string}};
type SaveAction = () => Promise<SaveResult>;

// Matches Mattermost's PluginCustomSettingsComponentProps<ConfigMessage[]>
// (webapp/src/types/mattermost-webapp/index.d.ts), except value is typed to
// allow null/undefined - a fresh install has no WelcomeMessages saved yet,
// and plugin.json's default for this setting is null.
type Props = {
    id: string;
    value: ConfigMessage[] | null | undefined;
    disabled?: boolean;
    onChange: (id: string, value: ConfigMessage[]) => void;
    setSaveNeeded: () => void;
    registerSaveAction?: (saveAction: SaveAction) => void;
    unRegisterSaveAction?: (saveAction: SaveAction) => void;
};

type KeyedMessage = {
    clientKey: string;
    data: ConfigMessage;
};

// Top-level System Console component for the WelcomeMessages setting.
// Every leaf edit bubbles up through exactly one onChange call per level
// (see action_editor.tsx / welcome_message_row.tsx); this container is
// where that bubbling terminates in the single props.onChange('WelcomeMessages', ...)
// + props.setSaveNeeded() call the System Console page's Save button needs.
export default function WelcomeMessagesSetting(props: Props) {
    const [rows, setRows] = useState<KeyedMessage[]>(
        () => (props.value ?? []).map((data) => ({clientKey: generateKey(), data})),
    );
    const rowsRef = useRef(rows);
    rowsRef.current = rows;

    const messages = rows.map((row) => row.data);
    const validation = validateWelcomeMessages(messages);

    const emitChange = (updatedRows: KeyedMessage[]) => {
        setRows(updatedRows);
        props.onChange(props.id, updatedRows.map((row) => row.data));
        props.setSaveNeeded();
    };

    useEffect(() => {
        // The System Console's Save button is shared across every setting
        // on the page, so a per-setting component can't disable it
        // directly - registering a save action lets this one block Save
        // with a clear error if the current state is invalid, on top of
        // the inline field errors already shown.
        const saveAction: SaveAction = async () => {
            const result = validateWelcomeMessages(rowsRef.current.map((row) => row.data));
            if (!result.valid) {
                return {
                    error: {
                        message: result.summary ?? 'One or more welcome messages have invalid fields. Fix the highlighted fields before saving.',
                    },
                };
            }
            return {};
        };

        props.registerSaveAction?.(saveAction);
        return () => props.unRegisterSaveAction?.(saveAction);

        // Registered once on mount; the save action reads rowsRef.current
        // rather than closing over `rows`, so it always sees the latest
        // edits without needing to re-register on every change.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const handleRowChange = (index: number, updated: ConfigMessage) => {
        const next = rows.slice();
        next[index] = {...next[index], data: updated};
        emitChange(next);
    };

    const handleRowRemove = (index: number) => {
        const next = rows.slice();
        next.splice(index, 1);
        emitChange(next);
    };

    const handleAddFromTemplate = (template: MessageTemplate) => {
        emitChange([...rows, {clientKey: generateKey(), data: template.build()}]);
    };

    const atLimit = props.disabled || rows.length >= MAX_WELCOME_MESSAGES;

    return (
        <div className='welcomebot-welcome-messages-setting'>
            <p className='help-text'>
                {'Configure per-team welcome messages: message text, delay, guest inclusion, and follow-up actions that add users to channels. Each welcome message applies to one team - use one of the templates below to get started, or add a blank one.'}
            </p>
            {validation.summary && (
                <div className='has-error control-label'>{validation.summary}</div>
            )}
            {rows.map((row, index) => (
                <WelcomeMessageRow
                    key={row.clientKey}
                    value={row.data}
                    disabled={props.disabled}
                    errors={validation.rowErrors[index]}
                    onChange={(updated) => handleRowChange(index, updated)}
                    onRemove={() => handleRowRemove(index)}
                />
            ))}
            <div className='welcomebot-add-message-templates'>
                <span className='help-text'>{'Add a welcome message:'}</span>
                {MESSAGE_TEMPLATES.map((template) => (
                    <button
                        key={template.key}
                        type='button'
                        className='btn btn-default'
                        title={template.description}
                        disabled={atLimit}
                        onClick={() => handleAddFromTemplate(template)}
                    >
                        {`+ ${template.label}`}
                    </button>
                ))}
            </div>
        </div>
    );
}
