package btc

import (
	"github.com/Sayplaydiv/btc_client/core/btc/internal"
	"github.com/dabankio/wallet-core/bip44"
)

// NewBip44Deriver btc bip44 实现
func NewBip44Deriver(bip44Path string, isSegWit bool, seed []byte, chainID int) (bip44.Deriver, error) {
	coin, err := internal.New(bip44Path, isSegWit, seed, chainID)
	return coin, err
}
