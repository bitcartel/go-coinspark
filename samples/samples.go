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

		addressString:=address.Encode()

        if addressString != "" {
            fmt.Println("CoinSpark address: ", addressString)
		} else {
            fmt.Println("CoinSpark address encode failed!")
		}
}

func DecodeCoinSparkAddress() {
        fmt.Println("\nDecoding a CoinSpark address...\n");

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

func main() {
        CreateCoinSparkAddress()
        DecodeCoinSparkAddress()
}

