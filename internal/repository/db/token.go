package db

import (
	"artion-api-graphql/internal/types"
	"artion-api-graphql/internal/types/sorting"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/big"
	"time"
)

const (
	// CoTokens is the name of database collection.
	coTokens = "tokens"

	// fiTokenContract is the column storing the address of the NFT token contract.
	fiTokenContract = "contract"

	// fiTokenIsActive is the column storing the NFT token activity mark.
	fiTokenIsActive = "is_active"

	// fiTokenMetadataURI is the column storing the NFT token metadata URI.
	fiTokenMetadataURI = "uri"

	// FiTokenName is the column storing the name of the NFT token.
	fiTokenName = "name"

	// fiTokenDescription is the column storing the description of the NFT token.
	fiTokenDescription = "desc"

	// fiTokenSymbol is the column storing symbol of the token
	fiTokenSymbol = "symbol"

	// fiTokenIpRights is the column storing URL of IP document
	fiTokenIpRights = "ip_rights"

	// fiTokenImageURI is the column storing the image URI of the NFT token.
	fiTokenImageURI = "image"

	// fiTokenImageType is the column storing type of the image of the NFT token.
	fiTokenImageType = "image_type"

	// fiTokenCreatedBy is the column marking creator of the token.
	fiTokenCreatedBy = "created_by"

	// fiTokenHasListingSince is the column marking listed token earliest date/time.
	fiTokenHasListingSince = "listed_since"

	// fiTokenHasAuctionSince is the column marking the earliest time a token is auctioned.
	fiTokenHasAuctionSince = "auction_since"

	// fiTokenHasAuctionUntil is the column marking the latest time a token is auctioned.
	fiTokenHasAuctionUntil = "auction_until"

	// fiTokenHasOfferUntil is the column marking offered token latest date/time.
	fiTokenHasOfferUntil = "offer_until"

	// fiTokenHasBids is the column marking auctioned token with at least one bid.
	fiTokenHasBids = "has_bid"

	// fiTokenLastTrade is the column storing the last trade date/time.
	fiTokenLastTrade = "last_trade"

	// fiTokenLastListing is the column storing the last listing creation date/time.
	fiTokenLastListing = "last_list"

	// fiTokenLastOffer is the column storing the last offer creation date/time.
	fiTokenLastOffer = "last_offer"

	// fiTokenLastAuction is the column storing the latest auction creation date/time.
	fiTokenLastAuction = "last_auction"

	// fiTokenLastBid is the column storing the latest auction bid date/time.
	fiTokenLastBid = "last_bid"

	// fiTokenAmountLastTrade is the column storing the last price token was sold for.
	fiTokenAmountLastTrade = "amo_trade"

	// fiTokenAmountLastOffer is the column storing the last price token was sold for.
	fiTokenAmountLastOffer = "amo_offer"

	// fiTokenAmountLastBid is the column storing the last bid placed on the token.
	fiTokenAmountLastBid = "amo_bid"

	// fiTokenAmountLastList is the column storing the latest price token is listed at.
	fiTokenAmountLastList = "amo_list"

	// fiTokenReservePrice is the column storing reserve price of running auction.
	fiTokenReservePrice = "reserve"

	// fiTokenMinListPrice is the column storing minimal listing price in USD.
	fiTokenMinListPrice = "min_list"

	// fiTokenMinListValid is the column storing end of minimal listing price validity.
	fiTokenMinListValid = "min_list_valid"

	// fiTokenPrice is the column storing price of token in USD aggregated from listings and auctions.
	fiTokenPrice = "price"

	// fiTokenPriceValid is the column storing end of fiTokenPrice validity
	fiTokenPriceValid = "price_valid"

	// fiTokenCategories is the column storing categories ids of the token.
	fiTokenCategories = "categories"

	// fiTokenMetadataUpdate is the column storing the time
	// of the metadata update schedule of the NFT token.
	fiTokenMetadataUpdate = "meta_update"

	// fiTokenMetadataUpdate is the column storing the time
	// of the metadata update schedule of the NFT token.
	fiTokenMetadataUpdateFailures = "meta_failures"
)

// GetToken loads specific NFT token for the given contract address and token ID
func (db *MongoDbBridge) GetToken(contract *common.Address, tokenId *big.Int) (token *types.Token, err error) {
	col := db.client.Database(db.dbName).Collection(coTokens)
	result := col.FindOne(context.Background(), bson.D{{Key: fieldId, Value: types.TokenID(contract, tokenId)}})

	var row types.Token
	if err = result.Decode(&row); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Errorf("can not decode token %s at %s; %s", tokenId.String(), contract.String(), err.Error())
		return nil, err
	}
	return &row, err
}

// StoreToken inserts new NFT token or updates existing token in persistent database.
func (db *MongoDbBridge) StoreToken(token *types.Token) error {
	if token == nil {
		return fmt.Errorf("no value to store")
	}

	// get the collection
	col := db.client.Database(db.dbName).Collection(coTokens)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()

	// try to do the insert
	id := token.ID()
	rs, err := col.UpdateOne(
		ctx,
		bson.D{{Key: fieldId, Value: id}},
		bson.D{
			{Key: "$set", Value: token},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: fieldId, Value: id},
			}},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Errorf("can not store token %s at %s; %s", token.TokenId.String(), token.Contract.String(), err.Error())
		return err
	}
	if rs.UpsertedCount > 0 {
		log.Infof("token %s on contract %s added to database", token.TokenId.String(), token.Contract.String())
	}
	return nil
}

// UpdateToken updates the token data i the database using provided update data set.
func (db *MongoDbBridge) UpdateToken(contract *common.Address, tokenID *big.Int, data bson.D) error {
	// get the collection
	col := db.client.Database(db.dbName).Collection(coTokens)
	rs, err := col.UpdateOne(
		context.Background(),
		bson.D{{Key: fieldId, Value: types.TokenID(contract, tokenID)}},
		bson.D{{Key: "$set", Value: data}},
	)
	if err != nil {
		log.Errorf("can not update token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err.Error())
		return err
	}
	if rs.UpsertedCount > 0 {
		log.Infof("token %s/%s updated", contract.String(), (*hexutil.Big)(tokenID).String())
	}
	return nil
}

// UpdateTokenMetadata updates basic metadata of the NFT token.
func (db *MongoDbBridge) UpdateTokenMetadata(nft *types.Token) error {
	if nft == nil {
		return fmt.Errorf("no value to store")
	}

	return db.UpdateToken(&nft.Contract, (*big.Int)(&nft.TokenId), bson.D{
		{Key: fiTokenName, Value: nft.Name},
		{Key: fiTokenDescription, Value: nft.Description},
		{Key: fiTokenImageURI, Value: nft.ImageURI},
		{Key: fiTokenImageType, Value: nft.ImageType},
		{Key: fiTokenIpRights, Value: nft.IpRights},
		{Key: fiTokenSymbol, Value: nft.Symbol},
		{Key: fiTokenCategories, Value: nft.Categories},
		{Key: fiTokenMetadataUpdate, Value: nft.MetaUpdate},
		{Key: fiTokenMetadataUpdateFailures, Value: nft.MetaFailures},
		{Key: fiTokenIsActive, Value: nft.IsActive},
	})
}

// UpdateTokenMetadataRefreshSchedule sets the NFT metadata update schedule time.
func (db *MongoDbBridge) UpdateTokenMetadataRefreshSchedule(nft *types.Token) error {
	if nft == nil {
		return fmt.Errorf("no value to store")
	}

	return db.UpdateToken(&nft.Contract, (*big.Int)(&nft.TokenId), bson.D{
		{Key: fiTokenMetadataUpdate, Value: nft.MetaUpdate},
		{Key: fiTokenMetadataUpdateFailures, Value: nft.MetaFailures},
	})
}

// TokenMarkOffered marks the given NFT as having offer for the given price.
func (db *MongoDbBridge) TokenMarkOffered(contract *common.Address, tokenID *big.Int, price types.TokenPrice, ts *time.Time) error {
	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenLastOffer, Value: *ts},
		{Key: fiTokenHasOfferUntil, Value: db.OpenOfferUntil(contract, tokenID)},
		{Key: fiTokenAmountLastOffer, Value: price},
	})
}

// TokenMarkListed marks the given NFT as listed for direct sale for the given price.
func (db *MongoDbBridge) TokenMarkListed(contract *common.Address, tokenID *big.Int, price types.TokenPrice, ts *time.Time) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasListingSince = db.OpenListingSince(contract, tokenID)
	t.MinListPrice, t.MinListValid = db.MinListingPrice(contract, tokenID)
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenLastListing, Value: *ts},
		{Key: fiTokenHasListingSince, Value: t.HasListingSince},
		{Key: fiTokenAmountLastList, Value: price},
		{Key: fiTokenMinListPrice, Value: t.MinListPrice},
		{Key: fiTokenMinListValid, Value: t.MinListValid},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkAuctioned marks the given NFT as auctioned for the given price.
func (db *MongoDbBridge) TokenMarkAuctioned(contract *common.Address, tokenID *big.Int, reservePrice types.TokenPrice, ts *time.Time) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasAuctionSince, t.HasAuctionUntil = db.OpenAuctionRange(contract, tokenID)
	t.HasBids = false
	t.ReservePrice = reservePrice
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenLastAuction, Value: *ts},
		{Key: fiTokenHasAuctionSince, Value: t.HasAuctionSince},
		{Key: fiTokenHasAuctionUntil, Value: t.HasAuctionUntil},
		{Key: fiTokenHasBids, Value: t.HasBids},
		{Key: fiTokenReservePrice, Value: t.ReservePrice},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkBid marks the given NFT as having auction bid for the given price.
func (db *MongoDbBridge) TokenMarkBid(contract *common.Address, tokenID *big.Int, price types.TokenPrice, ts *time.Time) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasBids = true
	t.AmountLastBid = price
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenHasBids, Value: t.HasBids},
		{Key: fiTokenAmountLastBid, Value: t.AmountLastBid},
		{Key: fiTokenLastBid, Value: ts},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkUnlisted marks the given NFT as listed for direct sale for the given price.
func (db *MongoDbBridge) TokenMarkUnlisted(contract *common.Address, tokenID *big.Int) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasListingSince = db.OpenListingSince(contract, tokenID)
	t.MinListPrice, t.MinListValid = db.MinListingPrice(contract, tokenID)
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenHasListingSince, Value: t.HasListingSince},
		{Key: fiTokenMinListPrice, Value: t.MinListPrice},
		{Key: fiTokenMinListValid, Value: t.MinListValid},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkUnOffered marks the given NFT as not having buy offers.
func (db *MongoDbBridge) TokenMarkUnOffered(contract *common.Address, tokenID *big.Int) error {
	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenHasOfferUntil, Value: db.OpenOfferUntil(contract, tokenID)},
	})
}

// TokenMarkUnAuctioned marks the given NFT as not having an active auction on.
func (db *MongoDbBridge) TokenMarkUnAuctioned(contract *common.Address, tokenID *big.Int) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasAuctionSince, t.HasAuctionUntil = db.OpenAuctionRange(contract, tokenID)
	t.HasBids = false
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenHasAuctionSince, Value: t.HasAuctionSince},
		{Key: fiTokenHasAuctionUntil, Value: t.HasAuctionUntil},
		{Key: fiTokenHasBids, Value: t.HasBids},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkUnBid marks the given NFT as not having a bid anymore.
func (db *MongoDbBridge) TokenMarkUnBid(contract *common.Address, tokenID *big.Int) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasBids = false
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenHasBids, Value: false},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMarkSold marks the given NFT as sold for the given price.
func (db *MongoDbBridge) TokenMarkSold(contract *common.Address, tokenID *big.Int, price types.TokenPrice, ts *time.Time) error {
	t, err := db.GetToken(contract, tokenID)
	if t == nil {
		log.Errorf("unable to load token %s/%s; %s", contract.String(), (*hexutil.Big)(tokenID).String(), err)
		return err
	}

	t.HasAuctionSince, t.HasAuctionUntil = db.OpenAuctionRange(contract, tokenID)
	t.HasListingSince = db.OpenListingSince(contract, tokenID)
	t.MinListPrice, t.MinListValid = db.MinListingPrice(contract, tokenID)
	t.HasOfferUntil = db.OpenOfferUntil(contract, tokenID)
	t.AmountLastTrade = price
	t.HasBids = false
	t.AmountPrice, t.PriceValid = db.getTokenPrice(t)

	return db.UpdateToken(contract, tokenID, bson.D{
		{Key: fiTokenLastTrade, Value: *ts},
		{Key: fiTokenHasListingSince, Value: t.HasListingSince},
		{Key: fiTokenMinListPrice, Value: t.MinListPrice},
		{Key: fiTokenMinListValid, Value: t.MinListValid},
		{Key: fiTokenHasOfferUntil, Value: t.HasOfferUntil},
		{Key: fiTokenHasAuctionSince, Value: t.HasAuctionSince},
		{Key: fiTokenHasAuctionUntil, Value: t.HasAuctionUntil},
		{Key: fiTokenAmountLastTrade, Value: t.AmountLastTrade},
		{Key: fiTokenHasBids, Value: t.HasBids},
		{Key: fiTokenPrice, Value: t.AmountPrice},
		{Key: fiTokenPriceValid, Value: t.PriceValid},
	})
}

// TokenMetadataRefreshSet pulls s set of NFT tokens scheduled to be updated up to this time.
func (db *MongoDbBridge) TokenMetadataRefreshSet() ([]*types.Token, error) {
	list := make([]*types.Token, types.MetadataRefreshSetSize)
	col := db.client.Database(db.dbName).Collection(coTokens)

	// load the set from database
	cur, err := col.Find(
		context.Background(),
		bson.D{
			{Key: fiTokenMetadataUpdate, Value: bson.D{{"$lt", time.Now()}}},
			{Key: fiTokenMetadataURI, Value: bson.D{{"$ne", ""}}},
		},
		options.Find().SetSort(bson.D{{Key: fiTokenMetadataUpdate, Value: -1}}).SetLimit(types.MetadataRefreshSetSize),
	)
	if err != nil {
		log.Errorf("can not pull metadata refresh set; %s", err.Error())
		return nil, err
	}
	defer func() {
		if err := cur.Close(context.Background()); err != nil {
			log.Errorf("can not close cursor; %s", err.Error())
		}
	}()

	var i int
	for cur.Next(context.Background()) {
		var row types.Token
		if err := cur.Decode(&row); err != nil {
			log.Errorf("can not decode Token; %s", err.Error())
			return nil, err
		}
		list[i] = &row
		i++
	}
	return list[:i], nil
}

func (db *MongoDbBridge) getTokenPrice(t *types.Token) (tokenPrice int64, priceValidUntil *types.Time) {
	now := time.Now()

	// has auction - use auction top bid (or reserve price)
	if t.HasAuctionSince != nil && t.HasAuctionUntil != nil {
		if (*time.Time)(t.HasAuctionSince).Before(now) && (*time.Time)(t.HasAuctionUntil).After(now) {
			if t.HasBids {
				tokenPrice = t.AmountLastBid.Usd
			} else {
				tokenPrice = t.ReservePrice.Usd
			}
			priceValidUntil = t.HasAuctionUntil
		}
	}

	// has listing
	if t.MinListPrice.Usd != 0 {
		// the listing is cheaper then auction
		if tokenPrice == 0 || tokenPrice > t.MinListPrice.Usd {
			tokenPrice = t.MinListPrice.Usd

			// if validity from auction is not shorter, set validity by listings validity
			if t.MinListValid != nil && (priceValidUntil == nil || (*time.Time)(priceValidUntil).After(time.Time(*t.MinListValid))) {
				priceValidUntil = t.MinListValid
			}
		}
	}

	return
}

func (db *MongoDbBridge) ListTokens(filter *types.TokenFilter, sorting sorting.TokenSorting, sortDesc bool, cursor types.Cursor, count int, backward bool) (out *types.TokenList, err error) {
	var list types.TokenList
	col := db.client.Database(db.dbName).Collection(coTokens)
	ctx := context.Background()

	bsonFilter := tokenFilterToBson(filter)

	list.TotalCount, err = db.getTotalCount(col, bsonFilter)
	if err != nil {
		return nil, err
	}

	ld, err := db.findPaginated(col, bsonFilter, cursor, count, sorting, backward != sortDesc)
	if err != nil {
		log.Errorf("error loading tokens list; %s", err.Error())
		return nil, err
	}

	// close the cursor as we leave
	defer func() {
		err = ld.Close(ctx)
		if err != nil {
			log.Errorf("error closing tokens list cursor; %s", err.Error())
		}
	}()

	for ld.Next(ctx) {
		if len(list.Collection) < count {
			var row types.Token
			if err = ld.Decode(&row); err != nil {
				log.Errorf("can not decode the token in list; %s", err.Error())
				return nil, err
			}
			list.Collection = append(list.Collection, &row)
		} else {
			// skip the last row and set HasNext only
			list.HasNext = true
		}
	}

	if cursor != "" {
		list.HasPrev = true
	}
	if backward {
		list.Reverse()
	}
	return &list, nil
}

func tokenFilterToBson(f *types.TokenFilter) bson.D {
	filter := bson.D{}
	if f == nil {
		return filter
	}

	if f.Search != nil {
		filter = append(filter, bson.E{Key: fiTokenName, Value: primitive.Regex{
			Pattern: *f.Search,
			Options: "i",
		}})
	}

	now := types.Time(time.Now().UTC())

	if f.HasListing != nil {
		if *f.HasListing {
			filter = filterAddDateTimeLimit(filter, fiTokenHasListingSince, "$lte", now)
		} else {
			filter = filterAddDateTimeMiss(filter, fiTokenHasListingSince, "$gt", now)
		}
	}

	if f.HasAuction != nil {
		if *f.HasAuction {
			filter = filterAddDateTimeLimit(filter, fiTokenHasAuctionSince, "$lte", now)
			filter = filterAddDateTimeLimit(filter, fiTokenHasAuctionUntil, "$gt", now)
		} else {
			filter = filterAddDateTimeMiss(filter, fiTokenHasAuctionSince, "$gt", now)
			filter = filterAddDateTimeMiss(filter, fiTokenHasAuctionUntil, "$lte", now)
		}
	}

	if f.HasOffer != nil {
		if *f.HasOffer {
			filter = filterAddDateTimeLimit(filter, fiTokenHasOfferUntil, "$gt", now)
		} else {
			filter = filterAddDateTimeMiss(filter, fiTokenHasOfferUntil, "$lte", now)
		}
	}

	if f.HasBids != nil {
		filter = append(filter, bson.E{Key: fiTokenHasBids, Value: *f.HasBids})
		// filter for HasAuction time-range too, to exclude time-terminated auctions
		filter = filterAddDateTimeLimit(filter, fiTokenHasAuctionSince, "$lte", now)
		filter = filterAddDateTimeLimit(filter, fiTokenHasAuctionUntil, "$gt", now)
	}

	if f.Collections != nil && len(*f.Collections) > 0 {
		if len(*f.Collections) == 1 {
			filter = append(filter, bson.E{Key: fiTokenContract, Value: (*f.Collections)[0].String()})
		} else {
			values := make([]string, len(*f.Collections))
			for i, value := range *f.Collections {
				values[i] = value.String()
			}
			filter = append(filter, bson.E{Key: fiTokenContract, Value: bson.D{{Key: "$in", Value: values}}})
		}
	}

	if f.Categories != nil && len(*f.Categories) > 0 {
		if len(*f.Categories) == 1 {
			filter = append(filter, bson.E{Key: fiTokenCategories, Value: (*f.Categories)[0]})
		} else {
			filter = append(filter, bson.E{Key: fiTokenCategories, Value: bson.D{{Key: "$in", Value: *f.Categories}}})
		}
	}

	if f.CreatedBy != nil {
		filter = append(filter, bson.E{Key: fiTokenCreatedBy, Value: *f.CreatedBy})
	}

	return filter
}
