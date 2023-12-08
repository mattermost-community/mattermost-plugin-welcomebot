# Disclaimer

**This repository is community supported and not maintained by Mattermost. Mattermost disclaims liability for integrations, including Third Party Integrations and Mattermost Integrations. Integrations may be modified or discontinued at any time.**

# Welcome Bot Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-welcomebot/master)](https://circleci.com/gh/mattermost/mattermost-plugin-welcomebot)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-welcomebot/master)](https://codecov.io/gh/mattermost/mattermost-plugin-welcomebot)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-welcomebot)](https://github.com/mattermost/mattermost-plugin-welcomebot/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-welcomebot/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-welcomebot/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

**Maintainer:** [@mickmister](https://github.com/mickmister)

Use this plugin to improve onboarding and HR processes. It adds a Welcome Bot that helps welcome users to teams and/or channels as well as easily join channels based on selections.

![image](https://user-images.githubusercontent.com/13119842/58736467-fd226400-83cb-11e9-827b-6bbe33d062ab.png)

*Welcome a new team member to Mattermost Contributors team. Then add the user to a set of channels based on their selection.*

![image](https://user-images.githubusercontent.com/13119842/58736540-30fd8980-83cc-11e9-8e8e-94ea3042b3b1.png)

## Configuration

1. Go to **System Console > Plugins > Management** and click **Enable** to enable the Welcome Bot plugin.
    - If you are running Mattermost v5.11 or earlier, you must first go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-welcomebot/releases), download the latest release, and upload it to your Mattermost instance [following this documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).

2. Modify your `config.json` file to include your Welcome Bot's messages and actions, under the `PluginSettings`. See below for an example of what this should look like.

## Usage

To configure the Welcome Bot, edit your `config.json` file with a message you want to send to a user in the following format:

```
        "Plugins": {
            "com.mattermost.welcomebot": {
                "WelcomeMessages": [
                    {
                        "TeamName": "your-team-name",
                        "DelayInSeconds": 3,
                        "IncludeGuests": false,
                        "Message": [
                            "Your welcome message here. Each list item specifies one line in the message text."
                        ],
                        "AttachmentMessage": [
                            "Attachment message containing user actions"
                        ],
                        "Actions" : [
                            {
                                "ActionType": "button",
                                "ActionDisplayName": "User Action",
                                "ActionName": "action-name",
                                "ActionSuccessfulMessage": [
                                    "Message posted after the user takes this action and joins channels specified by 'ChannelsAddedTo'."
                                ],
                                "ChannelsAddedTo": ["channel-1", "channel-2"]
                            },
                            {
                                "ActionType": "automatic",
                                "ChannelsAddedTo": ["channel-3", "channel-4"]
                            }
                        ]
                    }
                ]
            }
        },
```

where

- **TeamName**: The team for which the Welcome Bot sends a message for. Must be the team handle used in the URL, in lowercase. For example, in the following URL the **TeamName** value is `my-team`: https://example.com/my-team/channels/my-channel
- **DelayInSeconds**: The number of seconds after joining a team that the user receives a welcome message.
- **Message**: The message posted to the user.
- (Optional) **IncludeGuests**: Whether or not to include guest users.
- (Optional) **AttachmentMessage**: Message text in attachment containing user action buttons.
- (Optional) **Actions**: Use this to add new team members to channels automatically or based on which action button they pressed.
    - **ActionType**: One of `button` or `automatic`. When `button`: enables uses to select which types of channels they want to join. When `automatic`: the user is automatically added to the specified channels.
    - **ActionDisplayName**: Sets the display name for the user action buttons.
    - **ActionName**: Sets the action name used by the plugin to identify which action is taken by a user.
    - **ActionSuccessfulMessage**: Message posted after the user takes this action and joins the specified channels.
    - **ChannelsAddedTo**: List of channel names the user is added to. Must be the channel handle used in the URL, in lowercase. For example, in the following URL the **channel name** value is `my-channel`: https://example.com/my-team/channels/my-channel

The preview of the configured messages, as well as the creation of a channel welcome message, can be done via bot commands:
* `/welcomebot help` - Displays usage information.
* `/welcomebot list` - Lists the teams for which greetings were defined.
* `/welcomebot preview [team-name]` - Sends ephemeral messages to the user calling the command, with the preview of the welcome message[s] for the given team name and the user that requested the preview.
* `/welcomebot set_channel_welcome [welcome-message]` - Sets the given text as current's channel welcome message.
* `/welcomebot get_channel_welcome` - Gets the current channel's welcome message.
* `/welcomebot delete_channel_welcome` - Deletes the current channel's welcome message.

## Example

Suppose you have two teams: one for Staff (with team handle `staff`) which all staff members join, and another for DevSecOps (team handle `devsecops`), which only security engineers join.

Those who join the Staff team should be added to a set of channels based on their role:
  - Developers added to Bugs, Jira Tasks, and Sprint Planning channels
  - Account Managers added to Leads, Sales Discussion, and Win-Loss Analysis channels
  - Support added to Bugs, Customer Support and Leads channels

Moreover, those who join the DevSecOps team should automatically be added to Escalation Process and Incidents channels.

To accomplish the above, you can specify the following configuration in your `config.json` file.

```
        "Plugins": {
            "com.mattermost.welcomebot": {
                "WelcomeMessages": [
                    {
                        "TeamName": "staff",
                        "DelayInSeconds": 5,
                        "Message": [
                            "### Welcome {{.UserDisplayName}} to the Staff {{.Team.DisplayName}} team!",
                            "",
                            "If you have any questions about your account, please message your @system-admin.",
                            "",
                            "For feedback about the Mattermost app, please share in the ~mattermost channel."
                        ]
                    },
                    {
                        "TeamName": "staff",
                        "DelayInSeconds": 10,
                        "AttachmentMessage": [
                            "Let's get started by adding you to key channels! What's your role in the company?"
                        ],
                        "Actions" : [
                            {
                                "ActionType": "button",
                                "ActionDisplayName": "Developer",
                                "ActionName": "developer-action",
                                "ChannelsAddedTo": ["bugs", "jira-tasks", "sprint-planning"],
                                "ActionSuccessfulMessage": [
                                    "### Awesome! I've added you to the following developer channels:",
                                    "~bugs - To help investigate or report bugs",
                                    "~jira-tasks - To stay updated on Jira tasks",
                                    "~sprint-planning - To plan and manage your team's Jira sprint"
                                ]
                            },
                            {
                                "ActionType": "button",
                                "ActionDisplayName": "Customer Engineer",
                                "ActionName": "customer-engineer-action",
                                "ChannelsAddedTo": ["leads", "sales-discussion", "win-loss-analysis"],
                                "ActionSuccessfulMessage": [
                                    "### Awesome! I've added you to the following developer channels:",
                                    "~leads - To stay updated on incoming leads",
                                    "~sales-discussion - To collaborate with your fellow Customer Engineers,
                                    "~win-loss-analysis - To conduct win-loss analysis of closed deals"
                                ]
                            },
                            {
                                "ActionType": "button",
                                "ActionDisplayName": "Support",
                                "ActionName": "support-action",
                                "ChannelsAddedTo": ["bugs", "customer-support", "leads"],
                                "ActionSuccessfulMessage": [
                                    "### Awesome! I've added you to the following developer channels:",
                                    "~bugs - To help investigate or report bugs",
                                    "~customer-support - To troubleshoot and resolve customer issues",
                                    "~leads - To discuss potential accounts with other Customer Engineers"
                                ]
                            }
                        ]
                    },
                    {
                        "TeamName": "devsecops",
                        "DelayInSeconds": 5,
                        "Message": [
                            "### Welcome {{.UserDisplayName}} to the {{.Team.DisplayName}} team!",
                            "",
                            "**If you're not a member of the Security Meta Team and ended up on this team by accident, please report this issue and leave the team!**",
                            "",
                            "##### I've added you to a few channels to get you started:",
                            "",
                            "~escalation-process - To review the DevSecOps escalation process",
                            "~incidents - To collaborate on and resolve security incidents"
                        ],
                        "Actions" : [
                            {
                                "ActionType": "automatic",
                                "ChannelsAddedTo": ["escalation-process", "incidents"]
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

We've used `{{.UserDisplayName}}` and `{{.Team.DisplayName}}` in the example `config.json`. You can insert any variable from the `MessageTemplate` struct, which has the following fields:

```go
type MessageTemplate struct {
    WelcomeBot      *model.User
    User            *model.User
    Team            *model.Team
    Townsquare      *model.Channel
    DirectMessage   *model.Channel
    UserDisplayName string
}
```

## Development

This plugin contains a server and webapp portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.
Use `make check-style` to check the style.
Use `make deploy` to deploy the plugin to your local server.

For additional information on developing plugins, refer to [our plugin developer documentation](https://developers.mattermost.com/extend/plugins/).
