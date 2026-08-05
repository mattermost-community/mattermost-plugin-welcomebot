import React, {useCallback, useEffect, useState} from 'react';
import {useDispatch} from 'react-redux';

import type {Team} from '@mattermost/types/teams';

import {getTeams as fetchTeams, searchTeams as searchTeamsAction} from 'mattermost-redux/actions/teams';

const SEARCH_DEBOUNCE_MS = 150;
const DEFAULT_PAGE_SIZE = 20;

type Props = {
    id?: string;
    value: string; // team name (URL handle), '' if unset
    onChange: (teamName: string) => void;
    disabled?: boolean;
};

// Single-team picker. Dispatches mattermost-redux's team search/list action
// creators directly into the ambient System Console Redux store rather than
// calling a custom plugin HTTP endpoint - see project harness notes on why
// that's unnecessary here.
export default function TeamSelect(props: Props) {
    const dispatch = useDispatch();
    const [term, setTerm] = useState('');
    const [teams, setTeams] = useState<Team[]>([]);

    const runSearch = useCallback((searchTerm: string) => {
        const action = searchTerm ? searchTeamsAction(searchTerm) : fetchTeams(0, DEFAULT_PAGE_SIZE);
        (dispatch(action as any) as unknown as Promise<{data?: Team[] | {teams: Team[]}}>).
            then((result) => {
                const data = result?.data;
                const list: Team[] = Array.isArray(data) ? data : (data?.teams ?? []);
                setTeams(list);
            }).
            catch(() => setTeams([]));
    }, [dispatch]);

    useEffect(() => {
        const handle = setTimeout(() => runSearch(term), SEARCH_DEBOUNCE_MS);
        return () => clearTimeout(handle);
    }, [term, runSearch]);

    const knownValue = teams.some((team) => team.name === props.value);

    return (
        <div className='welcomebot-team-select'>
            <input
                type='text'
                className='form-control'
                placeholder='Search for a team...'
                value={term}
                disabled={props.disabled}
                onChange={(e) => setTerm(e.target.value)}
            />
            <select
                id={props.id}
                className='form-control'
                value={props.value}
                disabled={props.disabled}
                onChange={(e) => props.onChange(e.target.value)}
            >
                <option value=''>{'Select a team...'}</option>
                {props.value && !knownValue && (
                    <option value={props.value}>{props.value}</option>
                )}
                {teams.map((team) => (
                    <option
                        key={team.id}
                        value={team.name}
                    >
                        {`${team.display_name} (${team.name})`}
                    </option>
                ))}
            </select>
        </div>
    );
}
