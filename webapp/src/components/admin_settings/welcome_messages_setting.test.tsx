import {screen, fireEvent} from '@testing-library/react';
import React from 'react';

import {emptyGlobalState, makeFakeStore, renderWithStore} from './test_utils';
import type {ConfigMessage} from './types';
import WelcomeMessagesSetting from './welcome_messages_setting';

function renderSetting(value: ConfigMessage[] | null | undefined) {
    const onChange = jest.fn();
    const setSaveNeeded = jest.fn();
    const registerSaveAction = jest.fn();
    const unRegisterSaveAction = jest.fn();
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <WelcomeMessagesSetting
            id='WelcomeMessages'
            value={value}
            onChange={onChange}
            setSaveNeeded={setSaveNeeded}
            registerSaveAction={registerSaveAction}
            unRegisterSaveAction={unRegisterSaveAction}
        />,
        store,
    );
    return {onChange, setSaveNeeded, registerSaveAction, unRegisterSaveAction};
}

test('renders with an empty array without crashing', () => {
    renderSetting([]);
    expect(screen.getByText(/\+ blank welcome message/i)).toBeInTheDocument();
});

test('renders with a null value (fresh install) without crashing', () => {
    renderSetting(null);
    expect(screen.getByText(/\+ blank welcome message/i)).toBeInTheDocument();
});

test('all three template buttons are offered', () => {
    renderSetting([]);
    expect(screen.getByText(/\+ blank welcome message/i)).toBeInTheDocument();
    expect(screen.getByText(/\+ welcome message for all new users/i)).toBeInTheDocument();
    expect(screen.getByText(/\+ welcome message for members of a specific team/i)).toBeInTheDocument();
});

test('clicking the blank template appends an empty row and calls onChange and setSaveNeeded once', () => {
    const {onChange, setSaveNeeded} = renderSetting([]);

    fireEvent.click(screen.getByText(/\+ blank welcome message/i));

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith('WelcomeMessages', [expect.objectContaining({TeamName: '', Message: [], Actions: []})]);
    expect(setSaveNeeded).toHaveBeenCalledTimes(1);
});

test('clicking the "all new users" template pre-fills message text and an automatic town-square action', () => {
    const {onChange} = renderSetting([]);

    fireEvent.click(screen.getByText(/\+ welcome message for all new users/i));

    const [, messages] = onChange.mock.calls[0];
    expect(messages[0].Message.join('\n')).toContain('{{.UserDisplayName}}');
    expect(messages[0].Actions).toHaveLength(1);
    expect(messages[0].Actions[0].ChannelsAddedTo).toEqual(['town-square']);
});

test('clicking the "specific team" template pre-fills team-name-referencing text and a button action', () => {
    const {onChange} = renderSetting([]);

    fireEvent.click(screen.getByText(/\+ welcome message for members of a specific team/i));

    const [, messages] = onChange.mock.calls[0];
    expect(messages[0].Message.join('\n')).toContain('{{.Team.DisplayName}}');
    expect(messages[0].Actions[0].ActionType).toBe('button');
});

test('registers a save action on mount and unregisters it on unmount', () => {
    const {registerSaveAction, unRegisterSaveAction} = renderSetting([]);
    expect(registerSaveAction).toHaveBeenCalledTimes(1);
    expect(unRegisterSaveAction).not.toHaveBeenCalled();
});

test('the registered save action rejects when a row is invalid', async () => {
    const {registerSaveAction} = renderSetting([{
        TeamName: 'Not A Valid Handle',
        Actions: [],
        Message: [],
        AttachmentMessage: [],
        DelayInSeconds: 0,
        IncludeGuests: false,
    }]);

    const saveAction = registerSaveAction.mock.calls[0][0];
    const result = await saveAction();
    expect(result.error).toBeDefined();
});

test('the registered save action succeeds when every row is valid', async () => {
    const {registerSaveAction} = renderSetting([{
        TeamName: 'engineering',
        Actions: [],
        Message: [],
        AttachmentMessage: [],
        DelayInSeconds: 5,
        IncludeGuests: false,
    }]);

    const saveAction = registerSaveAction.mock.calls[0][0];
    const result = await saveAction();
    expect(result.error).toBeUndefined();
});
