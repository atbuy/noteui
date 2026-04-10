Output demo/noteui-demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1920
Set Height 1080
Set TypingSpeed 60ms
Set Theme "Dracula"
Set Padding 20
Set BorderRadius 8

# Setup a clean prompt without cluttering the recording
Hide
Type "export PS1='$ '"
Enter
Type "clear"
Enter
Show

# Launch noteui in demo mode
Type "noteui --demo"
Sleep 800ms
Enter

# Wait for the TUI to initialize and discover the sample notes
Sleep 2500ms

# Initial state: cursor is on the synthetic "/" root. The tree shows
# three collapsed categories (personal, reference, work) followed by
# two root-level notes (inbox.md, journal.md). The preview pane renders
# the selected item.
#
# Step through each category. The preview updates to show category
# metadata (subcategory and note counts) for each row.
Down
Sleep 900ms
Down
Sleep 900ms
Down
Sleep 900ms

# Expand the "work/" category with the right arrow. The two nested
# notes (meeting-notes.md, project-alpha.md) appear beneath it and the
# cursor remains on the work/ row.
Right
Sleep 1s

# Move into the expanded category. The preview now renders the actual
# note content for meeting-notes.md.
Down
Sleep 2s

# Next note: project-alpha.md. Preview re-renders.
Down
Sleep 2200ms

# Root-level notes follow the expanded work/ children.
Down
Sleep 1800ms

Down
Sleep 1800ms

# Switch focus to the preview pane with Tab. In preview focus, j/k and
# the arrow keys scroll the preview instead of moving the tree cursor.
Tab
Sleep 700ms

Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 700ms

# Escape returns focus to the tree.
Escape
Sleep 800ms

# Fuzzy search: "/" opens the search input. Typing narrows the tree
# in real time.
Type "/"
Sleep 500ms
Type "project"
Sleep 1800ms

# First Escape exits the search input but keeps the filter applied.
Escape
Sleep 1200ms

# Second Escape clears the filter and restores the full tree.
Escape
Sleep 1200ms

# Toggle into the temporary notes view with "t". The .tmp/scratch.md
# fixture appears in its dedicated pane.
Type "t"
Sleep 2s

# Toggle back to the main notes tree.
Type "t"
Sleep 900ms

# Open the help modal with "?". This overlays the full keybinding
# reference organised by action group.
Type "?"
Sleep 2500ms

# Escape closes the help modal.
Escape
Sleep 800ms

# Quit noteui cleanly with "q". The demo temp directory is removed
# automatically on exit.
Type "q"
Sleep 500ms
