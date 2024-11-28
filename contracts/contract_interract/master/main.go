package master

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
)

func CallHelper(slaveContractHash interop.Hash160, a, b int) int {

	result := contract.Call(slaveContractHash, "addNumbers", contract.All, a, b).(int)
	runtime.Notify("info", []byte("Calling helper"))
	return result
}
