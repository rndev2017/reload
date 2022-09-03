package common

import (
	"log"

	"github.com/fatih/color"
)

var (
	ErrorRed      = color.New(color.FgRed, color.Bold).SprintFunc()
	Ylw           = color.New(color.FgYellow).SprintFunc()
	HiYlw         = color.New(color.FgHiYellow).SprintFunc()
	ExtraHiYlw    = color.New(color.Bold, color.FgHiYellow).SprintFunc()
	HiCyan        = color.New(color.FgCyan).SprintFunc()
	HiGreen       = color.New(color.FgHiGreen).SprintFunc()
	ExtraHiGreen  = color.New(color.Bold, color.Underline, color.FgHiGreen).SprintFunc()
	Mgnta         = color.New(color.FgMagenta).SprintFunc()
	LogEvent      = func(format, event string) { log.Println(color.MagentaString(format, event)) }
	BasicLogError = func(msg string) { log.Fatalf("%s\t%s", ErrorRed("error"), msg) }
)
