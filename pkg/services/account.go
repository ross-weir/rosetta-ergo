// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"context"
	"errors"

	"github.com/coinbase/rosetta-sdk-go/server"
	storageErrs "github.com/coinbase/rosetta-sdk-go/storage/errors"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/errutil"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
)

// AccountAPIService implements the server.AccountAPIServicer interface.
type AccountAPIService struct {
	cfg     *config.Configuration
	storage *storage.Storage
}

// NewAccountAPIService returns a new *AccountAPIService.
func NewAccountAPIService(
	cfg *config.Configuration,
	storage *storage.Storage,
) server.AccountAPIServicer {
	return &AccountAPIService{
		cfg:     cfg,
		storage: storage,
	}
}

// AccountBalance implements /account/balance.
func (s *AccountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if s.cfg.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	// TODO: filter balances by request currencies

	// If we are fetching a historical balance,
	// use balance storage and don't return coins.
	amount, block, err := s.getBalance(
		ctx,
		request.AccountIdentifier,
		s.cfg.Currency,
		request.BlockIdentifier,
	)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrUnableToGetBalance, err)
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: block,
		Balances: []*types.Amount{
			amount,
		},
	}, nil
}

// AccountCoins implements /account/coins.
func (s *AccountAPIService) AccountCoins(
	ctx context.Context,
	request *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	if s.cfg.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	// TODO: filter coins by request currencies

	// TODO: support include_mempool query
	// https://github.com/coinbase/rosetta-bitcoin/issues/36#issuecomment-724992022
	// Once mempoolcoins are supported also change the bool service/types.go:MempoolCoins to true

	coins, block, err := s.storage.Coin().GetCoins(ctx, request.AccountIdentifier)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrUnableToGetCoins, err)
	}

	result := &types.AccountCoinsResponse{
		BlockIdentifier: block,
		Coins:           coins,
	}

	return result, nil
}

func (s *AccountAPIService) getBalance(
	ctx context.Context,
	accountIdentifier *types.AccountIdentifier,
	currency *types.Currency,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.Amount, *types.BlockIdentifier, error) {
	dbTx := s.storage.Db().ReadTransaction(ctx)
	defer dbTx.Discard(ctx)

	blockResponse, err := s.storage.Block().GetBlockLazyTransactional(
		ctx,
		blockIdentifier,
		dbTx,
	)
	if err != nil {
		return nil, nil, err
	}

	amount, err := s.storage.Balance().GetBalanceTransactional(
		ctx,
		dbTx,
		accountIdentifier,
		currency,
		blockResponse.Block.BlockIdentifier.Index,
	)
	if errors.Is(err, storageErrs.ErrAccountMissing) {
		return &types.Amount{
			Value:    "0",
			Currency: currency,
		}, blockResponse.Block.BlockIdentifier, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return amount, blockResponse.Block.BlockIdentifier, nil
}
