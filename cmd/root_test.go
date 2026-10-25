package cmd

import "testing"

func TestConfigureCommandGroups(t *testing.T) {
	configureCommandGroups()

	want := map[string]string{
		"deploy":   "vm",
		"destroy":  "vm",
		"start":    "vm",
		"shutdown": "vm",
		"stop":     "vm",
		"reboot":   "vm",
		"list":     "inspect",
		"info":     "inspect",
		"ip":       "inspect",
		"console":  "inspect",
		"edit":     "inspect",
		"download": "images",
		"config":   "config",
		"check":    "config",
	}

	for _, command := range rootCmd.Commands() {
		group, ok := want[command.Name()]
		if !ok {
			continue
		}
		if command.GroupID != group {
			t.Errorf("%s group = %q, want %q", command.Name(), command.GroupID, group)
		}
		delete(want, command.Name())
	}
	for name := range want {
		t.Errorf("command %q not found", name)
	}
}
