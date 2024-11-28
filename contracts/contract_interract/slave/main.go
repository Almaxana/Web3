package slave

import "github.com/nspcc-dev/neo-go/pkg/interop/runtime"

func AddNumbers(a, b int) int {
	runtime.Notify("info", []byte("Add numbers"))
	return a + b
}
