package main

import (
	"os"

	goversion "github.com/caarlos0/go-version"

	"github.com/laiambryant/tui-cardman/cmd/command"
	"github.com/laiambryant/tui-cardman/internal/tui/art"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	versionInfo := buildVersion(version, commit, date, builtBy)
	if err := command.Execute(versionInfo); err != nil {
		os.Exit(1)
	}
}

func buildVersion(version, commit, date, builtBy string) goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails("tui-cardman", "Terminal UI Card Manager", "https://github.com/laiambryant/tui-cardman"),
		goversion.WithASCIIName(art.Logo),
		func(i *goversion.Info) {
			if commit != "" {
				i.GitCommit = commit
			}
			if date != "" {
				i.BuildDate = date
			}
			if version != "" {
				i.GitVersion = version
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}
		},
	)
}
