// Package svc implements monitoring and scanning services of the API server.
package svc

import (
	"artion-api-graphql/internal/repository"
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	eth "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"time"
)

const (
	// obsBlocksNotificationTickInterval represents the interval
	// in which observed blocks are notified to repository.
	obsBlocksNotificationTickInterval = 10 * time.Second
)

// EventHandler represents a function used to process event log record.
type EventHandler func(*eth.Log)

// logObserver represents the service responsible for processing event logs of interest.
type logObserver struct {
	// mgr represents the Manager instance
	mgr *Manager

	// sigStop represents the signal for closing the observer
	sigStop chan bool

	// outEvent represents an output channel being fed
	// with recognized block events for processing
	inEvents chan *eth.Log

	// currentBlock contains the number of the currently processed block
	currentBlock *big.Int

	// lastProcessedBlock contains the number of the last processed block
	lastProcessedBlock *big.Int

	// topics represents a map of topics to their respective event handlers.
	topics map[common.Hash]EventHandler

	// contracts represents a list of observed contracts.
	contracts []common.Address
}

// newLogObserver creates a new instance of the event logs observer service.
func newLogObserver(mgr *Manager) *logObserver {
	return &logObserver{
		mgr:     mgr,
		sigStop: make(chan bool, 1),
		topics: map[common.Hash]EventHandler{
			/* event ContractCreated(address creator, address nft) */
			common.HexToHash("0x2d49c67975aadd2d389580b368cfff5b49965b0bd5da33c144922ce01e7a4d7b"): newNFTContract,
		},
	}
}

// name provides the name fo the log observer service.
func (lo *logObserver) name() string {
	return "events observer"
}

// init configures the log observer and subscribes it with the manager.
func (lo *logObserver) init() {
	lo.inEvents = lo.mgr.blkObserver.outEvents
	lo.contracts = repository.R().ObservedContractsAddressList()
	lo.mgr.add(lo)
}

// close signals the log observer to terminate.
func (lo *logObserver) close() {
	lo.sigStop <- true
}

// run collects incoming event logs from the channel and processes them using
//
func (lo *logObserver) run() {
	// start the notification ticker
	tick := time.NewTicker(obsBlocksNotificationTickInterval)

	defer func() {
		tick.Stop()
		lo.mgr.closed(lo)
	}()

	for {
		select {
		case <-lo.sigStop:
			return
		case <-tick.C:
			lo.notify()
		case evt := <-lo.inEvents:
			lo.process(evt)
			lo.processed(evt)
		}
	}
}

// process an incoming event
func (lo *logObserver) process(evt *eth.Log) {
	// is this an event from an observed contract?
	if !lo.isObservedContract(evt) {
		return
	}
}

// processed updates the information about the current and processed block number.
func (lo *logObserver) processed(evt *eth.Log) {
	if lo.currentBlock == nil {
		lo.currentBlock = new(big.Int).SetUint64(evt.BlockNumber)
		return
	}

	// the last block is done
	if lo.currentBlock.Uint64() < evt.BlockNumber {
		lo.lastProcessedBlock = lo.currentBlock
		lo.currentBlock = new(big.Int).SetUint64(evt.BlockNumber)
	}
}

// notify the repository about the latest observed block, if any.
func (lo *logObserver) notify() {
	if lo.lastProcessedBlock == nil {
		return
	}
	repository.R().NotifyLastObservedBlock(lo.lastProcessedBlock)
	log.Infof("last processed block is #%d", lo.lastProcessedBlock.Uint64())
}

// isObservedContract checks if the given event log should be investigated and processed.
func (lo *logObserver) isObservedContract(evt *eth.Log) bool {
	for _, adr := range lo.contracts {
		if 0 == bytes.Compare(adr.Bytes(), evt.Address.Bytes()) {
			return true
		}
	}
	return false
}

// addObservedContract is used to extend the list of observed contracts
// with a newly created NFT contract address; subsequent NFT events should be observed on it.
func (lo *logObserver) addObservedContract(adr *common.Address) {
	lo.contracts = append(lo.contracts, *adr)
	log.Infof("new contract %s is now observed", adr.String())
}

// topicsList provides a list of observed topics for blocks event filtering
func (lo *logObserver) topicsList() []common.Hash {
	list := make([]common.Hash, len(lo.topics))

	var i int
	for h := range lo.topics {
		list[i] = h
		i++
	}

	return list
}
