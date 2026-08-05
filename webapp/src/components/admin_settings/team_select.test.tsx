import {screen, waitFor} from '@testing-library/react';
import React from 'react';

import TeamSelect from './team_select';
import {emptyGlobalState, makeFakeStore, renderWithStore} from './test_utils';

test('dispatches a team-fetch action creator on mount', async () => {
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <TeamSelect
            value=''
            onChange={jest.fn()}
        />,
        store,
    );

    await waitFor(() => expect(store.dispatch).toHaveBeenCalled());
});

test('shows the current value even before it appears in fetched results', () => {
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <TeamSelect
            value='engineering'
            onChange={jest.fn()}
        />,
        store,
    );

    expect(screen.getByRole('option', {name: 'engineering'})).toBeInTheDocument();
});
