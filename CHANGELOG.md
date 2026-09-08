# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Native UI Sync Engine**: The `sync` (`s`), `fetch` (`p`), and `push_single` (`u`) operations now run entirely within the native Bubbletea event loop instead of suspending the TUI to run a raw bash shell.
- **Standup Finished Items**: The `tally standup` CLI and TUI report generator now explicitly loops in tasks marked as Closed, Done, Resolved, or Completed within the last 48 hours.
- **Scrollable TUI Modals**: The Standup Report (`r`) and the new Version dashboard (`v`) open in compact, centered UI modals with full vertical scrolling (`up`/`down`, `pgup`/`pgdown`, `j`/`k`).
- **Build Version Metadata**: Added a `tally version` command and `v` TUI hotkey. The Makefile now automatically injects `VERSION`, `COMMIT`, and `BRANCH` via LDFLAGS, and the Go binary dynamically exposes its embedded dependency tree.
- **Isolated Single-Task Sync (`u`)**: Added a targeted push feature mapped to `u` in the TUI (or `tally push <id>`) that instantly syncs a single task without incurring a full filesystem scan.
- **Corporate Alias Configuration**: Added `email` to the `7pace` block in `config.yaml` to gracefully handle user environments with differing ADO/Git and 7pace domain logins.
- **Tally Debug Task CLI**: Added `tally debug-task <id>` to instantly dump raw local task structs and yaml frontmatter to the console for troubleshooting.
- **Missing Parent Discovery**: If a fetched child task references an ADO parent outside the 14-day rolling window, the engine fires an on-demand API request to fetch the parent story so the hierarchy never breaks.
- **Historical 7pace Import**: Tally now fetches the user's historical 7pace logs during an ADO pull to reconstruct `TotalSeconds` and accurately portray past time.
- **Interactive Deletion Prompt**: Replaced the immediate `x` deletion with an inline footer prompt requiring explicit `y`/`n` confirmation before destroying data.
- **Fast Scroll Keybindings**: Added `Page Up` and `Page Down` keys (`pgup` / `pgdown`) to leap the cursor exactly one viewport height at a time.
- **Automatic Story Sizing**: Tally automatically defaults empty ADO tasks to `story_points: 1.0` when fetching to satisfy strict ADO Agile state transition rules upon pushing.

### Changed
- **Sync Visuals**: Replaced the raw terminal log wall during sync operations with an animated, indeterminate bouncing progress bar and a 1-line real-time status output.
- **Create Form Streamlined**: Stripped the redundant "Backlog?" toggle entirely from the UI and CLI; tasks seamlessly start as `New`.

### Fixed
- **UI Progress Bar Panic**: Fixed a runtime panic caused by `strings.Repeat` receiving a negative width when the sync modal caught an API error and forced a 100% fill override.
- **ANSI Color Bleed**: Forced explicit background colors on raw spaces within modals to prevent internal reset codes (`\x1b[0m`) from stripping backgrounds and bleeding the default terminal colors.
- **Massive Scrolling CPU Spikes**: Refactored the UI engine to cache flattened tree structures and pre-render expensive Lipgloss ANSI blocks. Drops CPU consumption during rapid key repeats from 100% to near 0%.
- **Cursor Reset Bug**: The TUI cursor no longer wildly jumps back to the top (index 0) of the list after deleting a task, gracefully falling back to its nearest available index.
- **UI State Persistence Loss**: Fixed a bug where Archive folders (Year, Month, Day) silently collapsed during UI refreshes due to tracking string path mismatches.
- **OData Silent Query Failures**: Implemented a strict hardcoded Go validation pass against all 7pace API logs. If the 7pace API OData `&$filter` silently drops bounds, Tally protects local time calculations from runaway inflation.
- **Bulk Fetch Sequence Collisions**: The ID generation engine now dynamically tracks in-memory sequence numbers during multi-task fetches to completely eliminate task overwriting (`.001` collisions).
- **Missing Single-Push Payload Fields**: Injected missing Story Points, Swimlanes, and Hierarchy relation structures into the targeted `u` command's JSON payload so ADO stops rejecting single-task updates.

## [0.1.0] - 2026-09-02

### Added
- **Native UI Sync Engine**: The `sync` (`s`), `fetch` (`p`), and `push_single` (`u`) operations now run entirely within the native Bubbletea event loop instead of suspending the TUI to run a raw bash shell.
- **Standup Finished Items**: The `tally standup` CLI and TUI report generator now explicitly loops in tasks marked as Closed, Done, Resolved, or Completed within the last 48 hours.
- **Scrollable TUI Modals**: The Standup Report (`r`) and the new Version dashboard (`v`) open in compact, centered UI modals with full vertical scrolling (`up`/`down`, `pgup`/`pgdown`, `j`/`k`).
- **Build Version Metadata**: Added a `tally version` command and `v` TUI hotkey. The Makefile now automatically injects `VERSION`, `COMMIT`, and `BRANCH` via LDFLAGS, and the Go binary dynamically exposes its embedded dependency tree.
- **ADO Sync Debug Toggle**: Added `ado.debug: true` flag to `config.yaml` to optionally enable verbose JSON telemetry dumps during syncs (silenced by default).
- **5 New Color Themes**: Added Gruvbox, Tokyo Night, Rosé Pine, Monokai, and Solarized Dark to the TUI's rotation (press `t` to cycle through them).
- **TUI Time Summary**: Added live granular time logging summary counters (Day, Week, Month, Year) to the footer of the TUI dashboard.
- **Dynamic Swimlane TUI Dropdown**: Added a dynamic dropdown to the TUI "Create Task" modal to select ADO swimlanes, driven by a new `ado.swimlanes` array in `config.yaml`.
- **Default Swimlane Auto-Routing**: Added `ado.default_swimlane` configuration setting so newly created tasks automatically route to the correct ADO board lane upon creation.
- **ADO Swimlane Support**: Users can now push tasks directly into specific board lanes by setting `ado.swimlane_field` in their config and `swimlane` in their markdown frontmatter.
- **Tally Debug CLI**: Introduced `tally debug [ado_id]` command to fetch and dump raw JSON payloads from the ADO REST API, specifically helping users hunt down their custom `WEF_*_Kanban.Lane` field names.
- **ADO Hierarchies**: Full parent-child relationship support. Subtasks can be created directly from the TUI by highlighting a parent Story and pressing `c`.
- **Dashboard Nesting**: Subtasks visually render indented underneath their parent Story on the TUI dashboard tree.
- **Granular Time Logging**: Core engine upgraded to store discrete `TimeLog` entries instead of a single merged integer, allowing independent syncing of time entries to 7pace with dynamic Activity IDs.
- **Dynamic Activity Types**: The TUI 'Log Time' dialog now automatically populates an Activity Type dropdown based on the `7pace.activities` map in `~/.config/tally/config.yaml`.
- **System.LinkTypes.Hierarchy-Reverse Patching**: The ADO sync engine dynamically links child tasks to parent stories in the ADO database during sync sweeps.
- Implemented full Task Deletion support with a new `tally delete [id]` CLI command and an `x` keybinding in the TUI.
- Integrated a new interactive `Log Time` dialog directly into the TUI (via the `a` keybinding) to rapidly log time without exiting the dashboard.
- Overhauled the Create Task screen into a clean, horizontally-centered floating dialog box.
- Enclosed text input fields in the dialog inside heavily styled, inset rounded borders.
- Re-aligned dialog form fields and removed redundant help text for a more professional layout.
- Added dynamic bottom-padding to the Dashboard to maintain the UI grid structure all the way to the footer, even when empty.
- Stripped bulky background badges from Type and Status columns in the list view to reduce visual noise.
- Integrated generic Bubbletea Form/List framework into a fully functional `tally ui` dashboard.
- Implemented `tally points` and automatic Story Point defaults to satisfy strict Azure DevOps state transition rules.
- Upgraded the Sync Engine to dynamically parse Markdown `# Description` and `# Acceptance Criteria` headers into native ADO fields.
- Implemented chronological Tree rendering in the Dashboard UI (Year -> Month -> Day).
- Added `tea.ExecProcess` keybindings in the TUI to launch `$EDITOR` (`e`) and execute `tally sync` (`s`) seamlessly.
- Added `tally edit` and `tally state` CLI commands.
- Configured dynamic calculation of time logged replacing static UI progress bars.
- Created `Makefile` with robust `PREFIX` and `DESTDIR` support for native Linux installation.
- Core Domain Model (`internal/core/task.go`) with Azure DevOps and 7pace time-tracking fields.
- Initial project directory skeleton matching a flat DDD/CQRS architecture.
- Standardized `GPL-3.0` License headers applied to all Go source files.

### Fixed
- **UI Progress Bar Panic**: Fixed a runtime panic caused by `strings.Repeat` receiving a negative width when the sync modal caught an API error and forced a 100% fill override.
- **ANSI Color Bleed**: Forced explicit background colors on raw spaces within modals to prevent internal reset codes (`\x1b[0m`) from stripping backgrounds and bleeding the default terminal colors.
- **ADO Update Payload Drops**: Swimlanes, Tags, and Story Points are now properly injected into the ADO JSON patch payload during *updates* (previously they only triggered on task creation).
- **ADO WYSIWYG Editor Swallowing**: Markdown body output is now wrapped in explicit HTML `<div>` elements during ADO syncs to prevent Azure DevOps from silently dropping bare text strings in rich text fields.
- **Recurrence Scheduling Rules**: Overhauled the Reconciliation engine to properly evaluate ISO weeks and calendar boundaries for `weekly`, `monthly`, and `weekdays` rules instead of cloning everything daily.
- **Ghost Spawns**: Forced the Reconciliation Engine to fire instantly upon form submission so newly created templates spawn clones immediately without requiring an app restart.
- **Template Sync Exclusion**: Explicitly blocked raw Master Templates (`recur-*`) from syncing to ADO and 7pace to prevent phantom timesheets and ghost work items.
- Fixed critical TUI transparency bugs where ANSI reset codes (`\x1b[0m`) emitted by Bubbletea text inputs and borders would shatter the UI background and expose the terminal's native background.
- Resolved severe contrast issues when cycling themes (specifically in Nord) by dynamically checking the active row's state and inverting text highlights when necessary.
- Decoupled column separators from dynamic row styling to ensure grid lines draw consistently down the screen.
