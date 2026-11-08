import React, {useCallback, useEffect, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import type {Channel} from '@mattermost/types/channels';
import type {GlobalState} from '@mattermost/types/store';

import {getChannels as fetchChannels, autocompleteChannels} from 'mattermost-redux/actions/channels';
import {getTeamByName} from 'mattermost-redux/selectors/entities/teams';

import {MAX_CHANNELS_PER_ACTION} from './validation';

const SEARCH_DEBOUNCE_MS = 150;
const DEFAULT_PAGE_SIZE = 50;

type Props = {
    id?: string;
    teamName: string;
    value: string[]; // channel names (URL handles)
    onChange: (channelNames: string[]) => void;
    disabled?: boolean;
};

// Multi-select of channels scoped to a single team, via mattermost-redux
// actions dispatched into the ambient store (same approach as TeamSelect -
// no custom plugin HTTP endpoint needed).
export default function ChannelMultiSelect(props: Props) {
    const dispatch = useDispatch();
    const team = useSelector((state: GlobalState) => getTeamByName(state, props.teamName));
    const [term, setTerm] = useState('');
    const [channels, setChannels] = useState<Channel[]>([]);

    const runSearch = useCallback((searchTerm: string) => {
        if (!team) {
            setChannels([]);
            return;
        }
        const action = searchTerm ? autocompleteChannels(team.id, searchTerm) : fetchChannels(team.id, 0, DEFAULT_PAGE_SIZE);
        (dispatch(action as any) as unknown as Promise<{data?: Channel[]}>).
            then((result) => setChannels(result?.data ?? [])).
            catch(() => setChannels([]));
    }, [dispatch, team]);

    useEffect(() => {
        const handle = setTimeout(() => runSearch(term), SEARCH_DEBOUNCE_MS);
        return () => clearTimeout(handle);
    }, [term, runSearch]);

    if (!props.teamName) {
        return (
            <div
                id={props.id}
                className='help-text'
            >
                {'Select a team above to choose channels.'}
            </div>
        );
    }

    const toggleChannel = (channelName: string) => {
        if (props.value.includes(channelName)) {
            props.onChange(props.value.filter((name) => name !== channelName));
        } else if (props.value.length < MAX_CHANNELS_PER_ACTION) {
            props.onChange([...props.value, channelName]);
        }
    };

    const atLimit = props.value.length >= MAX_CHANNELS_PER_ACTION;
    const unselected = channels.filter((channel) => !props.value.includes(channel.name));

    return (
        <div
            id={props.id}
            className='welcomebot-channel-multiselect'
        >
            <input
                type='text'
                className='form-control'
                placeholder='Search channels...'
                value={term}
                disabled={props.disabled}
                onChange={(e) => setTerm(e.target.value)}
            />
            <div
                className='welcomebot-channel-list'
                role='listbox'
                aria-multiselectable='true'
            >
                {props.value.map((channelName) => (
                    <label
                        key={channelName}
                        className='checkbox-inline'
                    >
                        <input
                            type='checkbox'
                            checked={true}
                            disabled={props.disabled}
                            onChange={() => toggleChannel(channelName)}
                        />
                        {channelName}
                    </label>
                ))}
                {unselected.map((channel) => (
                    <label
                        key={channel.id}
                        className='checkbox-inline'
                    >
                        <input
                            type='checkbox'
                            checked={false}
                            disabled={props.disabled || atLimit}
                            onChange={() => toggleChannel(channel.name)}
                        />
                        {`${channel.display_name} (${channel.name})`}
                    </label>
                ))}
            </div>
            {atLimit && (
                <div className='help-text'>{`At most ${MAX_CHANNELS_PER_ACTION} channels per action.`}</div>
            )}
        </div>
    );
}
