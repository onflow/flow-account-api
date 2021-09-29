package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-account-api/model"
	"github.com/onflow/flow-account-api/pkg/pg"
	"github.com/onflow/flow-account-api/storage"
	"github.com/onflow/flow-account-api/wallet"
)

type Store struct {
	conf        pg.Config
	environment string
	logger      zerolog.Logger
	db          *pg.Database
	done        chan bool
}

func NewStore(conf pg.Config, environment string, logger zerolog.Logger) (*Store, error) {
	return &Store{
		conf:        conf,
		environment: environment,
		logger:      logger,
		done:        make(chan bool, 1),
	}, nil
}

func (s *Store) Start() error {
	var err error
	s.db, err = pg.NewDatabase(s.conf)
	if err != nil {
		return err
	}

	err = s.migrate()
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	<-s.done

	return nil
}

func (s *Store) Stop() {
	_ = s.db.Close()
	s.done <- true
}

func (s *Store) migrate() (err error) {
	var v uint

	err = s.db.MigrateUp(context.Background(), &v)
	if err != nil {
		// https://github.com/mattes/migrate/issues/287
		if err.Error() != migrate.ErrNoChange.Error() {
			s.logger.
				Error().
				Err(err).
				Msg("error performing migration")

			return err
		} else if s.environment == wallet.EnvironmentTest {
			s.logger.
				Error().
				Err(err).
				Msg("expected to run migrations, but ran none, this is probably an error")

			return err
		}

		s.logger.Info().Msgf("already migrated to: %d", v)

		return nil
	}

	s.logger.Info().Msgf("Successfully migrated to version: %d", v)

	return nil
}

func (s Store) InsertAccount(account *model.Account) error {
	ctx := context.Background()

	err := s.db.RunInTransaction(ctx, func(ctx context.Context) error {
		_, err := s.db.Model(account).Insert()
		if err != nil {
			return err
		}

		for _, publicKey := range account.PublicKeys {
			// link public key to account
			publicKey.AccountAddress = account.Address

			_, err := s.db.Model(publicKey).Insert()
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, pg.ErrIntegrityViolation) {
			return storage.ErrExists
		}

		return err
	}

	return nil
}

func (s Store) GetAccountByPublicKey(publicKey string, account *model.Account) error {
	ctx := context.Background()

	err := s.db.RunInTransaction(ctx, func(ctx context.Context) error {
		return s.db.Model(account).
			Column("account.address").
			Relation("PublicKeys").
			Where("public_keys.public_key = ?", publicKey).
			Join("JOIN public_keys ON account.address = public_keys.account_address").
			Select()
	})

	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return storage.ErrNotFound
		}

		return err
	}

	return nil
}

func (s Store) GetAccountCount() (int, error) {
	return s.db.Model(&model.Account{}).Count()
}
