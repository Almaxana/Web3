package hello

import "github.com/nspcc-dev/neo-go/pkg/interop/runtime"

var notificationName string

// init initializes notificationName before calling any other smart-contract method
func init() {
	notificationName = "Hello world!"
}

// RuntimeNotify sends runtime notification with "Hello world!" name
func RuntimeNotify(args []any) {
	runtime.Notify(notificationName, args)
}

// func Main() {
// 	runtime.Log("LOG Hello world")
// }
