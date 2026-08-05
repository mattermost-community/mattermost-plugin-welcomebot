import {screen, waitFor} from '@testing-library/react';
import React from 'react';

import ChannelMultiSelect from './channel_multiselect';
import {emptyGlobalState, makeFakeStore, renderWithStore} from './test_utils';

test('prompts to select a team first when no team is set', () => {
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <ChannelMultiSelect
            teamName=''
            value={[]}
            onChange={jest.fn()}
        />,
        store,
    );

    expect(screen.getByText(/select a team above/i)).toBeInTheDocument();
    expect(store.dispatch).not.toHaveBeenCalled();
});

test('dispatches a channel-fetch action creator once the team resolves', async () => {
    const state = {
        entities: {
            teams: {teams: {t1: {id: 't1', name: 'engineering', display_name: 'Engineering'}}},
            channels: {channels: {}, channelsInTeam: {}},
        },
    };
    const store = makeFakeStore(state);
    renderWithStore(
        <ChannelMultiSelect
            teamName='engineering'
            value={[]}
            onChange={jest.fn()}
        />,
        store,
    );

    await waitFor(() => expect(store.dispatch).toHaveBeenCalled());
});

test('renders already-selected channels as checked', () => {
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <ChannelMultiSelect
            teamName='engineering'
            value={['town-square']}
            onChange={jest.fn()}
        />,
        store,
    );

    expect(screen.getByText('town-square')).toBeInTheDocument();
});
