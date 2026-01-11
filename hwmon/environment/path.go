package environment

import (
	"os"
	"path/filepath"
)

const KeyHostSys = "HOST_SYS"

func GetEnvPath(key, fallback string, elem ...string) (v string) {
	v = os.Getenv(key)
	if v == "" {
		v = fallback
	}

	return filepath.Join(append([]string{v}, elem...)...)
}
