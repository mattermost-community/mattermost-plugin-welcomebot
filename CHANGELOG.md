# Changelog

The changelog can be found at https://github.com/mattermost/mattermost-plugin-welcomebot/releases.

## v1.4.0 — 2026-04-23

### Changed

- **Channel welcome messages no longer send a bot DM.** Previously, joining a channel triggered both a DM from Welcomebot and an ephemeral post. The DM was redundant with the team-join welcome and noisy when joining multiple channels at once. Channel welcomes now deliver as an ephemeral post only. Team-join welcomes are unaffected and still deliver via DM.

### Improved

- **Channel welcome ephemeral delivery is now asynchronous.** The ephemeral fires in a background goroutine after a configurable delay so the `UserHasJoinedChannel` hook returns immediately.
- **Plugin-initiated auto-joins now deliver the channel welcome directly.** When the welcomebot adds a user to channels via `automatic` actions on team join, Mattermost does not re-fire `UserHasJoinedChannel` back to the calling plugin. The welcome ephemeral is now sent from `joinChannel` itself, ensuring delivery for the team-join auto-add flow.

### Added

- **`ChannelWelcomeAutoJoinDelaySeconds` setting.** Controls how long the plugin waits before sending the channel welcome ephemeral. Configurable from System Console → Plugins → Welcome Bot. Defaults to 5 seconds. Applies to all join types.

- **`/welcomebot welcome` command.** Any user can run this in any channel to re-show that channel's welcome message as an ephemeral visible only to them. The primary recovery path for missed welcomes — see known limitations below.

- **`POST /admin/set_channel_welcome` endpoint.** Allows system admins to set channel welcome messages programmatically. Accepts `{"channel_id": "...", "message": "..."}` with a Bearer token. Intended for setup scripts — see `setup-welcomebot.sh` for a reference implementation.

### Known Limitations

- **Channel welcome ephemerals are reliable on first join, but not on rejoin.** On a user's first-ever join to a team, the auto-added channels are cold in the client — no cached state exists. When the user navigates to each channel, the ephemeral is still live in the WebSocket buffer and renders correctly. On subsequent rejoins (leave team → rejoin), the client already has those channels in its local store and rehydrates the post list from the server. Because the server never stores ephemerals, the welcome is gone before the user opens the channel.

  This is a platform limitation. Mattermost does not provide a `UserHasViewedChannel` hook or any equivalent event that would allow the plugin to trigger delivery at the moment the user first opens a channel. There is no workaround that guarantees delivery on rejoin without user action.

  **Recovery:** Users can run `/welcomebot welcome` in any channel at any time to re-show its welcome message as an ephemeral visible only to them. This command is the reliable fallback and should be mentioned in onboarding documentation.
