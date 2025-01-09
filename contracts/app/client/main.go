package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"git.frostfs.info/TrueCloudLab/hrw"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/actor"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/notary"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/unwrap"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/vmstate"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"github.com/spf13/viper"
)

const (
	cfgRPCEndpoint  = "rpc_endpoint"
	cfgBackendKey   = "backend_key"
	cfgWallet       = "wallet"
	cfgPassword     = "password"
	cfgNyanContract = "nyan_contract"
	cfgBackendURL   = "backend_url"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if len(os.Args) != 2 {
		die(fmt.Errorf("invalid args: %v", os.Args))
	}

	viper.GetViper().SetConfigType("yml")

	f, err := os.Open(os.Args[1])
	die(err)
	die(viper.GetViper().ReadConfig(f))
	die(f.Close())

	rpcCli, err := rpcclient.New(ctx, viper.GetString(cfgRPCEndpoint), rpcclient.Options{})
	die(err)

	backendKey, err := keys.NewPublicKeyFromString(viper.GetString(cfgBackendKey))
	die(err)

	w, err := wallet.NewWalletFromFile(viper.GetString(cfgWallet))
	die(err)
	acc := w.GetAccount(w.GetChangeAddress())
	err = acc.Decrypt(viper.GetString(cfgPassword), w.Scrypt)
	die(err)

	contractHash, err := util.Uint160DecodeStringLE(viper.GetString(cfgNyanContract))
	die(err)

	die(claimNotaryDeposit(acc))
	die(makeNotaryRequest(backendKey, acc, rpcCli, contractHash))
}

func claimNotaryDeposit(acc *wallet.Account) error {
	resp, err := http.Get(viper.GetString(cfgBackendURL) + "/notary-deposit/" + acc.Address)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notary deposit failed: %d, %s", resp.StatusCode, resp.Status)
	}

	return nil
}

func makeNotaryRequest(backendKey *keys.PublicKey, acc *wallet.Account, rpcCli *rpcclient.Client, contractHash util.Uint160) error {
	coSigners := []actor.SignerAccount{
		{
			Signer: transaction.Signer{ // первый подписант - backend, который будет платить за tx, когда она примется. Мы не знаем его  SK, поэтому ставим PK
				Account: backendKey.GetScriptHash(),
				Scopes:  transaction.None,
			},
			Account: notary.FakeSimpleAccount(backendKey),
		},
		{
			Signer: transaction.Signer{
				Account: acc.ScriptHash(), // следующий подписант - client, данная программа, она знает свой SK, поэтому ставит его
				Scopes:  transaction.Global,
			},
			Account: acc,
		},
	}

	nyanCat, err := getFreeNyanCat(rpcCli, acc, contractHash) // находит свободную гифку
	if err != nil {
		return fmt.Errorf("get free cat: %w", err)
	}

	nAct, err := notary.NewActor(rpcCli, coSigners, acc) // обертка актора (клиенты; подписанты; акк, который отправляет tx)
	if err != nil {
		return err
	}

	tx, err := nAct.MakeTunedCall(contractHash, "mint", nil, nil, acc.ScriptHash(), nyanCat) // tx = вызов метода mint на
	// контракте nft (nyanCat - имя гифки) - себе получаем гифку
	if err != nil {
		return err
	}

	mainHash, fallbackHash, vub, err := nAct.Notarize(tx, err) // отправка нотариального запроса
	if err != nil {
		return err
	}

	fmt.Printf("Notarize sending: mainHash - %v, fallbackHash - %v, vub - %d\n", mainHash, fallbackHash, vub)

	res, err := nAct.Wait(mainHash, fallbackHash, vub, err) // ждем пока примется какя-нибудь tx  (основная (main), если все хорошо, либо fallBack)
	if err != nil {
		return err
	}

	if res.VMState != vmstate.Halt {
		return fmt.Errorf("invalid vm state: %s", res.VMState)
	}

	if len(res.Stack) != 1 {
		return fmt.Errorf("invalid stack size: %d", len(res.Stack))
	}

	tokenID, err := res.Stack[0].TryBytes() // если все хорошо, значит токен создан, берем его со стека
	if err != nil {
		return err
	}

	fmt.Println("new token id", hex.EncodeToString(tokenID))

	return nil
}

var listOfCats = []string{
	"404.gif",
	"america.gif",
	"balloon.gif",
	"bday.gif",
	"bloon.gif",
	"breakfast.gif",
	"daft.gif",
	"dub.gif",
	"easter.gif",
	"elevator.gif",
	"fat.gif",
	"fiesta.gif",
	"floppy.gif",
	"ganja.gif",
	"gb.gif",
	"grumpy.gif",
	"j5.gif",
	"jacksnyan.gif",
	"jamaicnyan.gif",
	"jazz.gif",
	"jazzcat.gif",
	"manyan.gif",
	"melon.gif",
	"mexinyan.gif",
	"mummy.gif",
	"newyear.gif",
	"nyanamerica.gif",
	"nyancat.gif",
	"nyancoin.gif",
	"nyandoge.gif",
	"nyaninja.gif",
	"nyanvirus.gif",
	"oldnewyear.gif",
	"oldnyan.gif",
	"original.gif",
	"paddy.gif",
	"pikanyan.gif",
	"pirate.gif",
	"pumpkin.gif",
	"rasta.gif",
	"retro.gif",
	"sad.gif",
	"sadnyan.gif",
	"skrillex.gif",
	"slomo.gif",
	"slomocat.gif",
	"smooth.gif",
	"smurfcat.gif",
	"star.gif",
	"starsheep.gif",
	"tacnayn.gif",
	"tacodog.gif",
	"technyancolor.gif",
	"toaster.gif",
	"vday.gif",
	"watermelon.gif",
	"wtf.gif",
	"xmas.gif",
	"xmasold.gif",
	"zombie.gif",
}

func getFreeNyanCat(cli *rpcclient.Client, acc *wallet.Account, contractHash util.Uint160) (string, error) {
	// пробегает по списку гифок, определяет свободна или нет, дергая ownerOf. Найдя первую свободную, возвращает
	indexes := make([]uint64, len(listOfCats))
	for i := range indexes {
		indexes[i] = uint64(i)
	}

	act, err := actor.NewSimple(cli, acc)
	if err != nil {
		return "", err
	}

	h := hrw.Hash(acc.ScriptHash().BytesBE()) // сортировка опциональна, может быть какая-то другая логика поиска нужной гифки (ex ML), это просто
	// как пример того, в каком порядке можно обходить список всех возможных гифок в поиске свободной
	// если каждый клиент пойдет по порядку, все начнут с начала, а свободны только последние 2, то они все пройдут весь список - очень неэффективно, пусть
	// идут с разных концов, используем рандеву-хэширование
	hrw.Sort(indexes, h)

	var cat string
	for _, index := range indexes {
		cat = listOfCats[index]

		hash := sha256.New()
		hash.Write([]byte(cat))
		tokenID := hash.Sum(nil)

		if _, err := unwrap.Uint160(act.Call(contractHash, "ownerOf", tokenID)); err != nil {
			break
		}
	}

	if cat == "" {
		return "", errors.New("all cats are taken") // не осталось свободных токенов
	}

	return cat, nil
}

func die(err error) {
	if err == nil {
		return
	}

	debug.PrintStack()
	_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
