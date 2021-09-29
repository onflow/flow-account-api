package model

type Account struct {
	tableName             struct{}            `pg:"accounts"`
	Address               string              `json:"address" pg:"address,pk"`
	LockedAddress         string              `json:"lockedAddress" pg:"locked_address"`
	CreationTransactionID string              `json:"creationTxId" pg:"creation_tx_id"`
	PublicKeys            []*AccountPublicKey `json:"publicKeys" pg:"rel:has-many"`
}

type AccountPublicKey struct {
	tableName      struct{} `pg:"public_keys"`
	AccountAddress string   `json:"-" pg:"account_address"`
	PublicKey      string   `json:"publicKey" pg:"public_key,pk"`
	SigAlgo        string   `json:"signatureAlgorithm" pg:"sig_algo"`
	HashAlgo       string   `json:"hashAlgorithm" pg:"hash_algo"`
}
