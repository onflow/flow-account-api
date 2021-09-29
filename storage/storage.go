package storage

import (
	"errors"

	"github.com/onflow/flow-account-api/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

type Store interface {
	InsertAccount(account *model.Account) error
	GetAccountByPublicKey(publicKey string, account *model.Account) error
	GetAccountCount() (int, error)
}
