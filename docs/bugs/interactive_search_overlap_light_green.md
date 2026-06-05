# Bug Report: Interactive Search Line Overlap and Light Green Highlight

## Description
The user reported that on certain search results, the session name visually overlapped with the matched string. This is typically caused by unescaped Tab (`\t`) or Backspace (`\b`) characters in the snippet confusing the Lipgloss terminal width calculations, resulting in improper padding and terminal wrapping.
Additionally, the user requested that the active row selection highlight color be changed to "light green", maintaining the full row width expansion from `v1.3.0`.

## Expected Behavior
- Update `cleanSnippet()` to completely strip out or substitute `\t` and `\b` control characters before rendering, guaranteeing accurate string width calculations.
- Update `selectedStyle` in the interactive pane to use a light green background (`lipgloss.Color("10")` or `"119"`) with a high-contrast foreground, expanding across the entire width of the terminal.
- This UI fix triggers a version bump to `v1.7.0`.
