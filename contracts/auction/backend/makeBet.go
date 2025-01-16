package main

import (
	"encoding/binary"
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/network/payload"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/notary"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"go.uber.org/zap"
)

func (s *Server) proceedMainTxMakeBet(nAct *notary.Actor, notaryEvent *result.NotaryRequestEvent) error {
	err := nAct.Sign(notaryEvent.NotaryRequest.MainTransaction)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	mainHash, fallbackHash, vub, err := nAct.Notarize(notaryEvent.NotaryRequest.MainTransaction, nil)
	s.log.Info("notarize sending",
		zap.String("hash", notaryEvent.NotaryRequest.Hash().String()),
		zap.String("main", mainHash.String()), zap.String("fb", fallbackHash.String()),
		zap.Uint32("vub", vub))

	_, err = nAct.Wait(mainHash, fallbackHash, vub, err) // ждем, пока какая-нибудь tx будет принята
	if err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	return nil
}

func validateNotaryRequestMakeBet(req *payload.P2PNotaryRequest, s *Server) (util.Uint160, int, error) {
	args, contractHash, err := validateNotaryRequestPerProcessing(req)
	if err != nil {
		return util.Uint160{}, 0, err
	}

	contractHashExpected := s.auctionHash

	if !contractHash.Equals(contractHashExpected) {
		return util.Uint160{}, 0, fmt.Errorf("unexpected contract hash: %s", contractHash)
	}

	if len(args) != 2 { // makeBet принимает ровно 2 аргумента
		return util.Uint160{}, 0, fmt.Errorf("invalid param length: %d", len(args))
	}

	bet := int(binary.LittleEndian.Uint16(args[0].Param()))

	sh, err := util.Uint160DecodeBytesBE(args[1].Param())
	if err != nil {
		return util.Uint160{}, 0, fmt.Errorf("could not decode script hash: %w", err)
	}

	return sh, bet, err
}
