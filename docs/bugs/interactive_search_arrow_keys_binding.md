# Bug Report: Interactive Search Arrow Keys Not Scrolling

## Description
The user reported that despite the logic being present for `tea.KeyUp` and `tea.KeyDown`, scrolling down the interactive results list with arrow keys fails in certain terminal/Tmux combinations. 
This occurs because relying strictly on `msg.Type == tea.KeyUp` can sometimes fail to match key events if the terminal sends application cursor keys that Bubbletea interprets strictly via `msg.String()`. Additionally, power users often expect `ctrl+j`/`ctrl+k` or `ctrl+n`/`ctrl+p` bindings to navigate lists dynamically.

## Expected Behavior
- The `Update` loop should switch on `msg.String()` to capture literal `"up"`, `"down"`, `"ctrl+j"`, `"ctrl+k"`, `"ctrl+n"`, and `"ctrl+p"` strings, guaranteeing absolute compatibility across all Tmux environments.
- Arrow keys and Vim/Emacs style navigation should seamlessly move the `m.cursor`.
- This bugfix will bump the version to `v1.2.0`.
