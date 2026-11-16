// lazypad switches the PC between the desktop and the couch gaming modes set in
// its config, each one with its own display, TV and gamepad combo.
package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/henriquepw/lazypad/internal/daemon"
)

const help = `usage: lazypad <command>

commands:
  daemon         run the daemon (reads the gamepads and listens on the socket)
  restart        restart the daemon
  install        install the systemd user unit for this binary
  mode <name>    enter the mode
  toggle <name>  toggle between the mode and the desktop
  desktop        go back to the desktop
  status         show the current mode

mode, toggle and desktop take --no-tv to leave the TV as it is
`

func main() {
	args := os.Args[1:]
	noTV := slices.Contains(args, "--no-tv")
	args = slices.DeleteFunc(args, func(a string) bool { return a == "--no-tv" })
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
	}

	var err error
	switch {
	case cmd == "daemon":
		err = daemon.Run()
	case cmd == "restart":
		err = daemon.Restart()
	case cmd == "install":
		err = install()
	case (cmd == "mode" || cmd == "toggle") && len(args) == 2,
		(cmd == "desktop" || cmd == "status") && len(args) == 1:
		line := strings.Join(args, " ")
		if noTV {
			line += " --no-tv"
		}
		var reply string
		if reply, err = daemon.Send(line); err == nil {
			fmt.Println(reply)
		}
	default:
		fmt.Print(help)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "lazypad:", err)
		os.Exit(1)
	}
}
