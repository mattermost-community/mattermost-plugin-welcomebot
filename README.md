# Welcome Bot Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-welcomebot/master.svg)

Use this plugin to improve onboarding and HR processes. It adds a Welcome Bot that helps add new team members to channels.

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
"WelcomeMessages": [
    {
        "TeamName": "your-team-name",
        "DelayInSeconds": 3,
        "Message": [
            "Your welcome message here. Each list item specifies one line in the message text."
        ]
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
                ]
                "ChannelsAddedTo": ["channel1", "channel2"],
            },
            {
                "ActionType": "automatic",
                "ChannelsAddedTo": ["channel3", "channel4"]
            }
        ]
    }
]
```

where

- **TeamName**: The team for which the Welcome Bot sends a message for. Refers to the team handle used in the URL. For example, in the following URL the **TeamName** value is `myteam`: https://example.com/myteam/channels/mychannel
- **DelayInSeconds**: The number of seconds after joining a team that the user receives a welcome message.
- **Message**: The message posted to the user.
- (Optional) **AttachmentMessage**: Message text in attachment containing user action buttons. 
- (Optional) **Actions**: Use this to add new team members to channels automatically or based on which action button they pressed.
    - **ActionType**: One of `button` or `automatic`. When `button`: enables uses to select which types of channels they want to join. When `automatic`: the user is automatically added to the specified channels.
    - **ActionDisplayName**: Sets the display name for the user action buttons.
    - **ActionName**: Sets the action name used by the plugin to identify which action is taken by a user.
    - **ActionSuccessfulMessage**: Message posted after the user takes this action and joins the specified channels.
    - **ChannelsAddedTo**: List of channels the user is added to.

## Example

Suppose you have two teams: one for Staff (with team handle `staff`) which all staff members join, and another for DevSecOps (team handle `devsecops`), which only security engineers join.

Those who join the Staff team should be added to a set of channels based on their role:
  - Developers added to Bugs, Jira Tasks, and Sprint Planning channels
  - Account Managers added to Leads, Sales Discussion, and Win-Loss Analysis channels
  - Support added to Bugs, Customer Support and Leads channels

Moreover, those who join the DevSecOps team should automatically be added to Escalation Process and Incidents channels.

To accomplish the above, you can specify the following configuration in your config.json file.

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
                    "Let's get started by adding you to key channels! What is your role in the company?"
                ],
                "Actions" : [
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "Developer",
                        "ActionName": "developer-action",
                        "ChannelsAddedTo": ["bugs", "jira-tasks", "sprint-planning"],
                        "ActionSuccessfulMessage": [
                            "### Awesome! I have added you to the following developer channels:",
                            "~bugs - To help investigate or report bugs",
                            "~jira-tasks - To stay updated on Jira tasks",
                            "~sprint-planning - To plan and manage your team's Jira sprint"
                        ]
                    },
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "Account Manager",
                        "ActionName": "account-manager-action",
                        "ChannelsAddedTo": ["leads", "sales-discussion", "win-loss-analysis"],
                        "ActionSuccessfulMessage": [
                            "### Awesome! I have added you to the following developer channels:",
                            "~leads - To stay updated on incoming leads",
                            "~sales-discussion - To collaborate with your fellow account managers",
                            "~win-loss-analysis - To conduct win-loss analysis of closed deals"
                        ]
                    },
                    {
                        "ActionType": "button",
                        "ActionDisplayName": "Support",
                        "ActionName": "support-action",
                        "ChannelsAddedTo": ["bugs", "customer-support", "leads"],
                        "ActionSuccessfulMessage": [
                            "### Awesome! I have added you to the following developer channels:",
                            "~bugs - To help investigate or report bugs",
                            "~customer-support - To troubleshoot and resolve customer issues",
                            "~leads - To discuss potential accounts with other account managers"
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
                    "**If you are not a member of the Security Meta Team and ended up on this team by accident, please report this issue and leave the team!**",
                    "",
                    "##### I've added you to a few channels to get you started:",
                    "",
                    "~escalation-process - To review the DevSecOps escalation process",
                    "~incidents - To collaborate and resolve seucrity incidents",
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
