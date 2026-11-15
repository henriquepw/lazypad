package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Gamepad struct {
		Vendors  []uint16
		MangoHud Combo
		FPSLimit Combo `toml:"fps_limit"`
	}
	Close struct {
		Processes []string
		Flatpaks  []string
	}
	Modes map[string]Mode
}

type Combo struct {
	Combo []string
	Hold  time.Duration
}

type Mode struct {
	Combo
	Gamescope   []string
	ExitOnTVOff bool `toml:"exit_on_tv_off"`
	Display     Display
	TV          *TV
	Audio       *Audio
}

type Display struct {
	Type    string   // hdmi or virtual
	Output  string   // output that shows the mode
	Disable []string // hdmi: outputs turned off while in the mode
}

type TV struct {
	Type        string // adb or appletv
	Addr        string
	MAC         string        // adb: for Wake-on-LAN
	Input       string        // adb: TvInputInfo id, listed by `dumpsys tv_input`
	App         string        // appletv: bundle id opened on wake
	WakeTimeout time.Duration `toml:"wake_timeout"`
}

type Audio struct {
	Card    string
	Profile string
	Sink    string
}

func Load() (Config, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	_, err = toml.DecodeFile(filepath.Join(dir, "lazypad", "config.toml"), &cfg)

	return cfg, err
}
