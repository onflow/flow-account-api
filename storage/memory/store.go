package memory

import (
	"sync"

	"github.com/onflow/flow-account-api/model"
	"github.com/onflow/flow-account-api/storage"
)

type Store struct {
	mut                 sync.RWMutex
	accounts            map[string]model.Account
	publicKeysToAddress map[string]string
}

func NewStore() *Store {
	return &Store{
		accounts:            make(map[string]model.Account),
		publicKeysToAddress: make(map[string]string),
	}
}

func (s *Store) InsertAccount(account *model.Account) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	_, ok := s.accounts[account.Address]
	if ok {
		return storage.ErrExists
	}

	s.accounts[account.Address] = *account

	for _, publicKey := range account.PublicKeys {
		_, ok := s.publicKeysToAddress[publicKey.PublicKey]
		if ok {
			return storage.ErrExists
		}

		s.publicKeysToAddress[publicKey.PublicKey] = account.Address
	}

	return nil
}

func (s *Store) GetAccountByPublicKey(publicKey string, account *model.Account) error {
	s.mut.RLock()
	defer s.mut.RUnlock()

	address, ok := s.publicKeysToAddress[publicKey]
	if !ok {
		return storage.ErrNotFound
	}

	a, ok := s.accounts[address]
	if !ok {
		return storage.ErrNotFound
	}

	*account = a

	return nil
}

func (s *Store) GetAccountCount() (int, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	return len(s.accounts), nil
}
