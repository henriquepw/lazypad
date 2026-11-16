package daemon

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/henriquepw/lazypad/internal/audio"
	"github.com/henriquepw/lazypad/internal/compositor"
	"github.com/henriquepw/lazypad/internal/config"
	"github.com/henriquepw/lazypad/internal/display"
	"github.com/henriquepw/lazypad/internal/gamepad"
	"github.com/henriquepw/lazypad/internal/mangohud"
	"github.com/henriquepw/lazypad/internal/proc"
	"github.com/henriquepw/lazypad/internal/steam"
	"github.com/henriquepw/lazypad/internal/tv"
	"github.com/henriquepw/lazypad/internal/uinput"
)

// desktop is the state with no mode active.
const desktop = "desktop"

type request struct {
	cmd     string
	gamepad *gamepad.Gamepad // who triggered the combo, so it can rumble
	reply   chan string
}

type mode struct {
	name    string
	cfg     config.Mode
	display display.Display
	tv      tv.TV // nil when the mode has no tv
}

type daemon struct {
	cfg       config.Config
	modes     map[string]*mode
	current   *mode // nil on the desktop
	prevAudio audio.State
	tvMissing int
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	comp, err := compositor.Detect()
	if err != nil {
		return err
	}

	d := &daemon{cfg: cfg, modes: map[string]*mode{}}
	for name, mc := range cfg.Modes {
		m := &mode{name: name, cfg: mc}
		if m.display, err = display.New(mc.Display, comp); err != nil {
			return fmt.Errorf("mode %s: %w", name, err)
		}
		if mc.TV != nil {
			if m.tv, err = tv.New(*mc.TV); err != nil {
				return fmt.Errorf("mode %s: %w", name, err)
			}
		}
		d.modes[name] = m
	}

	// created up front, so the compositor already knows the device when it is used
	kb, err := uinput.NewKeyboard(uinput.KeyLeftAlt, uinput.Key7)
	if err != nil {
		return err
	}

	requests := make(chan request)
	binds := []gamepad.Bind{
		{
			Combo: cfg.Gamepad.MangoHud.Combo,
			Hold:  cfg.Gamepad.MangoHud.Hold,
			Fire: func(*gamepad.Gamepad) {
				// fails when no game with mangohud is running
				if err := mangohud.Toggle(); err != nil {
					log.Print(err)
				}
			},
		},
		{
			Combo: cfg.Gamepad.FPSLimit.Combo,
			Hold:  cfg.Gamepad.FPSLimit.Hold,
			Fire: func(*gamepad.Gamepad) {
				if err := mangohud.ToggleFPSLimit(kb); err != nil {
					log.Print(err)
				}
			},
		},
	}
	for name, m := range cfg.Modes {
		binds = append(binds, gamepad.Bind{
			Combo: m.Combo.Combo,
			Hold:  m.Hold,
			Fire: func(g *gamepad.Gamepad) {
				requests <- request{cmd: "toggle " + name, gamepad: g}
			},
		})
	}
	if err := gamepad.Watch(cfg.Gamepad.Vendors, binds); err != nil {
		return err
	}

	os.Remove(socketPath())
	ln, err := net.Listen("unix", socketPath())
	if err != nil {
		return err
	}

	defer ln.Close()
	go serve(ln, requests)

	poll := time.NewTicker(10 * time.Second)
	defer poll.Stop()
	for {
		select {
		case req := <-requests:
			reply := d.handle(req)
			if req.reply != nil {
				req.reply <- reply
			}
		case <-poll.C:
			if d.current != nil && d.current.cfg.ExitOnTVOff && d.tvOff() {
				log.Print("TV turned off, going back to the desktop")
				d.leave(nil, true)
			}
		}
	}
}

func Restart() error {
	return proc.Run("systemctl", "--user", "restart", "lazypad")
}

func (d *daemon) handle(req request) string {
	// --no-tv leaves the TV as it is, for when the switch comes from the TV
	// itself, like a stream started or quit in Moonlight
	line, noTV := strings.CutSuffix(req.cmd, " --no-tv")
	cmd, name, _ := strings.Cut(line, " ")
	var want *mode
	switch cmd {
	case "status":
		want = d.current
	case "desktop":
	case "mode", "toggle":
		m, ok := d.modes[name]
		if !ok {
			return "error: unknown mode: " + name
		}
		// toggling the current mode goes back to the desktop
		if cmd == "mode" || m != d.current {
			want = m
		}
	default:
		return "error: unknown command: " + req.cmd
	}

	if want != d.current {
		if d.current != nil {
			d.leave(req.gamepad, !noTV)
		}
		if want != nil {
			d.enter(want, req.gamepad, !noTV)
		}
	}

	if d.current == nil {
		return desktop
	}
	return d.current.name
}

func (d *daemon) enter(m *mode, g *gamepad.Gamepad, useTV bool) {
	if g != nil {
		g.Rumble(1)
	}

	// a physical output only shows up once the TV is on, while a streaming
	// client on the TV connects right away, so it waits for the stream to be ready
	virtual := m.cfg.Display.Type == "virtual"
	if useTV && !virtual {
		d.wake(m)
	}

	if a := m.cfg.Audio; a != nil {
		if prev, err := audio.Current(a.Card); d.must(err) {
			d.prevAudio = prev
		}
	}
	d.must(steam.Stop())
	d.must(m.display.Start())
	if a := m.cfg.Audio; a != nil {
		d.must(audio.Use(a.Card, audio.State{Sink: a.Sink, Profile: a.Profile}))
	}

	for _, name := range d.cfg.Close.Processes {
		exec.Command("pkill", "-x", name).Run()
	}
	for _, app := range d.cfg.Close.Flatpaks {
		exec.Command("flatpak", "kill", app).Run()
	}

	d.must(steam.StartGamescope(m.cfg.Gamescope))
	if useTV && virtual {
		d.wake(m)
	}
	d.current = m
	d.tvMissing = 0
}

func (d *daemon) wake(m *mode) {
	if m.tv == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.TV.WakeTimeout)
	defer cancel()
	if err := m.tv.Wake(ctx); err != nil {
		d.fail(err)
	}
}

func (d *daemon) leave(g *gamepad.Gamepad, useTV bool) {
	m := d.current
	if g != nil {
		g.Rumble(2)
	}
	d.must(steam.Stop())
	steam.StopGamescope()

	d.must(m.display.Stop())
	if a := m.cfg.Audio; a != nil && d.prevAudio != (audio.State{}) {
		d.must(audio.Use(a.Card, d.prevAudio))
	}

	d.must(steam.StartSilent())
	if useTV && m.tv != nil {
		d.must(m.tv.Sleep())
	}
	d.current = nil
}

// tvOff reports whether the TV was turned off with its own remote. No reply
// for 3 polls in a row also counts as off.
func (d *daemon) tvOff() bool {
	if d.current.tv == nil {
		return false
	}

	awake, err := d.current.tv.Awake()
	if err != nil {
		d.tvMissing++
		return d.tvMissing >= 3
	}

	d.tvMissing = 0
	return !awake
}

func (d *daemon) must(err error) bool {
	if err != nil {
		d.fail(err)
	}
	return err == nil
}

func (d *daemon) fail(err error) {
	log.Print(err)
	exec.Command("notify-send", "lazypad", err.Error()).Run()
}
