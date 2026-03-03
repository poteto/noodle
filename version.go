package main

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

const fallbackVersion = "dev"

//go:embed VERSION
var canonicalVersion string

func currentVersion() string {
	if version := normalizeVersion(canonicalVersion); version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackVersion
	}
	if version := normalizeVersion(info.Main.Version); version != "" {
		return version
	}
	return fallbackVersion
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	switch version {
	case "", "devel", "(devel)", "unknown":
		return ""
	default:
		return version
	}
}
