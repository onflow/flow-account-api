package main

import (
	"os"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/psiemens/graceland"
	"github.com/psiemens/sconfig"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-account-api/pkg/pg"
	"github.com/onflow/flow-account-api/storage/postgres"
	"github.com/onflow/flow-account-api/wallet"
)

type Config struct {
	AppName     string `default:"flow-account-api"`
	Environment string `required:"true"`
	Port        int    `default:"8080"`

	CreatorAddress     string `required:"true"`
	CreatorPrivateKey  string
	CreatorKeyIndex    int    `default:"0"`
	CreatorKeySigAlgo  string `required:"true"`
	CreatorKeyHashAlgo string `required:"true"`

	NetworkType   string `default:"emulator"`
	AccessAPIHost string

	AccountLimit                       int  `default:"0"` // Zero is assumed to mean no limit

	PostgreSQLHost              string        `default:"localhost"`
	PostgreSQLPort              uint16        `default:"5432"`
	PostgreSQLUsername          string        `default:"postgres"`
	PostgreSQLPassword          string        `required:"false"`
	PostgreSQLDatabase          string        `required:"true"`
	PostgreSQLSSL               bool          `default:"true"`
	PostgreSQLLogQueries        bool          `default:"false"`
	PostgreSQLSetLogger         bool          `default:"false"`
	PostgreSQLRetryNumTimes     uint16        `default:"30"`
	PostgreSQLRetrySleepTime    time.Duration `default:"1s"`
	PostgreSQLMigrationPath     string        `default:"/data/migrations"`
	PostgreSQLPoolSize          int           `required:"true"`
	PostgresLoggerPrefix        string        `required:"true"`
	PostgresPrometheusSubSystem string        `required:"true"`
}

var conf Config

const envPrefix = "FLOW"

func main() {
	err := sconfig.New(&conf).
		FromEnvironment(envPrefix).
		Parse()
	if err != nil {
		panic(err)
	}

	creatorAddress := flow.HexToAddress(conf.CreatorAddress)

	creatorKeySigAlgo := crypto.StringToSignatureAlgorithm(conf.CreatorKeySigAlgo)
	creatorKeyHashAlgo := crypto.StringToHashAlgorithm(conf.CreatorKeyHashAlgo)

	creatorPrivateKey, err := crypto.DecodePrivateKeyHex(creatorKeySigAlgo, conf.CreatorPrivateKey)
	if err != nil {
		panic(err)
	}

	creatorSigner := crypto.NewInMemorySigner(creatorPrivateKey, creatorKeyHashAlgo)

	accounts, err := wallet.NewAccounts(
		conf.AccessAPIHost,
		creatorAddress,
		conf.CreatorKeyIndex,
		creatorSigner,
		conf.AccountLimit,
	)
	if err != nil {
		panic(err)
	}

	logger := zerolog.New(os.Stderr)

	// store := memory.NewStore()

	store, err := postgres.NewStore(getPostgresConfig(conf, logger), conf.Environment, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initial Postgres database")
	}

	service := wallet.NewService(conf.Port, logger, accounts, store, conf.NetworkType)

	group := graceland.NewGroup()

	group.Add(service)
	group.Add(store)

	err = group.Start()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to run server")
	}
}

func getPostgresConfig(conf Config, logger zerolog.Logger) pg.Config {
	return pg.Config{
		ConnectPGOptions: pg.ConnectPGOptions{
			ConnectionString: getPostgresConnectionString(conf),
			RetrySleepTime:   conf.PostgreSQLRetrySleepTime,
			RetryNumTimes:    conf.PostgreSQLRetryNumTimes,
			TLSConfig:        nil,
			ConnErrorLogger: func(
				numTries int,
				duration time.Duration,
				host string,
				db string,
				user string,
				ssl bool,
				err error,
			) {
				// warn is probably a little strong here
				logger.Info().
					Int("numTries", numTries).
					Dur("duration", duration).
					Str("host", host).
					Str("db", db).
					Str("user", user).
					Bool("ssl", ssl).
					Err(err).
					Msg("connection failed")
			},
		},
		SetInternalPGLogger: conf.Environment != wallet.EnvironmentTest, // docker-compose will die from spam
		PGApplicationName:   conf.AppName,
		PGLoggerPrefix:      conf.PostgresLoggerPrefix,
		MigrationPath:       conf.PostgreSQLMigrationPath,
		PGPoolSize:          conf.PostgreSQLPoolSize,
	}
}

func getPostgresConnectionString(conf Config) string {
	return pg.ToURL(
		int(conf.PostgreSQLPort),
		conf.PostgreSQLSSL,
		conf.PostgreSQLUsername,
		conf.PostgreSQLPassword,
		conf.PostgreSQLDatabase,
		conf.PostgreSQLHost,
	)
}
