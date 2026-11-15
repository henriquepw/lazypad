// Package display puts a mode on screen.
package display

import (
	"fmt"

	"github.com/henriquepw/lazypad/internal/compositor"
	"github.com/henriquepw/lazypad/internal/config"
)

type Display interface {
	// Start turns the output on and makes new windows open on it.
	Start() error
	// Stop puts the outputs back as they were.
	Stop() error
}

func New(cfg config.Display, comp compositor.Compositor) (Display, error) {
	switch cfg.Type {
	case "hdmi":
		return hdmi{comp: comp, output: cfg.Output, disable: cfg.Disable}, nil
	case "virtual":
		return virtual{comp: comp, output: cfg.Output}, nil
	default:
		return nil, fmt.Errorf("unknown display type: %q", cfg.Type)
	}
}

// hdmi swaps the desk outputs for a physical one. With them disabled, every
// window ends up on it.
type hdmi struct {
	comp    compositor.Compositor
	output  string
	disable []string
}

func (h hdmi) Start() error {
	if err := h.comp.Enable(h.output); err != nil {
		return err
	}
	for _, out := range h.disable {
		if err := h.comp.Disable(out); err != nil {
			return err
		}
	}
	return nil
}

func (h hdmi) Stop() error {
	for _, out := range h.disable {
		if err := h.comp.Enable(out); err != nil {
			return err
		}
	}
	return h.comp.Disable(h.output)
}

// virtual adds a headless output for streaming, leaving the desk outputs on.
type virtual struct {
	comp   compositor.Compositor
	output string
}

func (v virtual) Start() error {
	if err := v.comp.CreateVirtual(v.output); err != nil {
		return err
	}

	// the desk outputs stay on, so focus decides where gamescope opens
	return v.comp.Focus(v.output)
}

func (v virtual) Stop() error {
	return v.comp.DestroyVirtual()
}
