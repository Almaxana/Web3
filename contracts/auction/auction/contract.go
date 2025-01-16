package auction

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/lib/address"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/management"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
)

// Prefixes used for contract data storage.
const (
	initBetKey         = "i"
	currentBetKey      = "c"
	lotKey             = "l" // nft id
	organizerKey       = "o" // organizer of the auction
	potentialWinnerKey = "w" // owner of the last bet
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

	// 96e218fa4570c97f8832ee2b69d54369a78cd818 - адрес nft контракта, чтобы получить из него nftContractHashString, надо вручную в консоли
	// написать команду neo-go util convert 96e218fa4570c97f8832ee2b69d54369a78cd818 и взять из нее LE ScriptHash to Address
	nftContractHashString := "NNBLpUgJdq3zVR425TwrafEXgqxYDwXp6s"
	ownerOfLot := contract.Call(address.ToHash160(nftContractHashString), "ownerOf", contract.All, lotId).(interop.Hash160)
	if !ownerOfLot.Equals(auctionOwner) {
		panic("you can't start auction with lot " + string(lotId) + " because you're not its owner")
	}

	storage.Put(ctx, organizerKey, auctionOwner)
	storage.Put(ctx, lotKey, lotId)
	storage.Put(ctx, initBetKey, initBet)
	storage.Put(ctx, currentBetKey, initBet)
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

}

func Finish(finishInitiator interop.Hash160) {
	ctx := storage.GetContext()

	auctionOwner := storage.Get(ctx, organizerKey)
	if !finishInitiator.Equals(auctionOwner) {
		panic("only organizer of the auction can finish it")
	}

	potentialWinner := storage.Get(ctx, potentialWinnerKey).(interop.Hash160)
	if potentialWinner.Equals(auctionOwner) {
		panic("nobody made bets, lot isn't selled")
	}

	lotId := storage.Get(ctx, lotKey)

	nftContractHashString := "NNBLpUgJdq3zVR425TwrafEXgqxYDwXp6s"
	res := contract.Call(address.ToHash160(nftContractHashString), "transfer", contract.All, potentialWinner, lotId, nil).(bool)
	if !res {
		panic("error while transfer token to winner")
	}

	storage.Delete(ctx, initBetKey)
	storage.Delete(ctx, currentBetKey)
	storage.Delete(ctx, potentialWinnerKey)
	storage.Delete(ctx, organizerKey)
	storage.Delete(ctx, lotKey)

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
