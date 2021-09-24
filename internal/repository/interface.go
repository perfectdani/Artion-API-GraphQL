// Package repository implements persistent data access and processing.
package repository

import (
	"artion-api-graphql/internal/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Repository defines interface used to interact with the persistent storage
// and the blockchain node.
type Repository interface {

	StoreTokenEvent(*types.TokenEvent) error

	ListTokenEvents(nftAddr common.Address, tokenId hexutil.Big, cursor types.Cursor, count int) (out *types.TokenEventList, err error)

}