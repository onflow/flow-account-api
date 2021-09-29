package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/templates"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-account-api/model"
)

const gasLimit = 100

type Accounts struct {
	flowClient                  *client.Client
	creatorAddress              flow.Address
	creatorKeyIndex             int
	creatorSigner               crypto.Signer
	accountLimit                int
}

func NewAccounts(
	accessAddress string,
	creatorAddress flow.Address,
	creatorKeyIndex int,
	creatorSigner crypto.Signer,
	accountLimit int,
) (*Accounts, error) {
	flowClient, err := client.New(accessAddress, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Accounts{
		flowClient:                  flowClient,
		creatorAddress:              creatorAddress,
		creatorKeyIndex:             creatorKeyIndex,
		creatorSigner:               creatorSigner,
		accountLimit:                accountLimit,
	}, nil
}

func (a *Accounts) Create(newAccountKey *flow.AccountKey) (*model.Account, error) {
	ctx := context.Background()

	accountCreatorKey, err := a.getAccountKey(ctx, a.creatorAddress, a.creatorKeyIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get account creator key: %w", err)
	}

	latestBlock, err := a.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block header: %w", err)
	}

	tx := a.createAccountTransaction(
		a.creatorAddress, 
		accountCreatorKey, 
		newAccountKey, 
		latestBlock.ID, 
	)

	err = tx.SignEnvelope(a.creatorAddress, accountCreatorKey.Index, a.creatorSigner)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = a.flowClient.SendTransaction(ctx, *tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	result, err := waitForSeal(ctx, a.flowClient, tx.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction result: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to execute transaction (id=%s): %w", tx.ID(), result.Error)
	}

	var address flow.Address

	for _, event := range result.Events {
		if event.Type == flow.EventAccountCreated {
			accountCreatedEvent := flow.AccountCreatedEvent(event)
			address = accountCreatedEvent.Address()
		}
	}

	publicKey := hex.EncodeToString(newAccountKey.PublicKey.Encode())

	return &model.Account{
		Address:               address.Hex(),
		LockedAddress:         "",
		CreationTransactionID: tx.ID().Hex(),
		PublicKeys: []*model.AccountPublicKey{
			{
				PublicKey: publicKey,
				SigAlgo:   newAccountKey.SigAlgo.String(),
				HashAlgo:  newAccountKey.HashAlgo.String(),
			},
		},
	}, nil
}

func (a *Accounts) GetLimit() int {
	return a.accountLimit
}

func (a *Accounts) getAccountKey(
	ctx context.Context, 
	address flow.Address, 
	index int,
) (*flow.AccountKey, error) {
	account, err := a.flowClient.GetAccountAtLatestBlock(ctx, address)
	if err != nil {
		return nil, err
	}

	if len(account.Keys) < index {
		return nil, fmt.Errorf("account with address %s does not contain key at index %d", address, index)
	}

	return account.Keys[index], nil
}

func (a *Accounts) createAccountTransaction(
	creatorAddress flow.Address,
	creatorAccountKey *flow.AccountKey,
	accountKey *flow.AccountKey,
	referenceBlockID flow.Identifier,
) *flow.Transaction {
	tx := templates.CreateAccount(
		[]*flow.AccountKey{accountKey},
		nil,
		a.creatorAddress,
	)

	return tx.
		SetReferenceBlockID(referenceBlockID).
		SetGasLimit(gasLimit).
		SetProposalKey(creatorAddress, creatorAccountKey.Index, creatorAccountKey.SequenceNumber).
		SetPayer(creatorAddress)
}

func waitForSeal(ctx context.Context, flowClient *client.Client, id flow.Identifier) (*flow.TransactionResult, error) {
	result, err := flowClient.GetTransactionResult(ctx, id)
	if err != nil {
		return nil, err
	}

	for result.Status != flow.TransactionStatusSealed {
		time.Sleep(time.Second)
		result, err = flowClient.GetTransactionResult(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

