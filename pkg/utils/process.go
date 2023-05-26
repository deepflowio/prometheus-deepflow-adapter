package utils

import (
	"os"
	"path/filepath"
)

func GetProcessName() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(path)
}
