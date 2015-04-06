// Copyright 2015 Simon Liu.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package main runs some sample code
package main

import (
	coinspark "github.com/bitcartel/go-coinspark/coinspark"
	//coinspark "../coinspark"
	//"bufio"
	//"bytes"
	//"encoding/hex"
	"fmt"
	//"bytes"
	"crypto/rand"
	//"os"
	//"strconv"
	//"strings"
	//"math/rand"
)

func CreateCoinSparkAddress() {
	fmt.Println("\nCreating a CoinSpark address...\n")

	address := coinspark.CoinSparkAddress{}
	address.BitcoinAddress = "149wHUMa41Xm2jnZtqgRx94uGbZD9kPXnS"
	address.AddressFlags = coinspark.COINSPARK_ADDRESS_FLAG_ASSETS | coinspark.COINSPARK_ADDRESS_FLAG_PAYMENT_REFS
	address.PaymentRef = coinspark.CoinSparkPaymentRef{0} // or any unsigned 52-bit integer up to coinspark.COINSPARK_PAYMENT_REF_MAX

	addressString := address.Encode()

	if addressString != "" {
		fmt.Println("CoinSpark address: ", addressString)
	} else {
		fmt.Println("CoinSpark address encode failed!")
	}
}

func DecodeCoinSparkAddress() {
	fmt.Println("\nDecoding a CoinSpark address...\n")

	address := coinspark.CoinSparkAddress{}

	if address.Decode("s6GUHy69HWkwFqzFhJCY49seL8EFv") {
		fmt.Println("Bitcoin address: ", address.BitcoinAddress)
		fmt.Println("Address flags: ", address.AddressFlags)
		fmt.Println("Payment reference: ", address.PaymentRef.Ref)
		fmt.Printf(address.String())
	} else {
		fmt.Println("CoinSpark address decode failed!")
	}
}

func ProcessTransactionRawBinary(scriptPubKeys [][]byte, countInputs int) {
	fmt.Println("\nExtracting CoinSpark metadata from a transaction...\n")

	// scriptPubKeys is an array containing each output script of a transaction as raw binary data.
	// The transaction has scriptPubKeys.length outputs and countInputs inputs.

	numScriptPubKeys := len(scriptPubKeys)
	scriptPubKeysStringArray := make([]string, numScriptPubKeys)
	for i := 0; i < numScriptPubKeys; i++ {
		scriptPubKeysStringArray[i] = string(scriptPubKeys[i])
	}

	metadata := coinspark.ScriptsToMetadata(scriptPubKeysStringArray, false)

	if metadata != nil {
		genesis := coinspark.CoinSparkGenesis{}
		if genesis.Decode(metadata) {
			fmt.Printf(genesis.String())
		}

		transferList := coinspark.CoinSparkTransferList{}
		if transferList.Decode(metadata, countInputs, len(scriptPubKeys)) > 0 {
			fmt.Printf(transferList.String())
		}

		paymentRef := coinspark.CoinSparkPaymentRef{}
		if paymentRef.Decode(metadata) {
			fmt.Printf(paymentRef.String())
		}

		message := coinspark.CoinSparkMessage{}
		if message.Decode(metadata, len(scriptPubKeys)) {
			fmt.Printf(message.String())
		}
	}

}

func ProcessTransaction(scriptPubKeys []string, countInputs int) {
	fmt.Println("\nExtracting CoinSpark metadata from a transaction...\n")

	// scriptPubKeys is an array containing each output script of a transaction as a hex string
	// or raw binary (commented above). The transaction has scriptPubKeys.length outputs and
	// countInputs inputs.

	metadata := coinspark.ScriptsToMetadata(scriptPubKeys, true)

	if metadata != nil {
		genesis := coinspark.CoinSparkGenesis{}
		if genesis.Decode(metadata) {
			fmt.Printf(genesis.String())
		}

		transferList := coinspark.CoinSparkTransferList{}
		if transferList.Decode(metadata, countInputs, len(scriptPubKeys)) > 0 {
			fmt.Printf(transferList.String())
		}

		paymentRef := coinspark.CoinSparkPaymentRef{}
		if paymentRef.Decode(metadata) {
			fmt.Printf(paymentRef.String())
		}

		message := coinspark.CoinSparkMessage{}
		if message.Decode(metadata, len(scriptPubKeys)) {
			fmt.Printf(message.String())
		}
	}
}

func EncodeMetaData(metadata []byte) []byte {

	fmt.Println("\nEncoding CoinSpark metadata in a script...\n")

	// first get metadata from the encode() method of a CoinSparkGenesis, CoinSparkTransferList
	// or CoinSparkPaymentRef object, or the CoinSparkBase.metadataAppend() method.

	var scriptPubKey string

	if metadata != nil {
		scriptPubKey = coinspark.MetadataToScript(metadata, false)

		if scriptPubKey != "" {
			// now embed the raw bytes in $scriptPubKey directly in a transaction output
		} else {
			// handle the error
		}
	} else {
		// handle the error
	}

	return []byte(scriptPubKey)
}

func EncodeMetaDataToHex(metadata []byte) string {
	fmt.Println("\nEncoding CoinSpark metadata in a script...\n")

	// first get metadata from the encode() method of a CoinSparkGenesis, CoinSparkTransferList
	// or CoinSparkPaymentRef object, or the CoinSparkBase.metadataAppend() method.

	var scriptPubKey string

	if metadata != nil {
		scriptPubKey = coinspark.MetadataToScript(metadata, true)

		if scriptPubKey != "" {
			fmt.Println("Script: ", scriptPubKey)
		} else {
			fmt.Println("Metadata encode failed!")
		}
	} else {
		// handle the error
	}

	return scriptPubKey
}

func CreateGenesis() coinspark.CoinSparkGenesis {
	fmt.Println("\nCreating and encoding genesis metadata...\n")

	genesis := coinspark.CoinSparkGenesis{}

	genesis.SetQty(1234567, 1) // 1234567 units rounded up
	//actualQty:=genesis.GetQty() // can check final quantity assigned

	genesis.SetChargeFlat(4321, 0) // 4321 units rounded to nearest
	//actualChargeFlat:=genesis.GetChargeFlat(); // can check final flat charge assigned

	genesis.ChargeBasisPoints = 10 // additional 0.1% per payment

	genesis.UseHttps = false
	genesis.DomainName = "www.example.com"
	genesis.UsePrefix = true
	genesis.PagePath = "usd-1"

	assetHashLen := genesis.CalcHashLen(40) // 40 byte limit for OP_RETURN
	genesis.AssetHashLen = assetHashLen

	assetHash := make([]byte, assetHashLen)
	rand.Read(assetHash) // random hash in example
	genesis.AssetHash = assetHash

	err, metadata := genesis.Encode(40) // 40 byte limit for OP_RETURNs

	if metadata != nil {
		// use coinspark.MetadataToScript() to embed metadata in an output script
	} else if err != nil {
		// handle error
	}

	return genesis
}

func CreateMessage() coinspark.CoinSparkMessage {
	fmt.Println("\nCreating and encoding message metadata...\n")

	message := coinspark.CoinSparkMessage{}

	message.UseHttps = true
	message.ServerHost = "123.45.67.89"
	message.UsePrefix = false
	message.ServerPath = "msg"
	message.IsPublic = false
	message.OutputRanges = []coinspark.CoinSparkIORange{coinspark.CoinSparkIORange{0, 2}} // message is for outputs 0 and 1

	countOutputs := 3                                // 3 outputs for this transaction
	hashLen := message.CalcHashLen(countOutputs, 40) // 40 byte limit for OP_RETURN
	message.HashLen = hashLen

	hash := make([]byte, hashLen)
	rand.Read(hash) // random hash in example
	message.Hash = hash

	metadata := message.Encode(countOutputs, 40) // 40 byte limit for OP_RETURNs

	if metadata != nil {
		// use coinspark.MetadataToScript() to embed metadata in an output script
	} else {
		// handle error
	}

	return message
}

func CreateTransferList() coinspark.CoinSparkTransferList {
	fmt.Println("\nCreating and encoding transfer metadata...\n")

	countInputs := 3
	countOutputs := 5
	transferList := coinspark.CoinSparkTransferList{}
	transfers := []coinspark.CoinSparkTransfer{}

	transfer := coinspark.CoinSparkTransfer{}
	assetRef := coinspark.CoinSparkAssetRef{}
	assetRef.Decode("456789-65432-23456")
	transfer.AssetRef = assetRef
	transfer.Inputs = coinspark.CoinSparkIORange{0, 2}  // transfer from inputs 0 and 1
	transfer.Outputs = coinspark.CoinSparkIORange{0, 1} // transfer to outputs 0 only
	transfer.QtyPerOutput = 123
	transfers = append(transfers, transfer)

	transfer = coinspark.CoinSparkTransfer{}
	transfer.AssetRef = transfers[0].AssetRef
	transfer.Inputs = coinspark.CoinSparkIORange{2, 1}  // transfer from input 2 only
	transfer.Outputs = coinspark.CoinSparkIORange{1, 3} // transfer to outputs 1, 2 and 3
	transfer.QtyPerOutput = 456
	transfers = append(transfers, transfer)
	transferList.Transfers = transfers

	metadata := transferList.Encode(countInputs, countOutputs, 40) // 40 byte limit for OP_RETURNs

	if metadata != nil {
		// use coinspark.MetadataToScript() to embed metadata in an output script
	} else {
		// handle error
	}

	return transferList
}

func CreatePaymentRef() coinspark.CoinSparkPaymentRef {
	fmt.Println("\nCreating and encoding payment reference metadata...\n")

	paymentRef := coinspark.CoinSparkPaymentRef{}
	paymentRef.Randomize()            // randomizes the payment reference
	metadata := paymentRef.Encode(40) // assume 40 byte limit for OP_RETURNs

	if metadata != nil {
		// use coinspark.MetadataToScript() to embed metadata in an output script
	} else {
		// handle error
	}
	return paymentRef
}

func CoinSparkPaymentRefTransfersEncode(paymentRef coinspark.CoinSparkPaymentRef, transferList coinspark.CoinSparkTransferList, countInputs int, countOutputs int, metadataMaxLen int) []byte {

	metadata := paymentRef.Encode(metadataMaxLen)
	if metadata == nil {
		return nil
	}

	appendMetadataMaxLen := coinspark.MetadataMaxAppendLen(metadata, metadataMaxLen)
	// this is not simply metadataMaxLen-metadata.length since combining saves space

	appendMetaData := transferList.Encode(countInputs, countOutputs, appendMetadataMaxLen)
	if appendMetaData == nil {
		return nil
	}

	return coinspark.MetadataAppend(metadata, metadataMaxLen, appendMetaData)
}

func CreateAssetRef() coinspark.CoinSparkAssetRef {
	fmt.Println("\nFormatting an asset reference for users...\n")

	assetRef := coinspark.CoinSparkAssetRef{}

	assetRef.BlockNum = 456789
	assetRef.TxOffset = 65432
	assetRef.TxIDPrefix = [2]byte{0xa0, 0x5b}

	fmt.Println("Asset reference: ", assetRef.String())

	return assetRef
}

func readAssetRef() coinspark.CoinSparkAssetRef {
	fmt.Println("\nReading a user-provided asset reference...\n")

	assetRef := coinspark.CoinSparkAssetRef{}

	if assetRef.Decode("456789-65432-23456") {
		fmt.Println("Block number: ", assetRef.BlockNum)
		fmt.Println("Byte offset: ", assetRef.TxOffset)
		fmt.Println("TxID prefix: ", fmt.Sprintf("%02X%02X", assetRef.TxIDPrefix[0], assetRef.TxIDPrefix[1]))

		fmt.Printf(assetRef.String())

	} else {
		fmt.Println("Asset reference could not be read!")
	}

	return assetRef
}

func main() {
	CreateCoinSparkAddress()
	DecodeCoinSparkAddress()

	ProcessTransaction([]string{"6A2853504B6750A4AE00F454956DF4C7D6DE7BF8192486006A4ADF65B048BF847FE26D70588E9FA828D5"}, 15856)
	ProcessTransaction([]string{"abc", "6A2053504B743F282321E438188C4B381807227C10812B47920642B32E12417D8279", "def"}, 59364)
	ProcessTransaction([]string{"6A2553504B0872876AAE4C1CC00A747A3E6F1BC14CD7752DA0D507BD05ED903A1C8407CCE38087"}, 1925)

	metadataTransfers := coinspark.ScriptToMetadata("6A2053504B743F282321E438188C4B381807227C10812B47920642B32E12417D8279", true)
	EncodeMetaDataToHex(metadataTransfers)

	genesis := CreateGenesis()
	fmt.Println(genesis.String())

	message := CreateMessage()
	fmt.Println(message.String())

	transferList := CreateTransferList()
	fmt.Println(transferList.String())

	paymentRef := CreatePaymentRef()
	fmt.Println(paymentRef.String())

	metadata := CoinSparkPaymentRefTransfersEncode(paymentRef, transferList, 3, 5, 40)

	rawBinaryTransactions := [][]byte{EncodeMetaData(metadata), []byte{}, []byte{}, []byte{}, []byte{}}
	ProcessTransactionRawBinary(rawBinaryTransactions, 3)

	assetRef := CreateAssetRef()
	fmt.Println(assetRef.String())

	readAssetRef()
}
