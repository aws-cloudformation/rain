package table_test

import (
	"fmt"

	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/table"
)

func Example() {
	t := table.New("Name", "Rank")
	t.Append("Kirk", "Captain")
	t.Append("Spock", "Science Officer")
	t.Append("Scotty", "Chief Engineer")

	// Disable ANSI formatting
	console.IsTTY = false
	fmt.Println(t.String())
	// Output:
	// +--------+-----------------+
	// | Name   | Rank            |
	// |--------|-----------------|
	// | Kirk   | Captain         |
	// | Spock  | Science Officer |
	// | Scotty | Chief Engineer  |
	// +--------+-----------------+
}
