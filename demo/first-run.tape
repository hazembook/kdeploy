# Safe: completes the first-run form, then cancels at the image picker.
Output docs/media/first-run.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 20
Set Columns 120
Set Rows 32
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set WaitTimeout 15s

Hide
Type "export HOME=/var/tmp/kdeploy-demo/home PATH=/var/tmp/kdeploy-demo/bin:$PATH"
Enter
Type "demo/setup.sh first-run"
Enter
Type "clear"
Enter
Show
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/demo.yaml deploy demo"
Enter
Wait+Screen /Image library/
Sleep 1s
Screenshot docs/media/first-run-setup.png
Down 2
Enter
Sleep 1s
Down 2
Enter
Sleep 1s
Type "/var/tmp/kdeploy-demo/images"
Enter
Sleep 1s
Type "/var/tmp/kdeploy-demo/disks"
Enter
Wait+Screen /Choose the cloud image/
Sleep 1s
Screenshot docs/media/first-run-ready.png
Sleep 1s
Ctrl+C
Sleep 2s
