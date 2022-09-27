package main

import (
	"fmt"
	"github.com/Sayplaydiv/btc_client/core/btc"
	"github.com/blocktree/go-owcdrivers/addressEncoder"
	"github.com/blocktree/go-owcdrivers/btcLikeTxDriver"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/dabankio/wallet-core/bip39"
	"github.com/dabankio/wallet-core/bip44"
	"log"
)

func main() {
	createAddress()
	//signTransfer()
	//bech32Address()
}

func bech32Address() {

	BTCBech32Address := addressEncoder.BTC_testnetAddressBech32V0
	BTCAddress := addressEncoder.BTC_testnetAddressP2PKH

	account, _ := btcec.NewPrivateKey(btcec.S256())
	var pubkey = account.PubKey()
	witnessProg := btcutil.Hash160(pubkey.SerializeCompressed())

	address_bech32 := addressEncoder.AddressEncode(witnessProg, BTCBech32Address)
	log.Println("bech32:", address_bech32)

	addressWitnessPubKeyHash, err := btcutil.NewAddressWitnessPubKeyHash(witnessProg, &chaincfg.TestNet3Params)
	if err != nil {
		panic(err)
	}
	address := addressWitnessPubKeyHash.EncodeAddress()
	log.Println("bech32  0000:", address)

	address_nom := addressEncoder.AddressEncode(witnessProg, BTCAddress)
	log.Println("p2sh:", address_nom)

	address_de, _ := addressEncoder.AddressDecode(address_nom, BTCAddress)

	address_b1 := btcLikeTxDriver.Bech32Encode("tb", BTCBech32Address.Alphabet, address_de)
	log.Println("bech32 1111:", address_b1)
	deriver, err := btc.NewBip44Deriver(bip44.PathFormat, false, account.Serialize(), btc.ChainTestNet3)
	if err != nil {
		fmt.Println(err)
	}
	address_p2sh, _ := deriver.DeriveAddress()
	log.Println("p2sh 0000:", address_p2sh)

	//account,_:= btcec.NewPrivateKey(btcec.S256())
	//var pubkey = account.PubKey()
	//witnessProg := btcutil.Hash160(pubkey.SerializeCompressed())
	//addressWitnessPubKeyHash, err := btcutil.NewAddressWitnessPubKeyHash(witnessProg,  &chaincfg.TestNet3Params)
	//if err != nil {
	//	panic(err)
	//}
	//address := addressWitnessPubKeyHash.EncodeAddress()
	//log.Println(address)
	//log.Println(account.D.String())

}

func createAddress() {
	seedByte, _ := bip39.NewEntropy(256)
	deriver, err := btc.NewBip44Deriver(bip44.PathFormat, false, seedByte, btc.ChainTestNet3)
	if err != nil {
		fmt.Println(err)
	}
	address, _ := deriver.DeriveAddress()

	BTCBech32Address := addressEncoder.BTC_testnetAddressBech32V0
	BTCAddress := addressEncoder.BTC_testnetAddressP2PKH
	//解码
	address_de, _ := addressEncoder.AddressDecode(address, BTCAddress)

	//编码为bech32地址
	address_bech32 := btcLikeTxDriver.Bech32Encode("tb", BTCBech32Address.Alphabet, address_de)

	fmt.Println("p2sh address: ", address)
	log.Println("bech32 1111: ", address_bech32)

	fmt.Println(deriver.DerivePrivateKey())
	fmt.Println(deriver.DerivePublicKey())
}

func signTransfer() {
	input := new(btc.BTCUnspent)
	input.Add("f2437be18786d9e5cfb3617b3ea7d06c9e67a37b7069eb385181376ad1edb0ac", 0, 0.0001, "76a914f6f42a83334345d40f6e47a53ce7868c5f18cd2c88ac", "")

	addr, err := btc.NewBTCAddressFromString("mvtvSF4RgeVDVzVwUPY7RD4pFYLXBMmgih", btc.ChainTestNet3)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("addr", addr)
	amt, err := btc.NewBTCAmount(0.00001)
	// amt, err := btcutil.NewAmount(0.06)
	if err != nil {
		log.Fatal(err)
	}
	output := new(btc.BTCOutputAmount)
	output.Add(addr, amt)

	change, err := btc.NewBTCAddressFromString("tb1qwgg3mzd3f6gkmygd2cvvhnq8cxxmqv7fylpxwj", btc.ChainTestNet3)
	if err != nil {
		log.Fatal(err)
	}
	tt, err := btc.NewBTCTransaction(input, output, change, 2, btc.ChainTestNet3)
	if err != nil {
		log.Fatal(err)
	}
	ii, _ := tt.Encode()
	log.Println(ii)
	hh, err := tt.EncodeToSignCmd()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("hahah", hh)
	log.Println(tt.GetFee())
	bb, _ := btc.New(bip44.PathFormat, false, nil, btc.ChainTestNet3)
	cc, err := bb.Sign(hh, []string{"cNq4msygBZvVqVErf4Vk4FwyzT6hHcJPJCRAdsMJJ34xCNNavDy8"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(cc)

}
