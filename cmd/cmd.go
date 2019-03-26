/*
Package cmd houses the command line functionality of cfn.
*/
package cmd

// Command represents a callable command
type CommandFunc func([]string)

// Commands stores a mapping of command names to their functions
var Commands = make(map[string]CommandFunc)
