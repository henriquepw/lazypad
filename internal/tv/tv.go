package tv

import (
	"context"
	"fmt"

	"github.com/henriquepw/lazypad/internal/config"
)

type TV interface {
	// Wake turns the TV on and switches to the PC input or app.
	Wake(ctx context.Context) error
	Sleep() error
	Awake() (bool, error)
}

func New(cfg config.TV) (TV, error) {
	switch cfg.Type {
	case "adb":
		return adb{Addr: cfg.Addr, MAC: cfg.MAC, Input: cfg.Input}, nil
	case "appletv":
		return appleTV{Addr: cfg.Addr, App: cfg.App, Remote: cfg.Remote}, nil
	default:
		return nil, fmt.Errorf("unknown tv type: %q", cfg.Type)
	}
}
