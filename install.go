package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/henriquepw/lazypad/internal/proc"
)

//go:embed lazypad.service
var unit string

// install writes the systemd user unit, pointing at this binary.
func install() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := strings.ReplaceAll(unit, "@BIN@", bin)
	if err := os.WriteFile(filepath.Join(dir, "lazypad.service"), []byte(content), 0o644); err != nil {
		return err
	}

	return proc.Run("systemctl", "--user", "daemon-reload")
}
