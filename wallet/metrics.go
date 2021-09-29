package wallet

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const metricsNamespace = "flow"

type AccountsCollector struct {
	accounts prometheus.Gauge
}

func NewAccountsCollector(networkType string) *AccountsCollector {

	ac := &AccountsCollector{

		accounts: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "hardware_wallet_accounts_total",
			Namespace: metricsNamespace,
			Subsystem: networkType,
			Help:      "the number of accounts created by the service",
		}),
	}

	return ac
}

// CurrentNumberOfAccounts records the current number of accounts created by the service.
func (ac *AccountsCollector) CurrentNumberOfAccounts(accounts int) {
	ac.accounts.Set(float64(accounts))
}
