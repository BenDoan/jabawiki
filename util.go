package main

import (
	"path/filepath"
)

func AbsPathFromExe(parts ...string) string {
	path := exePath
	for _, part := range parts {
		path = filepath.Join(path, part)
	}
	return path
}

func getDataDirPath() string {
	return filepath.FromSlash(conf.DataDir)
}
