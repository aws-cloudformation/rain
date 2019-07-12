package config

import (
	"fmt"

	"github.com/aws-cloudformation/rain/console/text"
)

var Debug = false
var Profile = ""
var Region = ""

func Debugf(message string, parts ...interface{}) {
	if Debug {
		fmt.Println(text.Orange("DEBUG: " + fmt.Sprintf(message, parts...)))
	}
}
