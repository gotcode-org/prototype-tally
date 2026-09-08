# Tally: The Time-Tracking Command Line

Modern task managers are heavy, bloated Electron apps that trap your data inside proprietary databases. **Tally** breaks the mold by storing your entire life in an infinitely scalable, sharded directory of human-readable Markdown files. 

Whether you want to quickly clock-in to a task from your bash prompt using the CLI, or launch the interactive Terminal User Interface (TUI) to visualize your daily, weekly, and monthly productivity, Tally handles it with zero network requests—acting as a bulletproof local cache for your enterprise workflow.

## Core Features

### Sharded File Storage
Zero databases. Every task is its own Markdown file organized in a `YYYY/MM/DD` directory tree. This guarantees infinite scalability and absolutely zero Git merge conflicts when syncing across devices.

### Enterprise Offline Sync
Survive cloud outages. Create tasks and track time entirely locally. When the internet returns, run `tally sync` to automatically push your hours to the **7pace TimeTracker API** and generate stories in **Azure DevOps**.

### Sequential Task IDs
Every file is sequentially named, granting every task a pristine local ID (e.g., `20260831.001`). Instantly clock-in from the terminal by running `tally start 20260831.001`.

### Dynamic TUI Dashboards
Launch the full-screen Bubbletea TUI and use hotkeys to instantly toggle between **Day View**, **Week View**, and **Month View** to track your long-term velocity and project burn rates.

### Reusable Component Architecture
The internal UI layer is built on 100% generic, data-driven Form and List components. Business logic is strictly separated via a flat DDD/CQRS architecture in the `internal/` directory.

### Cascading Configuration
Respects standard XDG Base Directories while allowing per-project `.tally/config.yaml` overrides. Seamlessly switch ADO Area Paths just by changing directories in your terminal.
