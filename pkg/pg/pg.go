package pg

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/golang-migrate/migrate"
	"golang.org/x/xerrors"
)

// Database is a connection to a Postgres database.
type Database struct {
	*pg.DB
	version          uint
	migrationPath    string
	connectionString string
}

// NewDatabase creates a new database connection.
func NewDatabase(conf Config) (*Database, error) {
	ops, err := parseURL(conf.ConnectionString)
	if err != nil {
		return nil, err
	}

	// useful when debugging long running queries
	ops.ApplicationName = conf.PGApplicationName

	db, err := connectPG(&conf.ConnectPGOptions, ops)
	if err != nil {
		return nil, err
	}

	// set query logger
	if conf.SetInternalPGLogger {
		pg.SetLogger(&logger{log.New(os.Stdout, conf.PGLoggerPrefix, log.LstdFlags)})
	}

	provider := &Database{
		DB:               db,
		migrationPath:    conf.MigrationPath,
		connectionString: conf.ConnectionString,
	}

	return provider, nil
}

// Ping pings the database to ensure that we can connect to it.
func (d *Database) Ping(ctx context.Context) (err error) {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		i := 0
		_, err = tx.QueryOne(pg.Scan(&i), "SELECT 1")
		return err
	})
}

// Close closes the database.
func (d *Database) Close() error {
	return d.DB.Close()
}

// GetMigrationVersion returns what migration we are on.
func (d *Database) GetMigrationVersion(ctx context.Context, v *uint) error {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if v == nil {
			return errors.New("pg: migration version (v) is nil")
		}
		*v = d.version
		return nil
	})
}

// MigrateUp performs an up migration.
func (d *Database) MigrateUp(ctx context.Context, version *uint) error {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if version == nil {
			return errors.New("pg: migration version (v) is nil")
		}

		migrator, err := migrate.New("file://"+d.migrationPath, d.connectionString)
		if err != nil {
			return err
		}

		defer migrator.Close()

		// get the version first
		// ignore dirty, as Up will fail if dirty
		v, _, err := migrator.Version()

		if err != nil {
			if err.Error() != migrate.ErrNilVersion.Error() {
				return err
			}
		} else {
			// set the current migration version
			d.version = v
			*version = v
		}

		// Migrate all the way up ...
		if err := migrator.Up(); err != nil {
			return err
		}

		// get new version
		v, _, err = migrator.Version()

		// should be version now
		if err != nil {
			return err
		}

		// set the current migration version
		d.version = v
		*version = v

		// should be idempotent
		srcErr, dbErr := migrator.Close()
		if srcErr != nil {
			return srcErr
		} else if dbErr != nil {
			return dbErr
		}

		return nil
	})
}

// MigrateDown performs a down migration.
func (d *Database) MigrateDown(ctx context.Context) error {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		m, err := migrate.New("file://"+d.migrationPath, d.connectionString)
		if err != nil {
			return err
		}
		// cleanup after
		defer m.Close()

		// Migrate all the way down ...
		return m.Down()
	})
}

// TruncateAll truncates all tables other that schema_migrations.
func (d *Database) TruncateAll() error {
	// query the DB for a list of all our tables
	var tableInfo []struct {
		Table string
	}

	query := `
		SELECT table_name "table"
		FROM information_schema.tables WHERE table_schema='public'
		AND table_type='BASE TABLE' AND table_name!= 'schema_migrations';
	`

	if _, err := d.DB.Query(&tableInfo, query); err != nil {
		return err
	}

	// run a transaction that truncates all our tables
	return d.DB.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		for _, info := range tableInfo {
			if _, err := tx.Exec("TRUNCATE " + info.Table + " CASCADE;"); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *Database) RunInTransaction(ctx context.Context, next func(ctx context.Context) error) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	return tx.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return convertError(next(ctx))
	})
}

// ToURL constructs a Postgres querystring with sensible defaults.
func ToURL(port int, ssl bool, username, password, db, host string) string {
	str := "postgres://"

	if len(username) == 0 {
		username = "postgres"
	}
	str += url.PathEscape(username)

	if len(password) > 0 {
		str = str + ":" + url.PathEscape(password)
	}

	if port == 0 {
		port = 5432
	}

	if db == "" {
		db = "postgres"
	}

	if host == "" {
		host = "localhost"
	}

	mode := ""
	if !ssl {
		mode = "?sslmode=disable"
	}

	return str + "@" +
		host + ":" +
		strconv.Itoa(port) + "/" +
		url.PathEscape(db) + mode
}

// parseURL is a wrapper around `pg.ParseURL`
// that undoes the logic in https://github.com/go-pg/pg/pull/458; which is
// to ensure that InsecureSkipVerify is false.
func parseURL(sURL string) (*pg.Options, error) {
	options, err := pg.ParseURL(sURL)
	if err != nil {
		return nil, xerrors.Errorf("pg: %w", err)
	}

	if options.TLSConfig != nil {
		// override https://github.com/go-pg/pg/pull/458
		options.TLSConfig.InsecureSkipVerify = false
		// TLSConfig now requires either InsecureSkipVerify = true or ServerName not empty
		options.TLSConfig.ServerName = strings.Split(options.Addr, ":")[0]
	}

	return options, nil
}

type logger struct {
	log *log.Logger
}

func (l *logger) Printf(ctx context.Context, format string, v ...interface{}) {
	l.log.Printf(format, v...)
}
