# Safe: lists the catalog and opens the picker, then cancels. No download starts.
Output docs/media/download-80.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 18
Set Columns 80
Set Rows 24
Set Padding 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 25ms
Set WaitTimeout 15s

Hide
Type "demo/setup.sh deploy"
Enter
Type "clear"
Enter
Show
Type "bin/kdeploy download --list"
Enter
Sleep 2s
Type "clear"
Enter
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/demo.yaml download"
Enter
Wait+Screen /Download a cloud image/
Sleep 1s
Screenshot docs/media/download-80.png
Sleep 1s
Ctrl+C
Sleep 2s
