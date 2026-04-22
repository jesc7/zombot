package client

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

func runPath(interactive bool) (string, error) {
	res := os.Args[0]
	if !interactive {
		key, e := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\zspy", registry.QUERY_VALUE)
		if e != nil {
			return "", e
		}
		if res, _, e = key.GetStringValue("ImagePath"); e != nil {
			return "", e
		}
	}
	return res, nil
}
