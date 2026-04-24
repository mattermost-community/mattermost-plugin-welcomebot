# Changelog

The changelog can be found at https://github.com/mattermost/mattermost-plugin-welcomebot/releases.

## v1.4.0 — 2026-04-24

### Changed

- **Channel welcome messages no longer send a bot DM.** Previously, joining a channel triggered both a DM from Welcomebot and an ephemeral post. The DM was redundant with the team-join welcome and noisy when joining multiple channels at once. Channel welcomes now deliver as an ephemeral post only. Team-join welcomes are unaffected and still deliver via DM.

### Improved

- **Channel welcome ephemeral delivery is now asynchronous.** The ephemeral fires in a background goroutine after a configurable delay so the `UserHasJoinedChannel` hook returns immediately.
- **Plugin-initiated auto-joins now deliver the channel welcome directly.** When the welcomebot adds a user to channels via `automatic` actions on team join, Mattermost does not re-fire `UserHasJoinedChannel` back to the calling plugin. The welcome ephemeral is now sent from `joinChannel` itself, ensuring delivery for the team-join auto-add flow.

### Added

- **`ChannelWelcomeAutoJoinDelaySeconds` setting.** Controls how long the plugin waits before sending the channel welcome ephemeral. Configurable from System Console → Plugins → Welcome Bot. Defaults to 5 seconds. Applies to all join types.
- **`/welcomebot welcome` command.** Any user can run this in any channel to re-show that channel's welcome message as an ephemeral visible only to them. The primary recovery path for missed welcomes — see known limitations below.
- **`POST /admin/set_channel_welcome` endpoint.** Allows system admins to set channel welcome messages programmatically. Accepts `{"channel_id": "...", "message": "..."}`. Callers authenticate to the Mattermost server using `Authorization: Bearer <token>` (a personal access token or session token); Mattermost authenticates the request and injects `Mattermost-User-Id`, which the plugin uses to verify system admin role. Intended for setup scripts and administrative automation.
- **`SiteURL` added to `MessageTemplate`.** Templates can now reference `{{.SiteURL}}` in welcome messages.

### Fixed

- **`{{.SiteURL}}` rendered empty in button-action welcome messages.** The interactive button action flow in `ServeHTTP` never populated `SiteURL` on the message template. Fixed by setting `data.SiteURL = p.getSiteURL()` alongside the other template fields.
- **Nil panic in `ServeHTTP` action decoder.** When the request body decoded successfully but produced a nil action (e.g. JSON null payload), the error handler called `err.Error()` on a nil error. Added a nil guard.
- **`appErr` rendered as raw pointer in command responses.** Passing `*model.AppError` directly to a `%s` format verb produced `%!s(*model.AppError=...)`. Fixed across all call sites in `command.go` to use `appErr.Error()`.
- **Help text referred to "Direct channels" instead of "Private channels".** The `set_channel_welcome` command rejects private channels. The help string now matches the actual restriction.
- **`plugin.json` setting default encoded as string instead of number.** `ChannelWelcomeAutoJoinDelaySeconds` declared `type: number` but `default` was `"5"`. Changed to `5`.
- **`TestFilterLogEntries/filter_out_old_entries` was flaky on CI.** Two root causes: `filterLogEntries` used `Before(since)` which let entries at exactly `since` pass through (changed to `!After(since)`), and the test used repeated `time.Now()` calls that drifted from the `since` baseline on slow machines (anchored all timestamps to the same `now` variable).

### Maintenance

- **Updated `golangci-lint` from `v1.51.1` to `v1.64.8` with Go version fallback.** `v1.64.8` requires Go 1.23+ and cannot be installed via `go install` on CI (Go 1.21.13, `GOTOOLCHAIN=local`). The Makefile tries `go install` first; on failure it downloads the pre-built binary from the GitHub release tarball.
- **Removed fully-inactivated linters from `.golangci.yml`.** `deadcode`, `golint`, `interfacer`, `structcheck`, and `varcheck` caused exit code 7 in newer golangci-lint. Replaced `golint` with `revive`; the others are covered by `unused`.
- **Replaced deprecated `check-shadowing` govet option** with an explicit `disable: [shadow]` entry.
- **Fixed `goimports local-prefixes`** — was set to `mattermost-plugin-autolink` (copy-paste error). Updated to `mattermost-plugin-welcomebot`.
- **Renamed unused interface parameters** `c *plugin.Context` and `actor *model.User` to `_` in hook and HTTP handler signatures to satisfy `revive`.

### Known Limitations

- **Channel welcome ephemerals are reliable on first join, but not on rejoin.** On a user's first-ever join to a team, the auto-added channels are cold in the client — no cached state exists. When the user navigates to each channel, the ephemeral is still live in the WebSocket buffer and renders correctly. On subsequent rejoins (leave team → rejoin), the client already has those channels in its local store and rehydrates the post list from the server. Because the server never stores ephemerals, the welcome is gone before the user opens the channel.

  This is a platform limitation. Mattermost does not provide a `UserHasViewedChannel` hook or any equivalent event that would allow the plugin to trigger delivery at the moment the user first opens a channel. There is no workaround that guarantees delivery on rejoin without user action.

  **Recovery:** Users can run `/welcomebot welcome` in any channel at any time to re-show its welcome message as an ephemeral visible only to them. This command is the reliable fallback and should be mentioned in onboarding documentation.
