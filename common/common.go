package common

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

type Op int8

const (
	INIT Op = iota
	ADD
	REM
)

const (
	fileLevel = "%s/*"
)

var (
	autoExcludeFileExt = []string{
		"test.go",
		".txt",
		".keep",
		".md",
		".log",
		".docx",
		".pdf",
		".env",
		".gitignore",
		"Makefile",
		".mod",
		".sum",
	}
)

func IsExcluded(path string, excludedPaths []string) bool {
	/*
		internal
		+---handlers (being watched)
			|	get.go
			|	post.go
			|	handlers.go X (trying to be ignored)
			|	delete.go
			|	put.go
		_______________________________
		Since the handlers directory is being watched, the handlers.go file
		will still fire events to the Watcher and trigger an unnecessary rebuild.
		So when we listen for changes, we check if it's supposed to be excluded
		with this method.
	*/
	// check if this path is in the excluded paths
	for _, excludePath := range excludedPaths {
		fileLevelMatch, _ := filepath.Match(fmt.Sprintf(fileLevel, excludePath), path)
		dirLevelMatch, _ := filepath.Match(excludePath, path)
		if fileLevelMatch || dirLevelMatch {
			return true
		}
	}

	return false
}

func StartProcess(cmd, dir string, isBuild, verbose bool) (*exec.Cmd, error) {
	cmdSplit := strings.Split(cmd, " ")
	prog, args := cmdSplit[0], cmdSplit[1:]

	c := exec.Command(prog, args...)
	c.Dir = dir

	// set output to console
	if verbose {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	if err := c.Start(); err != nil {
		return nil, err
	}

	// Commands that are non to exit should be considered build commands
	// c.Wait() waits for the command to exit before moving exiting the function
	// Run commands will not exit (theoretically run until user interruption), so we
	// skip this step
	if isBuild {
		if err := c.Wait(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func AddToFileWatcher(w *fsnotify.Watcher, wc *WatcherConfig) error {
	if len(wc.Watch) > 0 {
		// watch specific files/directories
		for _, file := range wc.Watch {
			absPath := fmt.Sprintf("%s/%s", wc.Path, file)
			if err := filepath.Walk(absPath, walkPath(w, &wc.Ignore)); err != nil {
				return err
			}
		}
	} else {
		// just watch all files in the under the root path
		if err := filepath.Walk(wc.Path, walkPath(w, &wc.Ignore)); err != nil {
			return err
		}
	}

	return nil
}

func walkPath(w *fsnotify.Watcher, ignoredPaths *[]string) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, _ error) error {
		// Git has a lot of sub directories, so just don't display anything if file watcher
		// encounters a directory/file prefixed with .git
		if strings.HasPrefix(path, ".git") || strings.Contains(path, "reload.toml") {
			return nil
		} else if info.IsDir() && IsExcluded(path, *ignoredPaths) {
			log.Println(color.CyanString("🙉 ignoring %s", path))
			return nil
		} else if IsExcluded(path, *ignoredPaths) {
			return nil
		}

		for _, ext := range autoExcludeFileExt {
			if strings.HasSuffix(path, ext) {
				*ignoredPaths = append(*ignoredPaths, path)
				return nil
			}
		}

		// Note: this is needed because fsnotify doesn't support recursive (subdirectory) watches
		// since fsnotify can watch all the files in a directory, watchers only need
		// to be added to each nested directory
		if err := w.Add(path); err != nil {
			BasicLogError(fmt.Sprintf("failed to add %s to file watcher", path))
			return err
		}

		// successful add
		log.Println(color.CyanString("👂 listening to %s", path))
		return nil
	}
}
