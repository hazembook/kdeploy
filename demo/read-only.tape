# Safe: uses a fake virsh and disposable config. No real VM is touched.
Output docs/media/read-only.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 20
Set Columns 120
Set Rows 38
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set WaitTimeout 15s

Hide
Type "demo/setup.sh read-only"
Enter
Type "export HOME=/var/tmp/kdeploy-demo/home PATH=/var/tmp/kdeploy-demo/bin:$PATH"
Enter
Type "clear"
Enter
Show
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/read-only.yaml list"
Enter
Sleep 1s
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/read-only.yaml info demo"
Enter
Sleep 1s
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/read-only.yaml ip demo"
Enter
Sleep 1s
Type "bin/kdeploy --config /var/tmp/kdeploy-demo/config/read-only.yaml config show"
Enter
Sleep 2s
