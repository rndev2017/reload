package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"reload/common"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var composeCmd = &cobra.Command{
	Use:   "compose [service]",
	Short: "Live reload functionality for containerized services built for Docker compose",
	Run:   composeRun,
}

func init() {
	composeCmd.Flags().StringSliceP(
		"watch",
		"w",
		[]string{},
		"files/directories to watch (relative to --path)",
	)
	composeCmd.Flags().StringSlice(
		"ignore",
		[]string{},
		"files/directories to ignore (relative to --path)",
	)
	// Docker & Docker-Compose flags
	composeCmd.Flags().BoolP("verbose", "v", true, "Display docker-compose logs to console")
	rootCmd.AddCommand(composeCmd)
}

func constructComposeFlags(service string, flags *pflag.FlagSet) common.ComposeFlags {
	df := common.ComposeFlags{
		Service: service,
	}
	df.WC.Path, _ = flags.GetString("path")
	df.WC.Watch, _ = flags.GetStringSlice("watch")
	df.WC.Ignore, _ = flags.GetStringSlice("ignore")
	df.Verbose, _ = flags.GetBool("verbose")

	return df
}

func composeRun(cmd *cobra.Command, args []string) {
	var service string = ""
	if len(args) > 0 {
		service = args[0]
	}
	flags := constructComposeFlags(service, cmd.Flags())
	err := startComposeReload(flags)
	if err != nil {
		common.BasicLogError("failed to start live reload")
		os.Exit(1)
	}
}

func startComposeReload(flags common.ComposeFlags) error {
	// create a new watcher
	var w *fsnotify.Watcher
	w, _ = fsnotify.NewWatcher()
	defer w.Close()

	// add hidden files
	flags.WC.Ignore = append(flags.WC.Ignore, ".git")
	flags.WC.Ignore = append(flags.WC.Ignore, "reload.toml")

	err := common.AddToFileWatcher(w, &flags.WC)
	if err != nil {
		common.BasicLogError("failed to add watchlist to file watcher")
	}

	// construct build, run, & clean commands
	if flags.Service != "" {
		flags.Run = fmt.Sprintf("docker compose up %s", flags.Service)
		flags.Clean = fmt.Sprintf("docker compose stop %s", flags.Service)
	} else {
		flags.Run = "docker compose up"
		flags.Clean = fmt.Sprintf("docker compose stop %s", flags.Service)
	}

	done := make(chan bool)
	go runComposeReload(w, flags)

	// used to synchronize between the goroutine thread and the main thread
	// we're forcing the main thread to wait until there is some data to be consumed in the done channel
	<-done
	return nil
}

func runComposeReload(watcher *fsnotify.Watcher, flags common.ComposeFlags) error {
	// run initial build
	proc := runComposeCommands(flags, nil)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				common.BasicLogError("failed to read from watcher.Events channel")
			}
			if !common.IsExcluded(event.Name, flags.WC.Ignore) {
				// listen for file changes
				if event.Op&fsnotify.Create == fsnotify.Create {
					common.LogEvent("a wild %s has appeared", event.Name)
					flags.WC.Watch = append(flags.WC.Watch, event.Name)
					common.AddToFileWatcher(watcher, &flags.WC)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					common.LogEvent("%s has been removed", event.Name)
					// todo: handle removing from file watcher
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					common.LogEvent("%s has a new name", event.Name)
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					common.LogEvent("%s has changed", event.Name)
				}
				// rerun commands after changes
				if event.Op&fsnotify.Chmod != fsnotify.Chmod {
					proc = runComposeCommands(flags, proc)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				common.BasicLogError("failed to read from watcher.Errors channel")
			}
			common.BasicLogError(fmt.Sprintf("%+v", err))
		}
	}
}

func runComposeCommands(flags common.ComposeFlags, oldRunProc *exec.Cmd) *exec.Cmd {
	// setup (change dir)
	// change current working directory
	if err := os.Chdir(flags.WC.Path); err != nil {
		common.BasicLogError(fmt.Sprintf("unrecognized path %s", flags.WC.Path))
	}

	// cleanup
	if oldRunProc != nil {
		oldRunProc.Process.Kill()
		if err := oldRunProc.Wait(); err != nil {
			log.Printf(
				"%s\tcleaned up after old run process",
				common.HiCyan("[INFO]"),
			)
		}

		// remove the docker containers
		_, err := common.StartProcess(flags.Clean, flags.WC.Path, true, false)
		if err != nil {
			common.BasicLogError("failed to clean up containers")
		}
	}

	var runProc *exec.Cmd
	var err error
	if flags.Run != "" {
		// execute run cmd
		log.Printf("🏃 %s", common.HiGreen("running..."))
		runProc, err = common.StartProcess(flags.Run, flags.WC.Path, false, flags.Verbose)
		if err != nil {
			common.BasicLogError("failed to execute run process")
		}
	}

	// return the new proc
	return runProc
}
