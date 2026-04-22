package client

import (
	"context"
	"os"
	"path/filepath"
)

func Run(ctx context.Context, service bool) error {
	var e error

	exe_path, e := runPath(service)
	if e != nil {
		return e
	}

	f, e := os.ReadFile(filepath.Join(filepath.Dir(exe_path), "cfg.json"))
	if e != nil {
		return e
	}
	_ = f

	return e
}
