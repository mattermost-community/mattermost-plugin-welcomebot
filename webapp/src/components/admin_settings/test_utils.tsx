import {render} from '@testing-library/react';
import React from 'react';
import {Provider} from 'react-redux';

// A minimal fake store for components that read team/channel state via
// useSelector/useStore or dispatch mattermost-redux thunks. dispatch is a
// jest.fn so tests can assert it was called without actually running real
// thunk bodies (which would try to hit the real Mattermost REST API).
export function makeFakeStore(state: any = {}, dispatchImpl?: (action: any) => any) {
    const dispatch = jest.fn(dispatchImpl ?? (() => Promise.resolve({data: []})));
    return {
        getState: () => state,
        dispatch,
        subscribe: () => () => {},
    };
}

export function emptyGlobalState() {
    return {
        entities: {
            teams: {teams: {}},
            channels: {channels: {}, channelsInTeam: {}},
        },
    };
}

export function renderWithStore(ui: React.ReactElement, store: ReturnType<typeof makeFakeStore>) {
    return render(<Provider store={store as any}>{ui}</Provider>);
}
