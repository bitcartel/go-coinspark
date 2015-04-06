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

func main() {
	CreateCoinSparkAddress()
	DecodeCoinSparkAddress()

	ProcessTransaction([]string{"6A2853504B6750A4AE00F454956DF4C7D6DE7BF8192486006A4ADF65B048BF847FE26D70588E9FA828D5"}, 15856)
	ProcessTransaction([]string{"abc", "6A2053504B743F282321E438188C4B381807227C10812B47920642B32E12417D8279", "def"}, 59364)
	ProcessTransaction([]string{"6A2553504B0872876AAE4C1CC00A747A3E6F1BC14CD7752DA0D507BD05ED903A1C8407CCE38087"}, 1925)

}
