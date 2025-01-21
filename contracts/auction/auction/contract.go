package auction

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/lib/address"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/management"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
)

// Prefixes used for contract data storage.
const (
	initBetKey            = "i"
	currentBetKey         = "c"
	lotKey                = "l" // nft id
	organizerKey          = "o" // organizer of the auction
	potentialWinnerKey    = "w" // owner of the last bet
	nnsSelfDomain         = "auc.auc"
	nnsNftDomain          = "nft.auc"
	nnsRecordType         = 16
	nnsContractHashString = "NcCZaxnLkXvrd56DgpFSSBjhj2DqzH3jKP"
)

type AuctionItem struct {
	Owner      interop.Hash160
	InitialBet int
	CurrentBet int
	LotID      interop.Hash160
}

func _deploy(data interface{}, isUpdate bool) {
	if isUpdate {
		return
	}

	selfHash := runtime.GetExecutingScriptHash()
	contract.Call(address.ToHash160(nnsContractHashString), "register", contract.All, nnsSelfDomain, address.ToHash160("NfgHwwTi3wHAS8aFAN243C5vGbkYDpqLHP"), "owner_email@mail.ru", 100, 100, 31536000, 31536000)
	currentNnsRecord := contract.Call(address.ToHash160(nnsContractHashString), "getRecords", contract.All, nnsSelfDomain, nnsRecordType)
	if currentNnsRecord != nil {
		contract.Call(address.ToHash160(nnsContractHashString), "deleteRecords", contract.All, nnsSelfDomain, nnsRecordType)
	}
	contract.Call(address.ToHash160(nnsContractHashString), "addRecord", contract.All, nnsSelfDomain, nnsRecordType, address.FromHash160(selfHash))

}

func Update(script []byte, manifest []byte, data any) {
	management.UpdateWithData(script, manifest, data)
}

func Start(auctionOwner interop.Hash160, lotId []byte, initBet int) {
	ctx := storage.GetContext()

	currentOwner := storage.Get(ctx, organizerKey)
	if currentOwner != nil {
		panic("now current auction is processing, wait for finish")
	}
	if initBet < 0 {
		panic("initial bet must not be negative")
	}

	nftContractHashStringArray := contract.Call(address.ToHash160(nnsContractHashString), "resolve", contract.All, nnsNftDomain, nnsRecordType).([]string)
	nftContractHashString := nftContractHashStringArray[0]
	ownerOfLot := contract.Call(address.ToHash160(nftContractHashString), "ownerOf", contract.All, lotId).(interop.Hash160)
	if !ownerOfLot.Equals(auctionOwner) {
		panic("you cannot start auction with lot " + string(lotId) + " because you're not its owner")
	}

	storage.Put(ctx, organizerKey, auctionOwner)
	storage.Put(ctx, lotKey, lotId)
	storage.Put(ctx, initBetKey, initBet)
	storage.Put(ctx, currentBetKey, initBet)

	runtime.Notify("info", []byte("New auction started with initial bet = "+intToStr(initBet)))
}

func MakeBet(maker interop.Hash160, bet int) {
	ctx := storage.GetContext()

	currentOwner := storage.Get(ctx, organizerKey)
	if currentOwner == nil {
		panic("there are no auctions at this moment")
	}

	currentBet := storage.Get(ctx, currentBetKey).(int)
	if bet <= currentBet {
		panic("new bet must be greater than current")
	}
	auctionOwner := storage.Get(ctx, organizerKey)
	if maker.Equals(auctionOwner) {
		panic("owner of the auction can't make bet")
	}

	storage.Put(ctx, currentBetKey, bet)
	storage.Put(ctx, potentialWinnerKey, maker)

	runtime.Notify("info", []byte("New bet = "+intToStr(bet)+" is made"))

}

func Finish(finishInitiator interop.Hash160) {
	ctx := storage.GetContext()

	auctionOwner := storage.Get(ctx, organizerKey).(interop.Hash160)
	if !finishInitiator.Equals(auctionOwner) {
		panic("only organizer of the auction can finish it")
	}

	potentialWinner := storage.Get(ctx, potentialWinnerKey).(interop.Hash160)
	if potentialWinner == nil {
		storage.Delete(ctx, initBetKey)
		storage.Delete(ctx, currentBetKey)
		storage.Delete(ctx, potentialWinnerKey)
		storage.Delete(ctx, organizerKey)
		storage.Delete(ctx, lotKey)
		runtime.Notify("info", []byte("Auction finished"))
		return
	}

	lotId := storage.Get(ctx, lotKey)

	nftContractHashStringArray := contract.Call(address.ToHash160(nnsContractHashString), "resolve", contract.All, nnsNftDomain, nnsRecordType).([]string)
	nftContractHashString := nftContractHashStringArray[0]
	res := contract.Call(address.ToHash160(nftContractHashString), "transfer", contract.All, potentialWinner, lotId, nil).(bool)
	if !res {
		panic("error while transfer token to winner")
	}

	storage.Delete(ctx, initBetKey)
	storage.Delete(ctx, currentBetKey)
	storage.Delete(ctx, potentialWinnerKey)
	storage.Delete(ctx, organizerKey)
	storage.Delete(ctx, lotKey)

	runtime.Notify("info", []byte("Auction finished"))

}

func intToStr(value int) string {
	if value == 0 {
		return "0"
	}
	var chars = "0123456789"
	var result string
	for value > 0 {
		result = string(chars[value%10]) + result
		value = value / 10
	}

	return result
}

func ShowCurrentBet() string {
	data := storage.Get(storage.GetReadOnlyContext(), currentBetKey)
	if data == nil {
		return "0"
	}
	return string(data.([]byte))
}

func ShowLotId() string {
	data := storage.Get(storage.GetReadOnlyContext(), lotKey)
	if data == nil {
		return "nil"
	}

	return string(data.([]byte))
}
