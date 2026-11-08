import {screen, fireEvent, waitFor} from '@testing-library/react';
import React from 'react';

import {emptyGlobalState, makeFakeStore, renderWithStore} from './test_utils';
import {emptyMessage, emptyAction} from './types';
import WelcomeMessageRow from './welcome_message_row';

const AVAILABLE_TEAMS = [
    {id: 't1', name: 'engineering', display_name: 'Engineering'},
    {id: 't2', name: 'sales', display_name: 'Sales'},
];

function renderRow(overrides: Partial<ReturnType<typeof emptyMessage>> = {}) {
    const onChange = jest.fn();
    const onRemove = jest.fn();

    // TeamSelect populates its options from whatever its dispatched fetch
    // resolves with (it doesn't read the store's team state directly), so
    // the fake store's dispatch needs to resolve with the teams these
    // tests select between.
    const store = makeFakeStore(emptyGlobalState(), () => Promise.resolve({data: AVAILABLE_TEAMS}));
    renderWithStore(
        <WelcomeMessageRow
            value={{...emptyMessage(), ...overrides}}
            onChange={onChange}
            onRemove={onRemove}
            errors={{actions: overrides.Actions?.map(() => ({})) ?? []}}
        />,
        store,
    );
    return {onChange, onRemove};
}

async function selectTeam(name: string) {
    await waitFor(() => expect(screen.getByRole('option', {name: new RegExp(name, 'i')})).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText('Team'), {target: {value: name}});
}

test('changing the team clears every action\'s selected channels in one bubbled onChange', async () => {
    const {onChange} = renderRow({
        TeamName: 'engineering',
        Actions: [{...emptyAction(), ActionName: 'join', ChannelsAddedTo: ['town-square']}],
    });

    await selectTeam('sales');

    expect(onChange).toHaveBeenCalledTimes(1);
    const updated = onChange.mock.calls[0][0];
    expect(updated.TeamName).toBe('sales');
    expect(updated.Actions[0].ChannelsAddedTo).toEqual([]);

    expect(screen.getByText(/channel selections were cleared/i)).toBeInTheDocument();
});

test('changing the team is a no-op for channels when there were none selected', async () => {
    const {onChange} = renderRow({
        TeamName: 'engineering',
        Actions: [{...emptyAction(), ActionName: 'join', ChannelsAddedTo: []}],
    });

    await selectTeam('sales');

    const updated = onChange.mock.calls[0][0];
    expect(updated.Actions[0].ChannelsAddedTo).toEqual([]);
    expect(screen.queryByText(/channel selections were cleared/i)).not.toBeInTheDocument();
});

test('setting the team for the first time does not clear anything (nothing to clear from)', async () => {
    const {onChange} = renderRow({TeamName: '', Actions: []});

    await selectTeam('engineering');

    const updated = onChange.mock.calls[0][0];
    expect(updated.TeamName).toBe('engineering');
    expect(screen.queryByText(/channel selections were cleared/i)).not.toBeInTheDocument();
});

test('adding an action calls onChange with the action appended', () => {
    const {onChange} = renderRow({TeamName: 'engineering', Actions: []});

    fireEvent.click(screen.getByText(/\+ add action/i));

    const updated = onChange.mock.calls[0][0];
    expect(updated.Actions).toHaveLength(1);
});

test('removing the message calls onRemove', () => {
    const {onRemove} = renderRow();
    fireEvent.click(screen.getByText(/remove welcome message/i));
    expect(onRemove).toHaveBeenCalledTimes(1);
});
