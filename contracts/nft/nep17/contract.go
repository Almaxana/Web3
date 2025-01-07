package contract

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/management"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
)

// Prefixes used for contract data storage.
const (
	ownerKey       = 'o'
	totalSupplyKey = 's'
)

const (
	decimals   = 8
	multiplier = 100000000
)

func _deploy(data interface{}, isUpdate bool) {
	if isUpdate {
		return
	}

	args := data.(struct {
		Admin interop.Hash160 // владелец контракта и он же будет владеть всеми этими 100 выпущенными токенами
		Total int             // кол-во токенов, которые будут существовать
	})

	if args.Admin == nil {
		panic("invalid admin")
	}

	if len(args.Admin) != 20 {
		panic("invalid admin hash length")
	}

	if args.Total <= 0 {
		panic("invalid total supply")
	}

	ctx := storage.GetContext()
	storage.Put(ctx, ownerKey, args.Admin)

	total := args.Total * multiplier
	storage.Put(ctx, args.Admin, total)
	storage.Put(ctx, totalSupplyKey, total)
}

// Symbol returns the token symbol
func Symbol() string {
	return "MYTKN"
}

// Decimals returns the token decimals
func Decimals() int {
	return decimals
}

// TotalSupply returns the token total supply value
func TotalSupply() int {
	return storage.Get(storage.GetReadOnlyContext(), totalSupplyKey).(int)
}

// BalanceOf returns the amount of token on the specified address
func BalanceOf(holder interop.Hash160) int {
	if len(holder) != 20 {
		panic("bad owner address")
	}
	return storage.Get(storage.GetReadOnlyContext(), holder).(int)
}

// Transfer token from one user to another
func Transfer(from interop.Hash160, to interop.Hash160, amount int, data any) bool {
	ctx := storage.GetContext()

	if len(from) != 20 || len(to) != 20 {
		panic("invalid addresses")
	}

	if amount < 0 {
		panic("invalid amount")
	}

	if !runtime.CheckWitness(from) {
		return false
	}

	// наш текущий баланс токенов
	amountFrom := getIntFromDB(ctx, from)
	if amountFrom < amount { // пытаемся перевести больше, чем у нас есть
		return false
	}
	storage.Put(ctx, from, amountFrom-amount)

	// баланс токенов того, кому переводим
	amountTo := getIntFromDB(ctx, to)
	storage.Put(ctx, to, amountTo+amount)

	runtime.Notify("Transfer", from, to, amount)
	if management.GetContract(to) != nil { // если переводим не на обычный кошелек, а на адрес контракта, то надо вызвать
		// у него onNEP17Payment, чтоюы он мог этот перевод обработать
		contract.Call(to, "onNEP17Payment", contract.All, from, amount, data)
	}
	return true
}

func getIntFromDB(ctx storage.Context, key []byte) int {
	var res int
	val := storage.Get(ctx, key)
	if val != nil {
		res = val.(int)
	}
	return res
}
