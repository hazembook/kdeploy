# Safe: fake virsh/runtime tools, disposable paths, and Cancel at review.
Output docs/media/deploy-120.gif
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
Screenshot docs/media/deploy-120-resources.png
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
Screenshot docs/media/deploy-120-review.png
Sleep 1s
Down 3
Enter
Sleep 2s
