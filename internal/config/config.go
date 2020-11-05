package config

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/console"
)

// Debug defines whether debug mode is enabled
var Debug = false

// Profile holds the requested AWS profile name
var Profile = ""

// Region holds the requested AWS region name
var Region = ""

// Debugf prints messages for stdout only if Debug is true
func Debugf(message string, parts ...interface{}) {
	if Debug {
		fmt.Println(console.Grey("DEBUG: " + fmt.Sprintf(message, parts...)))
	}
}
