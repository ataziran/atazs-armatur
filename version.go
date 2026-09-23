package main

import "runtime/debug"

// formatVersion renders the one line `--version` prints. The data comes from
// the build info the toolchain embeds, so nothing has to be stamped in at
// build time.
func formatVersion(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil {
		return "atazs-armatur (unknown)"
	}
	version := info.Main.Version
	if version == "" {
		version = "(unknown)"
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
			return "atazs-armatur " + version + " (" + setting.Value[:7] + ")"
		}
	}
	return "atazs-armatur " + version
}
