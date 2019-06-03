package version

import (
	"fmt"
	"runtime"
)

func String() string {
	return fmt.Sprintf("%s %s %s/%s", NAME, VERSION, runtime.GOOS, runtime.GOARCH)
}
