#!/bin/sh
set -eu
# Optional: after dragging Timo.app into ~/Applications or /Applications
if [ -f "$HOME/Applications/Timo.app/Contents/Resources/timo" ]; then
 app="$HOME/Applications/Timo.app"
elif [ -f "/Applications/Timo.app/Contents/Resources/timo" ]; then
 app="/Applications/Timo.app"
else
 echo 'Move Timo.app into ~/Applications or /Applications first.' >&2; exit 1
fi
mkdir -p "$HOME/.local/bin"
ln -sf "$app/Contents/Resources/timo" "$HOME/.local/bin/timo"
echo 'Terminal command installed: ~/.local/bin/timo'
echo 'Add ~/.local/bin to your PATH if necessary.'
