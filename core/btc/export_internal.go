package btc

import (
	"btcsdk/walletsdk/btc_client/core"
	"btcsdk/walletsdk/btc_client/core/btc/internal"
)

// Btc (全部大写在导出到java那边有点问题)
type Btc struct {
	internal.BTC
}

// NewCoin btc impl of core.Coin
func NewCoin(bip44Path string, isSegWit bool, seed []byte, chainID int) (*internal.BTC, error) {
	coin, err := internal.New(bip44Path, isSegWit, seed, chainID)
	return coin, err
}

// NewFromMetadata .
func NewFromMetadata(metadata core.MetadataProvider) (c *internal.BTC, err error) {
	return internal.NewFromMetadata(metadata)
}

var New = internal.New

// FlagUseSegWitFormat BTC使用隔离见证地址
const FlagUseSegWitFormat = internal.FlagUseSegWitFormat
