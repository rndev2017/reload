package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reload/common"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// constants
var (
	basicKeys = []string{
		"containerized",
		"verbose",
		"path",
		"watch",
		"ignore",
	}
	composeKeys = []string{"service"}
	rootKeys    = []string{
		"build",
		"run",
	}
)

var startCmd = &cobra.Command{
	Use:   "start workflow",
	Short: "Run a custom workflow from the reload.toml file (no more nasty flags 🤮)",
	Run:   startRun,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func keysExist(title string, conf map[string]interface{}, keys []string) {
	for _, key := range keys {
		if _, ok := conf[key]; !ok {
			msg := fmt.Sprintf(
				"workflow %s expected key %s, but none was provided",
				common.HiYlw(title),
				common.Mgnta(key))
			common.BasicLogError(msg)
			os.Exit(1)
		}
	}
}

func getStringSlice(in []interface{}) []string {
	s := make([]string, len(in))
	for i, val := range in {
		s[i] = val.(string)
	}

	return s
}

func startRun(cmd *cobra.Command, args []string) {
	var arg string = ""
	if len(args) > 0 {
		arg = args[0]
	} else {
		common.BasicLogError("no workflow name provided")
	}

	// read path
	path, _ := cmd.Flags().GetString("path")

	// change directory
	if err := os.Chdir(path); err != nil {
		common.BasicLogError(fmt.Sprintf("unrecognized path %s", path))
		os.Exit(1)
	}

	// read file
	tomlData, err := ioutil.ReadFile("reload.toml")
	if err != nil {
		log.Printf(
			"%s\tcould not find a reload.toml file",
			common.ErrorRed("error"),
		)
		log.Fatalf(
			"%s\trun %s to create one",
			common.ExtraHiYlw("tip"),
			common.ExtraHiGreen("reload init"),
		)
		os.Exit(1)
	}

	// Decode the config
	var conf map[string]interface{}
	if _, err := toml.Decode(string(tomlData), &conf); err != nil {
		common.BasicLogError(fmt.Sprintf("reload.%+v", err))
		os.Exit(1)
	}

	// check if specified workflow exists
	workflow, ok := conf[arg]
	if !ok {
		common.BasicLogError(fmt.Sprintf("could not find workflow %s in reload.toml", arg))
		os.Exit(1)
	}

	// type switching (fancy...)
	switch workflowMap := workflow.(type) {
	case map[string]interface{}:
		keysExist(arg, workflowMap, basicKeys)

		containerized := workflowMap["containerized"]
		if containerized.(bool) {
			// Check command level keys
			keysExist(arg, workflowMap, composeKeys)

			// Construct docker compose flags
			cf := common.ComposeFlags{
				WC: common.WatcherConfig{
					Path:   workflowMap["path"].(string),
					Watch:  getStringSlice(workflowMap["watch"].([]interface{})),
					Ignore: getStringSlice(workflowMap["ignore"].([]interface{})),
				},
				Service: workflowMap["service"].(string),
				Verbose: workflowMap["verbose"].(bool),
			}

			// run the docker compose workflow
			if err = startComposeReload(cf); err != nil {
				common.BasicLogError(fmt.Sprintf("failed to run workflow %s", arg))
				os.Exit(1)
			}
		} else {
			// Check command level keys
			keysExist(arg, workflowMap, rootKeys)

			// Construct docker compose flags
			rf := common.RootFlags{
				WC: common.WatcherConfig{
					Path:   workflowMap["path"].(string),
					Watch:  getStringSlice(workflowMap["watch"].([]interface{})),
					Ignore: getStringSlice(workflowMap["ignore"].([]interface{})),
				},
				Build:   getStringSlice(workflowMap["build"].([]interface{})),
				Run:     workflowMap["run"].(string),
				Verbose: workflowMap["verbose"].(bool),
			}

			// run the docker compose workflow
			if err = startRootReload(rf); err != nil {
				common.BasicLogError(fmt.Sprintf("failed to run workflow %s", arg))
				os.Exit(1)
			}
		}
	}
}
