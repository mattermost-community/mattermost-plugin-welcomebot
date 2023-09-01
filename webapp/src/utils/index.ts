import Cookies from 'js-cookie';

import {plugins, apiVersion1, apiVersion4, bearer, mmAuthToken} from 'constants/apiConstants';

import {id} from 'manifest';

const getBaseUrls = (mmSiteUrl: string): {pluginApiBaseUrl: string; mattermostApiBaseUrl: string} => {
    const pluginUrl = `${mmSiteUrl}${plugins}/${id}`;
    const pluginApiBaseUrl = `${pluginUrl}${apiVersion1}`;
    const mattermostApiBaseUrl = `${mmSiteUrl}${apiVersion4}`;

    return {pluginApiBaseUrl, mattermostApiBaseUrl};
};

const getAuthToken = () => {
    const authToken = bearer + Cookies.get(`${mmAuthToken}`) || '';
    return authToken;
};

export default {
    getBaseUrls,
    getAuthToken,
};
