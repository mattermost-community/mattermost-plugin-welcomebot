type Actions = {
    actionType: string,
    actionName: string,
    actionDisplayName: string,
    channelsAddedTo: string[],
    actionSuccessfullMessage: string[]
};

type Configs = {
    teamName: string,
    delayInSeconds: number,
    message: string[],
    includeGuests: string | null,
    attachmentMessage: string[] | null,
    actions: Actions[] | null
};

type GroupTypes = {
    label: string;
    value: string;
};

type OptionTypes = {
    value: string;
    label: string;
    data: string;
};

type Teams = {
    display_name: string;
};

type Channels = {
    display_name: string;
    team_name: string,
};
