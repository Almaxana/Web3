package contract

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/iterator"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
	"github.com/nspcc-dev/neo-go/pkg/interop/util"
	// nicenamesnft "nft/wrappers/nep11"
)

// Prefixes used for contract data storage.
const (
	ownerKey = 'o'
	tokenKey = 't'
	nftKey   = 'n'

	tokensPrefix = "tt"
)

type NFTItem struct {
	ID         []byte
	Owner      interop.Hash160
	Name       string
	PrevOwners int
	Created    int
	Bought     int
}

func _deploy(data interface{}, isUpdate bool) {
	if isUpdate {
		return
	}

	args := data.(struct {
		Admin  interop.Hash160
		Token  interop.Hash160
		Market interop.Hash160
	})

	if args.Admin == nil {
		panic("invalid admin")
	}

	if len(args.Admin) != 20 {
		panic("invalid admin hash length")
	}

	ctx := storage.GetContext()
	storage.Put(ctx, ownerKey, args.Admin)
	storage.Put(ctx, tokenKey, args.Token)
	storage.Put(ctx, nftKey, args.Market)
}

func OnNEP11Payment(from interop.Hash160, amount int, token []byte, data any) {
	ctx := storage.GetContext()
	callingHash := runtime.GetCallingScriptHash()
	if !callingHash.Equals(storage.Get(ctx, nftKey).(interop.Hash160)) {
		panic("invalid nft")
	}

	if amount != 1 {
		panic("invalid amount")
	}

	key := append([]byte(tokensPrefix), token...)
	if data == nil {
		data = []byte{}
	}
	storage.Put(ctx, key, data)
}

func List() []map[string]string {
	ctx := storage.GetContext()

	nft := storage.Get(ctx, nftKey).(interop.Hash160)

	res := []map[string]string{}
	iter := storage.Find(ctx, []byte(tokensPrefix), storage.KeysOnly|storage.RemovePrefix)
	for iterator.Next(iter) {
		token := iterator.Value(iter).([]byte)
		prop := contract.Call(nft, "properties", contract.All, token).(map[string]string)
		res = append(res, prop)
	}

	return res
}

func OnNEP17Payment(from interop.Hash160, amount int, data any) {
	defer func() {
		if r := recover(); r != nil {
			runtime.Log(r.(string))
			util.Abort()
		}
	}()

	ctx := storage.GetContext()

	callingHash := runtime.GetCallingScriptHash()
	if !callingHash.Equals(storage.Get(ctx, tokenKey).(interop.Hash160)) { // в nep17 проверяем, что нам перевели GAS, а здесь проверяем, что нам перевели
		// наш MYTKN
		panic("invalid token")
	}

	if amount < 10_0000_0000 {
		panic("insufficient funds")
	}

	token := data.([]byte)
	key := append([]byte(tokensPrefix), token...)

	if storage.Get(ctx, key) == nil {
		panic("token not found")
	}
	storage.Delete(ctx, key)

	nft := storage.Get(ctx, nftKey).(interop.Hash160)
	contract.Call(nft, "transfer", contract.All, from, token, data)
	// nicenamesnft.Transfer(from, token , nil)
}

func TransferTokens(to interop.Hash160, amount int) {
	defer func() {
		if r := recover(); r != nil {
			runtime.Log(r.(string))
			util.Abort()
		}
	}()

	ctx := storage.GetContext()

	owner := storage.Get(ctx, ownerKey).(interop.Hash160)
	if !runtime.CheckWitness(owner) {
		panic("not witnesssed")
	}

	tokenHash := storage.Get(ctx, tokenKey).(interop.Hash160)

	contract.Call(tokenHash, "transfer", contract.All, runtime.GetExecutingScriptHash(), to, amount, nil)
}
