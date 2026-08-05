import {screen, fireEvent} from '@testing-library/react';
import React from 'react';

import ActionEditor from './action_editor';
import {emptyGlobalState, makeFakeStore, renderWithStore} from './test_utils';
import {emptyAction, ACTION_TYPE_AUTOMATIC, ACTION_TYPE_BUTTON} from './types';

function renderAction(overrides: Partial<ReturnType<typeof emptyAction>> = {}, errors = {}) {
    const onChange = jest.fn();
    const onRemove = jest.fn();
    const store = makeFakeStore(emptyGlobalState());
    renderWithStore(
        <ActionEditor
            value={{...emptyAction(), ...overrides}}
            teamName='engineering'
            onChange={onChange}
            onRemove={onRemove}
            errors={errors}
        />,
        store,
    );
    return {onChange, onRemove};
}

test('button display name field is only shown for button actions', () => {
    renderAction({ActionType: ACTION_TYPE_AUTOMATIC});
    expect(screen.queryByLabelText(/button text/i)).not.toBeInTheDocument();

    renderAction({ActionType: ACTION_TYPE_BUTTON});
    expect(screen.getByText(/button text/i)).toBeInTheDocument();
});

test('switching action type bubbles a single onChange with the merged value', () => {
    const {onChange} = renderAction({ActionType: ACTION_TYPE_AUTOMATIC, ActionName: 'join'});

    fireEvent.click(screen.getByLabelText(/button/i));

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ActionType: ACTION_TYPE_BUTTON, ActionName: 'join'}));
});

test('duplicate action name error is surfaced inline', () => {
    renderAction({ActionName: 'join'}, {actionName: 'Duplicate action name within this welcome message.'});
    expect(screen.getByText(/duplicate action name/i)).toBeInTheDocument();
});

test('removing an action calls onRemove', () => {
    const {onRemove} = renderAction();
    fireEvent.click(screen.getByText(/remove action/i));
    expect(onRemove).toHaveBeenCalledTimes(1);
});
