package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reload/common"

	"github.com/spf13/cobra"
)

var defaultTomlFile string = `# reload.toml config file

# basic workflow (no subcommands)
[basic]
containerized = false
verbose = true
path = "." # path to the project directory
watch = [ ] # files to watch
ignore = [ ] # files to ignore
build = [ ] # build shell commands
run = ""

# make use of the 'docker compose' functionality
[compose]
containerized = true
verbose = true
path = "." # path to the project directory
watch = [ ] # files to watch
ignore = [ ] # files to ignore
service = ""

# add as many as you like...
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a reload.toml file to configure workflows for live reload functionality",
	Run:   initRun,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initRun(cmd *cobra.Command, _ []string) {
	path, _ := cmd.Flags().GetString("path")

	// change directory
	if err := os.Chdir(path); err != nil {
		common.BasicLogError(fmt.Sprintf("unrecognized path %s", path))
		os.Exit(1)
	}

	// check if it iexists
	if _, err := os.Stat("reload.toml"); err == nil {
		log.Printf(
			"%s\treload.toml file already exists",
			common.ErrorRed("error"),
		)
		log.Fatalf(
			"%s\trun %s to get started",
			common.ExtraHiYlw("tip"),
			common.ExtraHiGreen("reload start"),
		)
	} else if errors.Is(err, os.ErrNotExist) {
		f, err := os.Create("reload.toml")
		if err != nil {
			common.BasicLogError("failed to create reload.toml")
		}
		f.WriteString(defaultTomlFile)
		defer f.Close()
		log.Printf("%s created", common.HiCyan("reload.toml"))
		log.Printf(
			"%s\trun %s to get started",
			common.ExtraHiYlw("tip"),
			common.ExtraHiGreen("reload start"),
		)
	}
}
