# vhs tape for the TUI screenshot/GIF. Regenerate with:  vhs docs/demo-tui.tape
# Requires: vhs (brew install vhs) and the extension installed (make install).
Output docs/demo-tui.gif

Require gh

Set Shell "bash"
Set FontSize 15
Set Width 1320
Set Height 800
Set Padding 18

Hide
Type "gh repos-hud --demo"
Enter
Sleep 2.5s
Show

# Static hero screenshot of the default attention-first view.
Sleep 800ms
Screenshot docs/demo-tui.png

# Demonstrate live interaction for the GIF.
Type "a"          # attention-only: hide green repos
Sleep 1.5s
Type "a"          # back to all
Sleep 1s
Type "/"          # filter
Type "pay"
Sleep 1.8s
Escape
Sleep 1s
Type "q"
Sleep 500ms
