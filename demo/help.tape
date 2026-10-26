# Safe: prints version and help; touches no VM state.
Output docs/media/help.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 20
Set Columns 120
Set Rows 34
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set WaitTimeout 15s

Type "bin/kdeploy --version"
Enter
Sleep 500ms
Type "bin/kdeploy --help"
Enter
Wait+Screen /VM LIFECYCLE/
Sleep 2s
