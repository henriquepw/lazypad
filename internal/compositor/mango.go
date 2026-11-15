package compositor

import "github.com/henriquepw/lazypad/internal/proc"

type Mango struct{}

func (Mango) Enable(output string) error {
	return proc.Run("mmsg", "dispatch", "enable_monitor,"+output)
}

func (Mango) Disable(output string) error {
	return proc.Run("mmsg", "dispatch", "disable_monitor,"+output)
}

func (Mango) CreateVirtual(output string) error {
	return proc.Run("mmsg", "dispatch", "create_virtual_output,"+output)
}

func (Mango) DestroyVirtual() error {
	return proc.Run("mmsg", "dispatch", "destroy_all_virtual_output")
}

func (Mango) Focus(output string) error {
	return proc.Run("mmsg", "dispatch", "focusmon,"+output)
}
