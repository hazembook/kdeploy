# Safe: fake virsh, default No. The VM and leftover files are left untouched.
Output docs/media/destroy-80.gif
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
Type "demo/setup.sh destroy"
Enter
Type "export HOME=/var/tmp/kdeploy-demo/home PATH=/var/tmp/kdeploy-demo/bin:$PATH"
Enter
Type "clear"
Enter
Show
Type "bin/kdeploy destroy demo --config /var/tmp/kdeploy-demo/config/destroy.yaml"
Enter
Wait+Screen /Destroy VM/
Sleep 1s
Screenshot docs/media/destroy-80.png
Sleep 1s
Enter
Sleep 2s
