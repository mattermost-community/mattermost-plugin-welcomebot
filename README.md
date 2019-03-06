# WelcomeBot Plugin (Beta) ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-welcomebot/master.svg)

*TODO ADD DESC*

## Installation

1. Go to the [releases page of this Github repository](https://github.com/mattermost/mattermost-plugin-welcomebot) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Modify your `config.json` file to include the types of welcome messages you wish to send, under the `PluginSettings`. See below for an example of what this should look like.

## Usage

*TODO ADD DESC OF CONFIG SETTINGS*

Below is an example of welcome messages modified in the `config.json` file:

```
"Plugins": {
    "com.mattermost.welcomebot": {
        "WelcomeMessages": [
            {
                "TeamName": "core",
                "DelayInSeconds": 3,
                "Message": [
                    "# Welcome {{.UserDisplayName}} to the Mattermost {{.Team.DisplayName}} nightly build server!",
                    "",
                    "### Please review our Code of Conduct ",
                    "",
                    "Thank you for visiting the virtual office of the Mattermost core team, and for your interest in the [Mattermost open source project](http://www.mattermost.org/vision/#mattermost-teams-v1). ",
                    "",
                    "This is the virtual office of for the Mattermost core team, project contributors, testers and invited guests. Please behave here as you would as a guest in an office space where people are working. ",
                    "",
                    "Important notes:",
                    "",
                    "##### 1) This is a virtual office, not a demo server",
                    "",
                    "You can use [Mattermost install guides](http://www.mattermost.org/installation/) to install your own version of Matterrmost if you want to experiment with features. ",
                    "",
                    "##### 2) This is a virtual office, not the Mattermost public forum (the [forum](https://forum.mattermost.org/) is here)",
                    "",
                    "Please do not create public channels or private groups. If you want to discuss something that’s not a topic started by the core team, please start a topic on the [public forum](https://forum.mattermost.org/), or discuss in the [IRC Public Discussion channel](https://pre-release.mattermost.com/core/channels/irc#). ",
                    "",
                    "If people are responding to your @ mentions, it’s fine to continue a conversation. If your mentions have been repeatedly been ignored, please stop. ",
                    "",
                    "Please move conversations to the appropriate channel by topic and do not post long threads in the Reception area. Moderators delete or move your messages if the topic isn't relevant to 300+ members viewing the Reception area. ",
                    "",
                    "##### 3) This is a virtual office, not a substitute for community systems",
                    "",
                    "There are standard community systems for [bugs, feature ideas, and troubleshooting help](http://docs.mattermost.com/process/community-systems.html). ",
                    "",
                    "If you receive a request from the core team, or matterbot, to change your behavior in the office, please follow the request or leave the office. ",
                    "",
                    "In addition to the basics discussed above the [Contributor Code of Conduct](http://contributor-covenant.org/version/1/3/0/code_of_conduct.txt) also applies.",
                    "",
                    "Updated March 5, 2016"
                ]
            },
            {
                "TeamName": "core",
                "DelayInSeconds": 5,
                "AttachmentMessage": [
                    "I can help you get started by joining you to a bunch of existing channels!  Which types of channels would you like to join?"
                ],
                "Actions" : [
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "I'm interested in Support",
                        "ActionName": "support-action",
                        "ChannelsAddedTo": ["peer-to-peer-help", "bugs"],
                        "ActionSuccessfulMessage": [
                            "### Awesome, I have added you to the following support related channels!",
                            "~peer-to-peer-help - General help and setup",
                            "~bugs - To help investigate or report bugs"
                        ]
                    },
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "Developing on Mattermost",
                        "ActionName": "developer-action",
                        "ChannelsAddedTo": ["developers", "developer-toolkit", "developer-meeting", "developer-performance", "bugs"],
                        "ActionSuccessfulMessage": [
                            "### Baller, I have added you to the following developer related channels!",
                            "~developers - Great for general developer questions",
                            "~developer-toolkit - Great questions about plugins or integrations",
                            "~developer-meeting - Weekly core staff and community meeting",
                            "~developer-performance - Great for questions about performance or load testing",
                            "~bugs - To help investigate or report bugs"
                        ]
                    },
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "I don't Know?",
                        "ActionName": "do-not-know-action",
                        "ChannelsAddedTo": ["peer-to-peer-help", "feature-ideas", "developers", "bugs"],
                        "ActionSuccessfulMessage": [
                            "### Great, I have added you to a few channels that might be interesting!",
                            "~peer-to-peer-help - General help and setup",
                            "~feature-ideas - To discuss potential feature ideas",
                            "~developers - Great for general developer questions",
                            "~bugs - To help investigate or report bugs"
                        ]
                    }
                ]
            },
            {
                "TeamName": "private-core",
                "DelayInSeconds": 3,
                "Message": [
                    "# Welcome {{.UserDisplayName}} to the {{.Team.DisplayName}} Team for staff!",
                    "",
                    "*If you are not a core staff member and ended up on this team by accident please report this issue and leave the team!*",
                    "",
                    "#### I've added you to a few channels to get you started!",
                    "",
                    "~stand-up - For reporting daily standups",
                    "~alerts - Where internal services post critical situations",
                    "~community - A channel focused on customer support",
                    "~platform - A channel focused on the weekly R&D meeting"
                ],
                "Actions" : [
                    {
                        "ActionType": "automatic",
                        "ChannelsAddedTo": ["standup", "alerts", "community", "platform"]
                    }
                ]
            }
        ]
    }
},
"PluginStates": {
    "com.mattermost.welcomebot": {
        "Enable": true
    }
}
```
