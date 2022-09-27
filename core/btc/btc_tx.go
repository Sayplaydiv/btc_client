package btc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/Sayplaydiv/btc_client/core/btc/internal"
	"github.com/Sayplaydiv/btc_client/core/btc/internal/helpers"
	"github.com/Sayplaydiv/btc_client/core/btc/internal/txauthor"
	"sync"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/pkg/errors"
)

// BTCTransaction represents a single bitcoin transaction.
type BTCTransaction struct {
	chainCfg        *chaincfg.Params
	tx              *wire.MsgTx
	totalInputValue *btcutil.Amount
	rawTxInput      *[]internal.RawTxInput //RawTxInput
}

// BTCUnspent represents a single bitcoin transaction.
type BTCUnspent struct {
	unspent []btcjson.ListUnspentResult
}

// Add add one utxo
func (us *BTCUnspent) Add(txId string, vOut int64, amount float64, scriptPubKey, redeemScript string) {
	us.unspent = append(us.unspent, btcjson.ListUnspentResult{
		TxID:         txId,
		Vout:         uint32(vOut),
		ScriptPubKey: scriptPubKey,
		RedeemScript: redeemScript,
		Amount:       amount,
	})
}

// BTCOutputAmount 交易输出
type BTCOutputAmount struct {
	addressValue map[BTCAddress]BTCAmount
	mutx         sync.Mutex
}

// Add 添加一个交易输出
// address 地址
// amount 金额
func (baa *BTCOutputAmount) Add(address *BTCAddress, amount *BTCAmount) {
	baa.mutx.Lock()
	defer baa.mutx.Unlock()
	if baa.addressValue == nil {
		baa.addressValue = make(map[BTCAddress]BTCAmount)
	}
	baa.addressValue[*address] = *amount
}

// NewBTCTransaction creates a new bitcoin transaction with the given properties.
// unSpent : listUnspent
// amounts: toAddress + amount
// change: 找零地址
// feeRate: 单位手续费/byte
// testNet: 测试网络传true
func NewBTCTransaction(unSpent *BTCUnspent, amounts *BTCOutputAmount, change *BTCAddress, feeRate int64, chainID int) (tr *BTCTransaction, err error) {
	return InternalNewBTCTransaction(unSpent, amounts, change, feeRate, chainID, nil)
}

// InternalNewBTCTransaction 内部用，构造btc transaction
func InternalNewBTCTransaction(unSpent *BTCUnspent, amounts *BTCOutputAmount, change *BTCAddress, feeRate int64, chainID int, manualTxOuts []*wire.TxOut) (tr *BTCTransaction, err error) {
	if unSpent == nil || amounts == nil || change == nil || feeRate == 0 {
		err = errors.New("maybe some parameter is missing?")
		return
	}

	tr = &BTCTransaction{
		rawTxInput: &[]internal.RawTxInput{},
	}
	tr.chainCfg, err = internal.ChainFlag2ChainParams(chainID)
	if err != nil {
		return nil, err
	}

	// 转换 to amount
	var txOut []*wire.TxOut
	for addr, amt := range amounts.addressValue {
		if !addr.address.IsForNet(tr.chainCfg) {
			err = errors.Errorf("%s is not the corresponding network address", addr.address)
		}

		// Create a new script which pays to the provided address.
		pkScript, err := txscript.PayToAddrScript(addr.address)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate pay-to-address script")
		}
		txOut = append(txOut, &wire.TxOut{
			Value:    int64(amt.amount),
			PkScript: pkScript,
		})
	}

	for _, manualTxOut := range manualTxOuts {
		txOut = append(txOut, manualTxOut)
	}

	relayFeePerKb := btcutil.Amount(feeRate * 1000)
	txIn := tr.makeInputSource(unSpent.unspent)
	if !change.address.IsForNet(tr.chainCfg) {
		err = errors.Errorf("%s is not the corresponding network address", change.address)
	}
	changeSource := tr.makeDestinationScriptSource(change.address.String())

	unsignedTransaction, err := txauthor.NewUnsignedTransaction(txOut, relayFeePerKb, txIn, changeSource)
	if err != nil {
		return
	}
	getScript := func(txId string) (scriptPubKey, redeemScript string, amount float64) {
		for i := range unSpent.unspent {
			if unSpent.unspent[i].TxID == txId {
				return unSpent.unspent[i].ScriptPubKey, unSpent.unspent[i].RedeemScript, unSpent.unspent[i].Amount
			}
		}
		return
	}
	for i := range unsignedTransaction.Tx.TxIn {
		txId := unsignedTransaction.Tx.TxIn[i].PreviousOutPoint.Hash.String()
		scriptPubKey, redeemScript, amt := getScript(txId)
		*tr.rawTxInput = append(*tr.rawTxInput, internal.RawTxInput{
			Txid:         unsignedTransaction.Tx.TxIn[i].PreviousOutPoint.Hash.String(),
			Vout:         unsignedTransaction.Tx.TxIn[i].PreviousOutPoint.Index,
			ScriptPubKey: scriptPubKey,
			RedeemScript: redeemScript,
			Amount:       amt,
		})
	}
	tr.totalInputValue = &unsignedTransaction.TotalInput
	tr.tx = unsignedTransaction.Tx
	return
}

// GetFee 获取目前的费率(in BTC, not satoshi)
// Returns the miner's fee for the current transaction
func (tx BTCTransaction) GetFee() (float64, error) {
	if tx.totalInputValue == nil {
		return 0., errors.New("transaction data not filled")
	}
	fee := *tx.totalInputValue - helpers.SumOutputValues(tx.tx.TxOut)
	return fee.ToBTC(), nil
}

// Encode encode to raw transaction
func (tx BTCTransaction) Encode() (string, error) {
	var buf bytes.Buffer
	if tx.tx == nil {
		return "", errors.New("transaction data not filled")
	}
	if err := tx.tx.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding); err != nil {
		return "", errors.Wrapf(err, "failed to encode msg of type %T", tx.tx)
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

// EncodeToSignCmd 结果可以用于签名接口
func (tx BTCTransaction) EncodeToSignCmd() (string, error) {
	data, err := tx.Encode()
	if err != nil {
		return "", err
	}

	cmd := &internal.SignRawTransactionCmd{
		RawTx:    data,
		Inputs:   tx.rawTxInput,
		PrivKeys: nil,
		Flags:    nil,
	}
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(cmdBytes), nil
}

// EncodeToSignCmdForNextSigner 构造给下个签名者签名的命令，
// signedRawTX: 当前签名者已签名好的交易数据
func (tx BTCTransaction) EncodeToSignCmdForNextSigner(signedRawTX string) (string, error) {
	cmd := &internal.SignRawTransactionCmd{
		RawTx:    signedRawTX,
		Inputs:   tx.rawTxInput,
		PrivKeys: nil,
		Flags:    nil,
	}
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(cmdBytes), nil
}

// makeInputSource creates an InputSource that creates inputs for every unspent
// output with non-zero output values.  The target amount is ignored since every
// output is consumed.  The InputSource does not return any previous output
// scripts as they are not needed for creating the unsinged transaction and are
// looked up again by the wallet during the call to signrawtransaction.
func (tx BTCTransaction) makeInputSource(unspentResults []btcjson.ListUnspentResult) txauthor.InputSource {
	// Return outputs in order.
	currentTotal := btcutil.Amount(0)
	currentInputs := make([]*wire.TxIn, 0, len(unspentResults))
	currentInputValues := make([]btcutil.Amount, 0, len(unspentResults))
	f := func(target btcutil.Amount) (btcutil.Amount, []*wire.TxIn, []btcutil.Amount, [][]byte, error) {
		for currentTotal < target && len(unspentResults) != 0 {
			u := unspentResults[0]
			unspentResults = unspentResults[1:]
			hash, _ := chainhash.NewHashFromStr(u.TxID)
			nextInput := wire.NewTxIn(&wire.OutPoint{
				Hash:  *hash,
				Index: u.Vout,
			}, nil, nil)
			amount, _ := NewBTCAmount(u.Amount)
			currentTotal += amount.amount
			currentInputs = append(currentInputs, nextInput)
			currentInputValues = append(currentInputValues, amount.amount)
		}
		return currentTotal, currentInputs, currentInputValues, make([][]byte, len(currentInputs)), nil
	}
	return txauthor.InputSource(f)
}

// makeDestinationScriptSource creates a ChangeSource which is used to receive
// all correlated previous input value.  A non-change address is created by this
// function.
func (tx BTCTransaction) makeDestinationScriptSource(destinationAddress string) txauthor.ChangeSource {
	return func() ([]byte, error) {
		addr, err := btcutil.DecodeAddress(destinationAddress, tx.chainCfg)
		if err != nil {
			return nil, err
		}
		return txscript.PayToAddrScript(addr)
	}
}
