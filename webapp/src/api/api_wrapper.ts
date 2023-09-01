import Utils from 'utils';

export const fetchChannels = async (mmSiteUrl: string) => {
    const url = Utils.getBaseUrls(mmSiteUrl).mattermostApiBaseUrl + '/channels?exclude_default_channels=true';
    const response = await fetch(url, {
        method: 'GET',
        headers: {Authentication: Utils.getAuthToken()},
    });
    return response.json();
};

export const fetchTeams = async (mmSiteUrl: string) => {
    const url = Utils.getBaseUrls(mmSiteUrl).mattermostApiBaseUrl + '/teams';
    const response = await fetch(url, {
        method: 'GET',
        headers: {Authentication: Utils.getAuthToken()},
    });
    return response.json();
};

