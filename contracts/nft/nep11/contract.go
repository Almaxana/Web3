package contract

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/iterator"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/crypto"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/gas"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/ledger"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/management"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/std"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
	"github.com/nspcc-dev/neo-go/pkg/interop/util"
)

// Prefixes used for contract data storage.

// в storage пары ключ-знаечние. Ключи, по которым хранятся балансы, начинаются с "b";
// ключи, по которым хранятся структуры токенов, начинаются с "t" и тд

// таким обраом мы храним:
// баланс для каждого пользователя: (b + ownerAddress) - по сколько у него nft
// список токенов для каждого пользователя: (a + ownerAddress + tokenId) - tokenId
// сами токены (t + tockenId) - serializedToken
// адрес владельца контракта
// общее кол-во токенов, созданных за время жизни контракта

const (
	balancePrefix = "b"
	accountPrefix = "a"
	tokenPrefix   = "t"

	ownerKey       = 'o'
	totalSupplyKey = 's'
)

const (
	minNameLen = 3
)

type NFTItem struct {
	ID    []byte
	Owner interop.Hash160
	Name  string // можно было бы использовать как id, но строка мб очень
	// длинная, а в bc размер ключа <= 64 байта. Поэтому будем брать id = hash(Name)
	PrevOwners int
	Created    int // момент создания (высота блоков)
	Bought     int // момент покупки (высота блоков)
}

// реализация nft стандарта

func _deploy(data interface{}, isUpdate bool) { // вызывается при деплое (и обновлении) контракта
	if isUpdate {
		return
	}

	args := data.(struct {
		Admin interop.Hash160 // адрес владельца
	})

	if args.Admin == nil { // проверяем что адрес владельца задан
		panic("invalid admin")
	}

	if len(args.Admin) != 20 { // проверяем что адрес владельца имеет корректную длину
		panic("invalid admin hash length")
	}

	ctx := storage.GetContext() // сохраняем в storage контракта адрес админа этого контракта
	storage.Put(ctx, ownerKey, args.Admin)

	name := ownerAddress(args.Admin) // преобразуем адрес (hash из 20 байтов) в строку
	nft := NFTItem{                  // сразу дополнительно создаем nft = строка с адресом, который нам передали
		ID:         crypto.Sha256([]byte(name)),
		Owner:      args.Admin,
		Name:       name,
		PrevOwners: 0,
		Created:    ledger.CurrentIndex(), // ledger = пакет, который может сказать в какой транзакции мы
		// сейчас находимся, номер блока и тп. В данном случае получаем текущую высоту bc = номер блока
		Bought: ledger.CurrentIndex(),
	}
	setNFT(ctx, nft.ID, nft)         // сохраняем токен в storage
	addToBalance(ctx, nft.Owner, 1)  // обновляем баланс пользователя
	addToken(ctx, nft.Owner, nft.ID) // обновляем список nft данного пользователя, чтобы потом
	// удобно сразу посмотреть список nft нужного пользователя

	storage.Put(ctx, totalSupplyKey, 1) // по ключу хранящему общее кол-во токенов сохраняем 1, т.к создали новый токен только что
}

// Symbol returns token symbol, it's NICENAMES.
func Symbol() string {
	return "NICENAMES"
}

// Decimals returns token decimals, this NFT is non-divisible, so it's 0.
func Decimals() int {
	return 0 // неделимый nft, мб передан от одного ownera другому только полностью
}

// TotalSupply is a contract method that returns the number of tokens minted.
func TotalSupply() int {
	return storage.Get(storage.GetReadOnlyContext(), totalSupplyKey).(int)
	// totalSupplyKey - общее число созданных в контракте никнеймов
}

// BalanceOf returns the number of tokens owned by the specified address.
func BalanceOf(holder interop.Hash160) int { // сколько токенов (никнеймов) есть у конкретного пользователя
	if len(holder) != 20 {
		panic("bad owner address")
	}
	ctx := storage.GetReadOnlyContext()
	return getBalanceOf(ctx, mkBalanceKey(holder))
}

// OwnerOf returns the owner of the specified token.
func OwnerOf(token []byte) interop.Hash160 { // узнать владельца токена
	ctx := storage.GetReadOnlyContext()
	return getNFT(ctx, token).Owner
}

// Properties returns properties of the given NFT.
func Properties(token []byte) map[string]string { // справочная инфа о nft токене
	ctx := storage.GetReadOnlyContext()
	nft := getNFT(ctx, token)

	result := map[string]string{
		"id":         string(nft.ID),
		"owner":      ownerAddress(nft.Owner),
		"name":       nft.Name,
		"prevOwners": std.Itoa10(nft.PrevOwners),
		"created":    std.Itoa10(nft.Created),
		"bought":     std.Itoa10(nft.Bought),
	}
	return result
}

// Tokens returns an iterator that contains all the tokens minted by the contract.
// Возвращает список всех токенов, созданных за время жизни контракта, но их может быть очень много, а на
// стековой машине есть ограничение по кол-ву токенов, находящихся на стеке (2048). Т.е. если будем возвращать не iterator.Iterator,
// а []string список строк, то он может не поместиться на стек и будет плохо. Поэтому возвращаем итератор и будем получать список не целиком,
// а порционно => не упремся в ограничение стека
func Tokens() iterator.Iterator {
	ctx := storage.GetReadOnlyContext()
	key := []byte(tokenPrefix)
	iter := storage.Find(ctx, key, storage.RemovePrefix|storage.KeysOnly) // ищем все элементы, которые имеют в ключе префикс токена t
	// и вместе с этим убираем префикс, берем непосредственно сам ключ
	return iter
}

// похожая функция на Tokens, но возвращает список. Будем использовать, когда будем дергать методы через консоль, а там с итераторами
// работать неудобно, лучше со списками. Это прокатит, пока токены помещаются на стек. Но Tokens должен быть обязательно для реализации
// стандарта nft
func TokensList() []string {
	ctx := storage.GetReadOnlyContext()
	key := []byte(tokenPrefix)
	iter := storage.Find(ctx, key, storage.RemovePrefix|storage.KeysOnly)
	keys := []string{}
	for iterator.Next(iter) {
		k := iterator.Value(iter)
		keys = append(keys, k.(string))
	}
	return keys
}

// TokensOf returns an iterator with all tokens held by the specified address.
func TokensOf(holder interop.Hash160) iterator.Iterator { // holder = id пользователя
	if len(holder) != 20 {
		panic("bad owner address")
	}
	ctx := storage.GetReadOnlyContext()
	key := mkAccountPrefix(holder)
	iter := storage.Find(ctx, key, storage.ValuesOnly)
	return iter
}

func TokensOfList(holder interop.Hash160) [][]byte {
	if len(holder) != 20 {
		panic("bad owner address")
	}
	ctx := storage.GetReadOnlyContext()
	key := mkAccountPrefix(holder)
	res := [][]byte{}
	iter := storage.Find(ctx, key, storage.ValuesOnly)
	for iterator.Next(iter) {
		res = append(res, iterator.Value(iter).([]byte))
	}
	return res
}

// Transfer token from its owner to another user, notice that it only has three
// parameters because token owner can be deduced from token ID itself.
func Transfer(to interop.Hash160, token []byte, data any) bool { // to - кому, from не пишем, т.к текущего владельца можно
	// узнать из самого токена
	if len(to) != 20 {
		panic("invalid 'to' address")
	}
	ctx := storage.GetContext()
	nft := getNFT(ctx, token) // получили nft в виде структуры NFTItem
	from := nft.Owner         // узнали, кто его хозяин

	if !runtime.CheckWitness(from) { // проверяем, что перевести токен кому-то другому собирается сам владелец
		// чтобы не случилось такого, что без нашего ведома распоряжаются нашими токенами
		return false
	}

	if !from.Equals(to) { // проверяем, что переводим токен не самому себе (можно конечно и самому себе, но от этого ничего не
		// поменяется, поэтому можно ничего не делать)
		nft.Owner = to
		nft.Bought = ledger.CurrentIndex()
		nft.PrevOwners += 1
		setNFT(ctx, token, nft)

		addToBalance(ctx, from, -1)
		removeToken(ctx, from, token) // удаляем токен из списка токенов предыдущего владельца
		addToBalance(ctx, to, 1)
		addToken(ctx, to, token)
	}

	postTransfer(from, to, token, data) // различная нотификация
	return true
}

func getNFT(ctx storage.Context, token []byte) NFTItem {
	key := mkTokenKey(token)
	val := storage.Get(ctx, key)
	if val == nil {
		panic("no token found")
	}

	serializedNFT := val.([]byte)
	deserializedNFT := std.Deserialize(serializedNFT) // хранили в storage в сериализованном виде, теперь чтобы работать
	// с ним, достав из storage, надо эти байты обратно в десериализовать
	return deserializedNFT.(NFTItem)
}

func nftExists(ctx storage.Context, token []byte) bool {
	key := mkTokenKey(token)
	return storage.Get(ctx, key) != nil
}

func setNFT(ctx storage.Context, token []byte, item NFTItem) {
	key := mkTokenKey(token)
	val := std.Serialize(item) // преобразование довольно сложной структуры item в последовательность
	// байт, потому что storage.Put() может принять либо int, либо []byte, либо bool, либо string (несмотря
	// на то, что написано в сигнатуре принимаемый тип any)
	storage.Put(ctx, key, val)
}

// postTransfer emits Transfer event and calls onNEP11Payment if needed.
func postTransfer(from interop.Hash160, to interop.Hash160, token []byte, data any) {
	runtime.Notify("Transfer", from, to, 1, token) // transfer от кого кому сколько что. Сразу ставим 1, т.к токен неделимый
	// и всегда передача будет происходить 1 токена
	if management.GetContract(to) != nil { // если переводим токен не просто другому пользователю (кошельку), а этот
		// кошелек является контрактом, то на этом контракте надо вызвать функцию onNEP11Payment, чтобы контракт узнал, что ему
		// что-то перевели и мог это обработать как-то
		contract.Call(to, "onNEP11Payment", contract.All, from, 1, token, data)
	}
}

// OnNEP17Payment mints tokens if at least 10 GAS is provided. You don't call
// this method directly, instead it's called by GAS contract when you transfer
// GAS from your address to the address of this NFT contract.

// эта функция позволяет создать nft за (газ/неотокен/самописный токен) = любой токен, в данном случае мы реализовали
// активацию при переводе газа (в коде есть на это проверка gas.Hash) (без нее создавался только один токен при деплое контракта).
// Но не напрямую непосредственно создавать nft, а при переводе газа на счет контракта
// будет создаваться nft
func OnNEP17Payment(from interop.Hash160, amount int, data any) {
	defer func() { // defer означает, что эта функция будет выполнена перед выходом из текущей функции (OnNEP17Payment)
		// в том числе и при аварийном выходе.
		// в данном случае функция проверяет, произошла ли паника, с помощью recover(). Если произошла (recover() вернул НЕ nil)
		// то залогирует ошибку и завершит выполнение смарт контракта
		if r := recover(); r != nil {
			runtime.Log(r.(string))
			util.Abort()
		}
	}()

	callingHash := runtime.GetCallingScriptHash()
	if !callingHash.Equals(gas.Hash) { // эмитим новый nft, только если нам перевели именно газ (если какой-то другой токен,
		// то работать с таким не будем)
		panic("only GAS is accepted")
	}

	name := data.(string)
	if len(name) < 3 { // хотим min длина никнейма 3 символа
		panic("name length at least 3 character")
	}

	price := 10_0000_0000 // min цена за сколько готовы создать новый nft
	if len(name) < 10 {   // все хотят короткие никнеймы, за такие надо платить больше
		price += 5_0000_0000
	}
	if len(name) < 6 { // за еще более короткие - еще больше
		price += 5_0000_0000
	}

	if amount < price { // если недостаточно нам перевели за создание, то создавать nft не будем
		panic("insufficient GAS for minting NFT")
	}

	ctx := storage.GetContext()
	tokenID := crypto.Sha256([]byte(name))
	if nftExists(ctx, tokenID) {
		panic("token already exists")
	}

	nft := NFTItem{
		ID:         tokenID,
		Owner:      from,
		Name:       name,
		PrevOwners: 0,
		Created:    ledger.CurrentIndex(),
		Bought:     ledger.CurrentIndex(),
	}
	setNFT(ctx, tokenID, nft)
	addToBalance(ctx, from, 1)
	addToken(ctx, from, tokenID)

	total := storage.Get(ctx, totalSupplyKey).(int) + 1
	storage.Put(ctx, totalSupplyKey, total)

	postTransfer(nil, from, tokenID, nil)
}

// mkAccountPrefix creates DB key-prefix for the account tokens specified
// by concatenating accountPrefix and account address.
func mkAccountPrefix(holder interop.Hash160) []byte {
	res := []byte(accountPrefix)
	return append(res, holder...)
}

// mkBalanceKey creates DB key for the account specified by concatenating balancePrefix
// and account address.
func mkBalanceKey(holder interop.Hash160) []byte {
	res := []byte(balancePrefix)
	return append(res, holder...)
}

// mkTokenKey creates DB key for the token specified by concatenating tokenPrefix
// and token ID.
func mkTokenKey(tokenID []byte) []byte {
	res := []byte(tokenPrefix)
	return append(res, tokenID...)
}

// getBalanceOf returns the balance of an account using database key.
func getBalanceOf(ctx storage.Context, balanceKey []byte) int {
	val := storage.Get(ctx, balanceKey)
	if val != nil {
		return val.(int)
	}
	return 0
}

// addToBalance adds an amount to the account balance. Amount can be negative.
func addToBalance(ctx storage.Context, holder interop.Hash160, amount int) {
	key := mkBalanceKey(holder)   // сформировали ключ баланса (с префиксом баланса)
	old := getBalanceOf(ctx, key) // смотрим текущий баланс
	old += amount
	if old > 0 {
		storage.Put(ctx, key, old)
	} else {
		storage.Delete(ctx, key) // чтобы не тратить доп газ на храниние данных
	}
}

// addToken adds a token to the account.
func addToken(ctx storage.Context, holder interop.Hash160, token []byte) {
	key := mkAccountPrefix(holder) // формируем ключ для аккаунта
	storage.Put(ctx, append(key, token...), token)
}

// removeToken removes the token from the account.
func removeToken(ctx storage.Context, holder interop.Hash160, token []byte) {
	key := mkAccountPrefix(holder)
	storage.Delete(ctx, append(key, token...))
}

func ownerAddress(owner interop.Hash160) string {
	b := append([]byte{0x35}, owner...)
	return std.Base58CheckEncode(b)
}
