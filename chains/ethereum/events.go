// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"strings"

	emitter "github.com/ChainSafe/ChainBridgeV2/contracts/Emitter"
	receiver "github.com/ChainSafe/ChainBridgeV2/contracts/Receiver"
	msg "github.com/ChainSafe/ChainBridgeV2/message"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func (l *Listener) handleTransferEvent(eventI interface{}) msg.Message {
	log15.Debug("Handling deposit proposal event")
	event := eventI.(ethtypes.Log)

	contractAbi, err := abi.JSON(strings.NewReader(emitter.EmitterABI))
	if err != nil {
		log15.Error("Unable to decode event", err)
	}

	var nftEvent emitter.EmitterNFTTransfer
	err = contractAbi.Unpack(&nftEvent, "NFTTransfer", event.Data)
	if err != nil {
		log15.Error("Unable to unpack NFTTransfer", err)
	}

	// Capture indexed values
	nftEvent.DestChain = event.Topics[1].Big()
	nftEvent.DepositId = event.Topics[2].Big()

	msg := msg.Message{
		Type:        msg.CreateDepositProposalType,
		Source:      l.cfg.id,
		Destination: msg.ChainId(uint8(nftEvent.DestChain.Uint64())),
		// TODO: Can we safely downsize?
		DepositId: uint32(nftEvent.DepositId.Uint64()),
		To:        nftEvent.To.Bytes(),
		Metadata:  nftEvent.Data,
	}

	return msg
}

func (l *Listener) handleVoteEvent(eventI interface{}) msg.Message {
	log15.Debug("handling vote event")
	event := eventI.(ethtypes.Log)

	contractAbi, err := abi.JSON(strings.NewReader(string(receiver.ReceiverABI)))
	if err != nil {
		log15.Error("Unable to decode event", err)
	}

	var depoistEvent receiver.ReceiverDepositProposalCreated
	err = contractAbi.Unpack(&depoistEvent, "DepositProposalCreated", event.Data)
	if err != nil {
		log15.Error("Unable to unpack DepositProposalCreated", err)
	}

	log15.Trace("deposit events", "struct", depoistEvent)

	message := msg.Message{
		Type:        msg.VoteDepositProposalType,
		Destination: l.cfg.id, // We are reading from the receiver, must write to the same contract
		Data:        depoistEvent.Hash[:],
	}
	v := msg.ChainId(depoistEvent.OriginChain.bytes()[0])
	message.EncodeVoteDepositProposalData(depoistEvent.DepositId, depoistEvent.OriginChain, depoistEvent.VoteStatus)
}

func (l *Listener) handleTestDeposit(eventI interface{}) msg.Message {
	event := eventI.(ethtypes.Log)
	data := ethcrypto.Keccak256Hash(event.Topics[0].Bytes()).Bytes()
	return msg.Message{
		Type:     msg.DepositAssetType,
		Metadata: data,
	}
}
