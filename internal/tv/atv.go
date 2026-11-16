package tv

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// appleTV controls an Apple TV through atvremote, from pyatv. It needs the
// companion protocol paired once with `atvremote -s <addr> --protocol companion pair`.
type appleTV struct {
	Addr   string
	App    string   // bundle id opened on wake
	Remote []string // atvremote commands sent once the app is open, like select or delay=2000
}

func (t appleTV) Wake(ctx context.Context) error {
	for {
		_, err := t.output("turn_on", "launch_app="+t.App)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Apple TV did not answer at %s: %w", t.Addr, err)
		case <-time.After(time.Second):
		}
	}

	if len(t.Remote) == 0 {
		return nil
	}
	// the delays between the keys add up, so it gets more time than a single command
	_, err := t.run(30*time.Second, t.Remote...)
	return err
}

func (t appleTV) Sleep() error {
	_, err := t.output("turn_off")
	return err
}

func (t appleTV) Awake() (bool, error) {
	out, err := t.output("power_state")
	if err != nil {
		return false, err
	}

	return strings.Contains(out, "PowerState.On"), nil
}

func (t appleTV) output(args ...string) (string, error) {
	return t.run(10*time.Second, args...)
}

func (t appleTV) run(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args = append([]string{"-s", t.Addr}, args...)
	out, err := exec.CommandContext(ctx, "atvremote", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("atvremote %s: %w: %s", strings.Join(args, " "), err, out)
	}

	return string(out), nil
}
