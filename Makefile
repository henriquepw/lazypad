.PHONY: install
install:
	GOBIN="$(HOME)/.local/bin" go install .
	"$(HOME)/.local/bin/lazypad" install
	systemctl --user try-restart lazypad.service
