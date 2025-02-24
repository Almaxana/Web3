// Code generated by neo-go contract generate-rpcwrapper --manifest <file.json> --out <file.go> [--hash <hash>] [--config <config>]; DO NOT EDIT.

// Package nftmarket contains RPC wrappers for NFT market contract.
package nftmarket

import (
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/unwrap"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"math/big"
)

// Invoker is used by ContractReader to call various safe methods.
type Invoker interface {
	Call(contract util.Uint160, operation string, params ...any) (*result.Invoke, error)
}

// Actor is used by Contract to call state-changing methods.
type Actor interface {
	Invoker

	MakeCall(contract util.Uint160, method string, params ...any) (*transaction.Transaction, error)
	MakeRun(script []byte) (*transaction.Transaction, error)
	MakeUnsignedCall(contract util.Uint160, method string, attrs []transaction.Attribute, params ...any) (*transaction.Transaction, error)
	MakeUnsignedRun(script []byte, attrs []transaction.Attribute) (*transaction.Transaction, error)
	SendCall(contract util.Uint160, method string, params ...any) (util.Uint256, uint32, error)
	SendRun(script []byte) (util.Uint256, uint32, error)
}

// ContractReader implements safe contract methods.
type ContractReader struct {
	invoker Invoker
	hash    util.Uint160
}

// Contract implements all contract methods.
type Contract struct {
	ContractReader
	actor Actor
	hash  util.Uint160
}

// NewReader creates an instance of ContractReader using provided contract hash and the given Invoker.
func NewReader(invoker Invoker, hash util.Uint160) *ContractReader {
	return &ContractReader{invoker, hash}
}

// New creates an instance of Contract using provided contract hash and the given Actor.
func New(actor Actor, hash util.Uint160) *Contract {
	return &Contract{ContractReader{actor, hash}, actor, hash}
}

// List invokes `list` method of contract.
func (c *ContractReader) List() ([]stackitem.Item, error) {
	return unwrap.Array(c.invoker.Call(c.hash, "list"))
}

// TransferTokens creates a transaction invoking `transferTokens` method of the contract.
// This transaction is signed and immediately sent to the network.
// The values returned are its hash, ValidUntilBlock value and error if any.
func (c *Contract) TransferTokens(to util.Uint160, amount *big.Int) (util.Uint256, uint32, error) {
	return c.actor.SendCall(c.hash, "transferTokens", to, amount)
}

// TransferTokensTransaction creates a transaction invoking `transferTokens` method of the contract.
// This transaction is signed, but not sent to the network, instead it's
// returned to the caller.
func (c *Contract) TransferTokensTransaction(to util.Uint160, amount *big.Int) (*transaction.Transaction, error) {
	return c.actor.MakeCall(c.hash, "transferTokens", to, amount)
}

// TransferTokensUnsigned creates a transaction invoking `transferTokens` method of the contract.
// This transaction is not signed, it's simply returned to the caller.
// Any fields of it that do not affect fees can be changed (ValidUntilBlock,
// Nonce), fee values (NetworkFee, SystemFee) can be increased as well.
func (c *Contract) TransferTokensUnsigned(to util.Uint160, amount *big.Int) (*transaction.Transaction, error) {
	return c.actor.MakeUnsignedCall(c.hash, "transferTokens", nil, to, amount)
}
