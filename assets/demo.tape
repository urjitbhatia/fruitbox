# VHS tape for the fruitbox demo GIF.
#   brew install vhs   &&   vhs assets/demo.tape
# Assumes `fruitbox` is on PATH and a demo project lives in /tmp/fbdemo.

Output assets/demo.gif

Set Shell "zsh"
Set FontSize 18
Set Width 1200
Set Height 800
Set Padding 24
Set Theme "Catppuccin Mocha"

# Hidden setup: clean prompt, put fruitbox on PATH, enter the demo project.
Hide
Type `export PATH="$(go env GOPATH)/bin:$PATH"; export PS1="$ "; cd /tmp/fbdemo; clear`
Enter
Sleep 500ms
Show

Type "cat compose.yaml"
Sleep 600ms
Enter
Sleep 2s

Type "fruitbox up -d"
Sleep 600ms
Enter
Sleep 6s

Type "fruitbox ps"
Sleep 600ms
Enter
Sleep 3s

Type "fruitbox logs --tail 6"
Sleep 600ms
Enter
Sleep 3500ms

Type "fruitbox down"
Sleep 600ms
Enter
Sleep 4s
