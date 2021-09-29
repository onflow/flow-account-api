package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-account-api/model"
	"github.com/onflow/flow-account-api/storage"
)

// Service is a hardware wallet service.
type Service struct {
	httpServer *http.Server
	logger     zerolog.Logger
	accounts   *Accounts
	store      storage.Store
	metrics    *AccountsCollector
}

// NewService creates a new hardware wallet service.
func NewService(port int, logger zerolog.Logger, accounts *Accounts, store storage.Store, networkType string) *Service {
	s := &Service{
		logger:   logger,
		accounts: accounts,
		store:    store,
		metrics:  NewAccountsCollector(networkType),
	}

	router := mux.NewRouter()

	router.
		Handle("/metrics", promhttp.Handler())

	router.
		HandleFunc("/health", healthCheck)

	router.
		HandleFunc("/accounts", s.createAccount).
		Methods(http.MethodPost)

	router.
		HandleFunc("/accounts", s.getAccount).
		Methods(http.MethodGet)

	// TODO: allow CORS options to be configured via environment variable
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	return s
}

func (s *Service) Start() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (s *Service) Stop() {
	_ = s.httpServer.Shutdown(context.Background())
}

type createAccountRequest struct {
	PublicKey string `json:"publicKey"`
	SigAlgo   string `json:"signatureAlgorithm"`
	HashAlgo  string `json:"hashAlgorithm"`
}

func (s *Service) createAccount(w http.ResponseWriter, r *http.Request) {
	// Double check that we haven't exceeded our limit
	if s.exceededAccountLimit() {
		respondWithError(w, http.StatusForbidden, "service out of available accounts")
		return
	}

	var req createAccountRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	sigAlgo := crypto.StringToSignatureAlgorithm(req.SigAlgo)
	if sigAlgo == crypto.UnknownSignatureAlgorithm {
		respondWithError(w, http.StatusBadRequest, "invalid signature algorithm")
		return
	}

	hashAlgo := crypto.StringToHashAlgorithm(req.HashAlgo)
	if hashAlgo == crypto.UnknownHashAlgorithm {
		respondWithError(w, http.StatusBadRequest, "invalid hash algorithm")
		return
	}

	publicKey, err := crypto.DecodePublicKeyHex(sigAlgo, req.PublicKey)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusBadRequest, "invalid public key")
		return
	}

	accountKey := flow.NewAccountKey().
		SetPublicKey(publicKey).
		SetHashAlgo(hashAlgo).
		SetWeight(flow.AccountKeyWeightThreshold)

	account, err := s.accounts.Create(accountKey)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create account")

		respondWithError(
			w,
			http.StatusInternalServerError,
			"failed to create account",
		)
		return
	}

	err = s.store.InsertAccount(account)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			s.logger.Error().Err(err).Msg("account with address or public key already exists")

			respondWithError(
				w,
				http.StatusConflict,
				"account with address or public key already exists",
			)
			return
		}

		s.logger.Error().Err(err).Msg("failed to store account")

		respondWithError(
			w,
			http.StatusInternalServerError,
			"failed to create account",
		)
		return
	}

	respondWithJSON(w, http.StatusCreated, &account)
}

func (s *Service) getAccount(w http.ResponseWriter, r *http.Request) {
	publicKeys, ok := r.URL.Query()["publicKey"]
	if !ok {
		respondWithError(
			w,
			http.StatusBadRequest,
			"publicKey is required",
		)
		return
	}

	if len(publicKeys) != 1 {
		respondWithError(
			w,
			http.StatusBadRequest,
			"must provide one public key",
		)
		return
	}

	publicKey := publicKeys[0]

	var account model.Account

	err := s.store.GetAccountByPublicKey(publicKey, &account)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.logger.Error().Err(err).Msgf("account with public key %s does not exist", publicKey)

			respondWithError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("account with public key %s does not exist", publicKey),
			)
			return
		}

		s.logger.Error().Err(err).Msg("failed to get account by public key")

		respondWithError(
			w,
			http.StatusInternalServerError,
			"failed to get account by public key",
		)
		return
	}

	respondWithJSON(w, http.StatusOK, &account)
}

func (s *Service) exceededAccountLimit() bool {
	maxAccounts := s.accounts.GetLimit()
	if maxAccounts == 0 {
		// Assume zero means infinite accounts
		return false
	}

	numAccounts, err := s.store.GetAccountCount()
	if err != nil {
		s.logger.Err(err).Msg("could not count number of accounts created by service")
		// If we encounter an error, do not allow users to create any more accounts
		return true
	}
	s.metrics.CurrentNumberOfAccounts(numAccounts)

	return numAccounts >= maxAccounts
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
