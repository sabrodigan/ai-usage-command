# Omarchy Todo

A local command-oriented todo app and Quickshell status bar widget. Requires Python 3 (standard library only) and Omarchy’s Quickshell bar.

```bash
python3 install.py
todo add "Send invoice" --priority high
todo add "Book appointment" --due "08/09/2026"
todo list
todo list --today
todo complete 1 --outcome "Outcome text"
todo reopen 1
todo change 2 --title "Book dentist" --due 2026-09-12
todo priority 2 high
todo delete 2
```

Run commands from this folder to install, or use `./bin/todo` without installation. Ensure `~/.local/bin` is on PATH. Installation backs up `shell.json`, preserves the existing layout, and adds the widget once. User plugins hot-reload.

Tasks default to today and normal priority. Dates accept `today`, `tomorrow`, `DD/MM/YYYY` (day/month/year), `YYYY-MM-DD`, or `none` (unscheduled). Priorities are `low`, `normal`, and `high`. `change` also accepts `--priority`; `edit` is an alias. IDs are permanent and never reused. Delete is permanent; complete/reopen preserve a task. Marking a task done records the local date and time plus an optional outcome; reopening clears both.

The badge always shows three counts: **all open tasks** in the default bar colour, **due today** in green, and **overdue** in red, separated by dots. Zero counts remain visible. Total open includes future and unscheduled tasks; today and overdue use the local date and exclude unscheduled tasks. Completed tasks are excluded from all counts. Click to open the interactive CRUD menu in a terminal. Hovering the widget shows the app name, version, and author in a tooltip; the terminal window header shows them too. Mutations request an immediate IPC refresh; five-second polling provides a fallback and date rollover. An exclamation mark indicates a read error.

`todo list --all` includes completed tasks; `--overdue` filters overdue tasks; `--json` provides structured output. `todo status` emits JSON counts.

Data lives in `$XDG_DATA_HOME/omarchy-todo/tasks.sqlite3` (default `~/.local/share/omarchy-todo/tasks.sqlite3`). SQLite transactions protect concurrent edits. Back up this database to preserve tasks. `TODO_DB` overrides its location for testing; the CLI and shell must share that environment to use a custom database.

Run tests: `python3 -m unittest discover -s omarchy-todo` from the parent folder.

To remove the widget, run `omarchy plugin disable sabrodigan.todo`. Task data is retained.

## Interactive window

Click the bar count or run `todo interactive`. First choose a todo number from the list (`C` creates a new task instead, Enter closes), then work with that todo through the CRUD menu. Press a menu letter (no Enter needed):

- **C — Create:** prompts for title, due date (default today), and priority (default normal). Returns to the current todo afterwards.
- **R — Read:** types out the todo's details with a blinking block cursor. Enter returns to the menu. Animation requires a terminal that supports cursor styling; redirected output is plain text.
- **U — Update:** asks status first. Choosing done asks for an outcome (Enter leaves it blank) and records the date and time; choosing open prompts for title, date, then priority in order. Enter retains each current value.
- **D — Delete:** asks `Delete this todo? [Y/n]`. Enter confirms deletion; `n` cancels. After a deletion the refreshed list appears for the next selection.
- **B — Back to list:** returns to the number prompt to choose another todo. Enter at the menu closes the window.

IDs are the numbers shown in the list. Enter alone at the number prompt closes the window. Ctrl+C cancels an action without saving partial changes. Dates accept `DD/MM/YYYY`, `YYYY-MM-DD`, `today`, `tomorrow`, and `none`.
