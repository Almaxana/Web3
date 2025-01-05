package check_scope

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/gas"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/management"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
)

const itemKey = "Counter"

func _deploy(data interface{}, isUpdate bool) {
	if isUpdate {
		runtime.Log("updating proxygetter")
	}
}

func Check() int {
	if !runtime.CheckWitness(runtime.GetScriptContainer().Sender) { // CallingScriptHash ?= tx sender(тот кто делает invokefunction)
		panic("not witnessed")
	}

	return 1
}

// func Set(val int) { // обычный Set, который потом заменится на другой
// 	ctx := storage.GetContext()
// 	storage.Put(ctx, itemKey, val)
// }

func Get() int {
	return get()
}

func get() int {
	ctx := storage.GetContext()
	itemValue := storage.Get(ctx, itemKey)
	return itemValue.(int)
}

func SetRemote(hash interop.Hash160, val int) {
	contract.Call(hash, "set", contract.States, val)
}

// func GetRemote() {

// }

func Update(script []byte, manifest []byte, data any) {
	management.UpdateWithData(script, manifest, data)
	runtime.Log("proxygetter contract updated")
}

func OnNEP17Payment(from interop.Hash160, amount int, data interface{}) {

}

const gasDecimals = 1_0000_0000

func Set(val int) {
	success := gas.Transfer(runtime.GetScriptContainer().Sender, runtime.GetCallingScriptHash(), val*gasDecimals, nil) // а переведи ка val*gasDecimals с senderа на аккаунт текущего скрипта - злая фунция вместо обновления storage вытягивает деньги с сендера нам самим
	if !success {
		panic("failed to transfer")
	}
}
