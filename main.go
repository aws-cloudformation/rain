// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
)

func main() {
	defer func() {
		spinner.Stop()

		if r := recover(); r != nil {
			if config.Debug {
				panic(r)
			}

			fmt.Println(text.Red(fmt.Sprint(r)))
			os.Exit(1)
		}

		os.Exit(0)
	}()

	cmd.Execute()
}
