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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "reload",
	Short: "Increase development speed & productivity with live reloading for all your apps.",
	Run:   rootRun,
}

// var w *fsnotify.Watcher

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// Initializes the root command
func init() {
	// basic app support (run custom build and run commands)
	rootCmd.PersistentFlags().StringP("path", "p", ".", "Path to watch files from")
	rootCmd.Flags().StringSliceP(
		"watch",
		"w",
		[]string{},
		"Files/directories to watch (relative to --path)",
	)
	rootCmd.Flags().StringSlice(
		"ignore",
		[]string{},
		"Files/directories to ignore (relative to --path)",
	)
	rootCmd.Flags().StringSliceP("build", "b", []string{}, "Shell command to build your project")
	rootCmd.Flags().StringP("run", "r", "", "Shell command to run your project")
	rootCmd.Flags().BoolP("verbose", "v", true, "Displays build and run output to the console")
}

func rootRun(cmd *cobra.Command, _ []string) {
	flags := constructRootFlags(cmd.Flags())
	err := startRootReload(flags)
	if err != nil {
		common.BasicLogError("failed to start live reload")
		os.Exit(1)
	}
}

func constructRootFlags(flags *pflag.FlagSet) common.RootFlags {
	rf := common.RootFlags{}
	rf.WC.Path, _ = flags.GetString("path")
	rf.WC.Watch, _ = flags.GetStringSlice("watch")
	rf.WC.Ignore, _ = flags.GetStringSlice("ignore")
	rf.Build, _ = flags.GetStringSlice("build")
	rf.Run, _ = flags.GetString("run")
	rf.Verbose, _ = flags.GetBool("verbose")

	return rf
}

func startRootReload(flags common.RootFlags) error {
	// both build and run can't be empty (at the same time)
	if len(flags.Build) <= 0 && flags.Run == "" {
		common.BasicLogError("--build or --run flags must be set")
		os.Exit(1)
	}

	// add hidden files like .git
	flags.WC.Ignore = append(flags.WC.Ignore, ".git")
	flags.WC.Ignore = append(flags.WC.Ignore, "reload.toml")

	// create a new watcher
	var w *fsnotify.Watcher
	w, _ = fsnotify.NewWatcher()
	defer w.Close()

	// add files to FileWatcher
	err := common.AddToFileWatcher(w, &flags.WC)
	if err != nil {
		common.BasicLogError("failed to add watchlist to file watcher")
	}

	done := make(chan bool)
	go runRootReload(w, flags)

	// used to synchronize between the goroutine thread and the main thread
	// we're forcing the main thread to wait until there is some data to be consumed in the done channel
	<-done
	return nil
}

func runRootReload(watcher *fsnotify.Watcher, flags common.RootFlags) {
	// run initial build
	proc := runRootCommands(flags, nil)
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
					proc = runRootCommands(flags, proc)
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

func runRootCommands(flags common.RootFlags, oldRunProc *exec.Cmd) *exec.Cmd {
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
	}

	// run build
	if len(flags.Build) > 0 {
		log.Printf("🏗️  %s", common.HiYlw("building..."))
		for _, cmd := range flags.Build {
			if _, err := common.StartProcess(cmd, flags.WC.Path, true, flags.Verbose); err != nil {
				common.BasicLogError("failed to execute build process")
			}
		}
	}

	// run the new proccess
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
