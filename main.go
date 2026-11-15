// lazypad switches the PC between the desktop and the couch gaming modes set in
// its config, each one with its own display, TV and gamepad combo.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/henriquepw/lazypad/internal/daemon"
)

const help = `usage: lazypad <command>

commands:
  daemon         run the daemon (reads the gamepads and listens on the socket)
  install        install the systemd user unit for this binary
  mode <name>    enter the mode
  toggle <name>  enter the mode, or go back to the desktop when already in it
  desktop        go back to the desktop
  status         show the current mode
`

func main() {
	args := os.Args[1:]
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
	}

	var err error
	switch {
	case cmd == "daemon":
		err = daemon.Run()
	case cmd == "install":
		err = install()
	case (cmd == "mode" || cmd == "toggle") && len(args) == 2,
		(cmd == "desktop" || cmd == "status") && len(args) == 1:
		var reply string
		if reply, err = daemon.Send(strings.Join(args, " ")); err == nil {
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
