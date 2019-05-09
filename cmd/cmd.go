/*
Package cmd houses the command line functionality of rain
*/
package cmd

type cmdType string

const (
	STACK    cmdType = "Stack"
	TEMPLATE cmdType = "Template"
	PLUGIN   cmdType = "Plugin"
)

// Command represents a callable command
type Command struct {
	Type cmdType
	Help string
	Run  func([]string)
}

// Commands stores a mapping of command names to their functions
var Commands = make(map[string]Command)

var CommandTypes = []cmdType{
	TEMPLATE,
	STACK,
	PLUGIN,
}
