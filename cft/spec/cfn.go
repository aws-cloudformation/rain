package spec

import (
	_ "embed"
	"encoding/json"
)

//go:embed schemas.json
var schemas []byte

var Cfn = make(map[string]Schema)

func init() {
	err := json.Unmarshal(schemas, &Cfn)
	if err != nil {
		panic(err)
	}
}
