package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"

	nftmarket "nft/wrappers/market"
	nicenamesnft "nft/wrappers/nep11"
	awesomeneotoken "nft/wrappers/nep17"

	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/actor"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/gas"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/nep17"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM) // если наше приложение зависнет, то нажав ctrl+c,
	// мы прервем выполнение текцщей операции

	rpcCli, err := rpcclient.New(ctx, "http://localhost:30333", rpcclient.Options{}) // rpc-client - клиент, который делает запросы к ноде блокчейна
	// (на чтение или на запись - любые), http://localhost:30333 - эндпоинт, на котором слушает наша единственная нода
	die(err)

	w, err := wallet.NewWalletFromFile("../../frostfs-aio/wallets/wallet1.json") // путь до кошелька, откуда будем брать SK и GAS для подписи операций на запись
	die(err)
	acc := w.GetAccount(w.GetChangeAddress()) // для работы нам нужен не только кошелек, но и аккаунт, привязанный к нему (адрес).
	// В одном кошельке мб не один адрес (например в morph/node-wallet их 3), но если (как в нашем случае с wallet1) он всего один, то можем использовать w.GetChangeAddress()
	// которая нам его и вернет
	err = acc.Decrypt("", w.Scrypt) // пароль от кошелька
	die(err)

	act, err := actor.NewSimple(rpcCli, acc) // создали actora, который реализует интерфейс Actor из rpc_wrapper.
	// Знает, куда откправлять запросы (т.к мы передали ему созданного rpc_client), и каким ключом подписывать транзакции (т.к. передали аккаунт кошелька)
	die(err)

	hashNFT, err := util.Uint160DecodeStringLE("843e4d56ef7ba813fb3accbf55fdddf687da4245")
	die(err)
	contractNFT := nicenamesnft.New(act, hashNFT) // создали структуру nft контракта
	_ = contractNFT

	// contractNFT.TokensOfList(acc.ScriptHash()) // удобно можем вызывать разные функции nft контракта, не заморачиваясь с командами в консоли

	hashMarket, err := util.Uint160DecodeStringLE("ef7892e35d530e8b260984d48838139ffea980fd")
	die(err)
	contractMarket := nftmarket.New(act, hashMarket) // создали структуру market контракта
	_ = contractMarket

	hashToken, err := util.Uint160DecodeStringLE("4701ad42c4bedc83467f52c6263dd6c929cf2d54")
	die(err)
	contractToken := awesomeneotoken.New(act, hashToken) // создали структуру mytkn контракта

	contractGAS := gas.New(act) // для нативного контракта GAS не нужно использовать кастомною обертку

	printBalances(contractGAS, contractToken, acc.ScriptHash())
	// printMarketNFT(contractMarket)
	// printNFTs(contractNFT, acc.ScriptHash())
	// sellNFT(act, contractNFT, hashMarket, "my-itmo-nft")
	// buyNFT(act, contractMarket, contractToken, hashMarket, "my-itmo-nft")
	// createNFT(act, contractGAS, hashNFT, "my-itmo-nft")
	// transferMyTKN(act, contractMarket, acc.ScriptHash())

}

func createNFT(act *actor.Actor, contractGAS *nep17.Token, hashNFT util.Uint160, name string) {
	_, err := act.WaitSuccess(contractGAS.Transfer(act.Sender(), hashNFT, big.NewInt(10_0000_0000), name)) // переводим деньги с wallet1 (т.к actor создан поверх
	// wallet1) контракту nft и как и раньше получаем новый nft с заданным именем name
	// WaitSuccess должидается вхождения tx в блок и проверяет успешность ее применения
	die(err)
}

func printBalances(contractGAS *nep17.Token, contractToken *awesomeneotoken.Contract, owner util.Uint160) {
	fmt.Println("Balances:")

	gasBalance, err := contractGAS.BalanceOf(owner)
	die(err)
	fmt.Println("GAS balance: ", gasBalance.Uint64())

	mytknBalance, err := contractToken.BalanceOf(owner)
	die(err)
	fmt.Println("MYTKN balance: ", mytknBalance.Uint64())
	fmt.Println()
}

func sellNFT(act *actor.Actor, c *nicenamesnft.Contract, to util.Uint160, name string) {
	for _, nft := range listNFTs(c, act.Sender()) {
		if nft.Name == name {
			_, err := act.WaitSuccess(c.Transfer(to, nft.ID, nil))
			die(err)
			return
		}
	}

	fmt.Println("not found nft: ", name)
	fmt.Println()
}

func transferMyTKN(act *actor.Actor, c *nftmarket.Contract, to util.Uint160) {
	_, err := act.WaitSuccess(c.TransferTokens(to, big.NewInt(10_0000_0000)))
	die(err)
}

func buyNFT(act *actor.Actor, c *nftmarket.Contract, ct *awesomeneotoken.Contract, marketHash util.Uint160, name string) {
	for _, nft := range listMarketNFT(c) {
		if nft.Name == name {
			_, err := act.WaitSuccess(ct.Transfer(act.Sender(), marketHash, big.NewInt(10_0000_0000), nft.ID))
			die(err)
			return
		}
	}

	fmt.Println("not found market nft: ", name)
	fmt.Println()
}

func printNFTs(c *nicenamesnft.Contract, owner util.Uint160) {
	res := listNFTs(c, owner)
	fmt.Println("owner nfts:")
	fmt.Println(res)
	fmt.Println()
}

func listNFTs(c *nicenamesnft.Contract, owner util.Uint160) []NFTItem {
	res, err := c.TokensOfExpanded(owner, 10) // res - идентификаторы токенов, которые есть
	die(err)

	var list []NFTItem
	for _, re := range res {
		prop, err := c.Properties(re)
		die(err)

		list = append(list, parseMap(prop.Value().([]stackitem.MapElement)))
	}

	return list
}

func printMarketNFT(c *nftmarket.Contract) {
	res := listMarketNFT(c)
	fmt.Println("market nfts:")
	fmt.Println(res)
	fmt.Println()
}

func listMarketNFT(c *nftmarket.Contract) []NFTItem {
	res, err := c.List()
	die(err)

	var list []NFTItem
	for _, re := range res {
		items := re.Value().([]stackitem.MapElement)
		list = append(list, parseMap(items))
	}

	return list
}

type NFTItem struct {
	ID         []byte
	Owner      util.Uint160
	Name       string
	PrevOwners int
	Created    int
	Bought     int
}

func (n NFTItem) String() string {
	return fmt.Sprintf("%x\nowner: %s\nname: %s\nprevOwners: %d\ncreated block: %d\nlast bought block: %d\n",
		n.ID, address.Uint160ToString(n.Owner), n.Name, n.PrevOwners, n.Created, n.Bought)
}

func parseMap(items []stackitem.MapElement) NFTItem {
	var res NFTItem

	for _, item := range items {
		k, err := item.Key.TryBytes()
		die(err)
		v, err := item.Value.TryBytes()
		die(err)

		kStr := string(k)

		switch kStr {
		case "id":
			res.ID = v
		case "owner":
			res.Owner, err = address.StringToUint160(string(v))
			die(err)
		case "name":
			res.Name = string(v)
		case "preOwners":
			res.PrevOwners, err = strconv.Atoi(string(v))
			die(err)
		case "created":
			res.Created, err = strconv.Atoi(string(v))
			die(err)
		case "bought":
			res.Bought, err = strconv.Atoi(string(v))
			die(err)
		}
	}

	return res
}

func die(err error) { // для debug
	if err == nil {
		return
	}

	debug.PrintStack()
	_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
