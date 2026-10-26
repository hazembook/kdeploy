# Safe: fake virsh/runtime tools, disposable paths, and Cancel at review.
Output docs/media/deploy-80.gif
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
Type "export HOME=/var/tmp/kdeploy-demo/home PATH=/var/tmp/kdeploy-demo/bin:$PATH"
Enter
Type "demo/setup.sh deploy"
Enter
Type "clear"
Enter
Show
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/demo.yaml deploy demo"
Enter
Wait+Screen /Choose the cloud image/
Sleep 1s
Enter
Sleep 2s
Screenshot docs/media/deploy-80-resources.png
Enter
Sleep 1s
Enter
Sleep 1s
Enter
Sleep 1s
Enter
Sleep 1s
Enter
Sleep 1s
Enter
Sleep 2s
Screenshot docs/media/deploy-80-review.png
Sleep 1s
Down 3
Enter
Sleep 2s
