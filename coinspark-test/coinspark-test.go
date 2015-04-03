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

// Package main expects an input file containing CoinSpark test data.
package main

import (
	coinspark "../coinspark"
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"math/rand"
)

func ProcessInput(path string) {
	file, err := os.Open(path)

	if err != nil {

		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)

	if !scanner.Scan() {
		fmt.Println("No header")
		os.Exit(1)
	}

	header := scanner.Text()
	scanner.Scan()
	header = strings.TrimSpace(header)

	switch header {
	case "CoinSpark Address Tests Input":
		ProcessAddressTests(scanner)
	case "CoinSpark Genesis Tests Input":
		ProcessGenesisTests(scanner)
	case "CoinSpark AssetRef Tests Input":
		ProcessAssetRefTests(scanner)
	case "CoinSpark AssetHash Tests Input":
		ProcessAssetHashTests(scanner)
	case "CoinSpark Transfer Tests Input":
		ProcessTransferTests(scanner)
	case "CoinSpark MessageHash Tests Input":
		ProcessMessageHashTests(scanner)
	case "CoinSpark Script Tests Input":
		ProcessScriptTests(scanner)
	default:
		fmt.Println("Unknown header / Unknown input test")
		os.Exit(1)
	}
}

func unpackLines(s []string, vars ...*string) {
	for i, str := range s {
		*vars[i] = str
	}
}

func getInputLine(scanner *bufio.Scanner) string {
	if !scanner.Scan() {
		return ""
	}
	s := scanner.Text()
	pos := strings.Index(s, " # ")
	if pos != -1 {
		s = s[0:pos]
	}
	return s
}

func getInputLines(scanner *bufio.Scanner, n int) []string {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = getInputLine(scanner)
	}
	return lines[:]
}

func splitToCoinSparkSatoshiQty(s string, f func(string) coinspark.CoinSparkSatoshiQty) []coinspark.CoinSparkSatoshiQty {
	vs := strings.Split(s, ",")
	vsf := make([]coinspark.CoinSparkSatoshiQty, len(vs))
	for i, v := range vs {
		vsf[i] = f(v)
	}
	return vsf
}

func splitToCoinSparkAssetQty(s string, f func(string) coinspark.CoinSparkAssetQty) []coinspark.CoinSparkAssetQty {
	vs := strings.Split(s, ",")
	vsf := make([]coinspark.CoinSparkAssetQty, len(vs))
	for i, v := range vs {
		vsf[i] = f(v)
	}
	return vsf
}

func splitToInt64(s string, f func(string) int64) []int64 {
	vs := strings.Split(s, ",")
	vsf := make([]int64, len(vs))
	for i, v := range vs {
		//vsf.append( f(v) )
		vsf[i] = f(v)
	}
	return vsf
}

func splitToBool(s string, f func(string) bool) []bool {
	vs := strings.Split(s, ",")
	vsf := make([]bool, len(vs))
	for i, v := range vs {
		//vsf.append( f(v) )
		vsf[i] = f(v)
	}
	return vsf
}

func intToCoinSparkSatoshiQty(v string) coinspark.CoinSparkSatoshiQty {
	x, _ := strconv.ParseInt(v, 10, 64)
	return coinspark.CoinSparkSatoshiQty(x)
}

func intToCoinSparkAssetQty(v string) coinspark.CoinSparkAssetQty {
	x, _ := strconv.ParseInt(v, 10, 64)
	return coinspark.CoinSparkAssetQty(x)
}

func intToBool(v string) bool {
	x, _ := strconv.Atoi(v)
	return (x == 1)
}

func ProcessGenesisTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark Genesis Tests Output\n")

	for true {
		var firstSpentTxId, firstSpentVout, metadataHex, outputsSatoshisString, outputsRegularString, feeSatoshis, dummy string
		lines := getInputLines(scanner, 7)
		if lines[0] == "" {
			break
		}

		unpackLines(lines, &firstSpentTxId, &firstSpentVout, &metadataHex, &outputsSatoshisString, &outputsRegularString, &feeSatoshis, &dummy)
		// Break apart and decode the input lines

		genesis := new(coinspark.CoinSparkGenesis)

		metadata, err := hex.DecodeString(metadataHex)
		if err != nil {
			fmt.Println("metadataHex = " + metadataHex)
			fmt.Println(err)
			os.Exit(1)
		}

		if !genesis.Decode(metadata) {
			fmt.Println("Failed to decode genesis metadata: " + metadataHex)
			os.Exit(1)
		}

		outputsSatoshis := splitToCoinSparkSatoshiQty(outputsSatoshisString, intToCoinSparkSatoshiQty)
		outputsRegular := splitToBool(outputsRegularString, intToBool)

		countOutputs := len(outputsRegular)

		validFeeSatoshis := genesis.CalcMinFee(outputsSatoshis, outputsRegular)

		// Perform the genesis calculation

		fee, err := strconv.ParseInt(feeSatoshis, 10, 64)

		var outputBalances []coinspark.CoinSparkAssetQty

		if coinspark.CoinSparkSatoshiQty(fee) >= validFeeSatoshis {
			outputBalances = genesis.Apply(outputsRegular)
		} else {
			outputBalances = make([]coinspark.CoinSparkAssetQty, countOutputs)
			for i := 0; i < countOutputs; i++ {
				outputBalances[i] = 0
			}
		}

		// Output the results

		fmt.Printf("%d # transaction fee satoshis to be valid\n", validFeeSatoshis)
		for i := 0; i < countOutputs; i++ {
			fmt.Printf("%d", outputBalances[i])
			if (i + 1) != countOutputs {
				fmt.Printf(",")
			}
		}
		fmt.Println(" # units of the asset in each output")
		vout, err := strconv.Atoi(firstSpentVout)
		fmt.Println(genesis.CalcAssetURL(firstSpentTxId, vout) + " # asset web page URL\n")
	}
}

func ProcessAddressTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark Address Tests Output\n")

	for true {

		inputLine := getInputLine(scanner)
		if inputLine == "" {
			break
		}

		address := new(coinspark.CoinSparkAddress)
		if address.Decode(inputLine) {
			fmt.Printf(address.String())
		} else {
			fmt.Println("Failed to decode address: " + inputLine)
			os.Exit(1)
		}

		encodedBytes := address.Encode()
		if encodedBytes == nil {
			fmt.Println("Encode returned nil for " + inputLine)
			os.Exit(1)
		}
		encoded := string(encodedBytes)

		if encoded != inputLine {
			fmt.Println("Encode address mismatch: " + encoded + " should be " + inputLine)
			os.Exit(1)
		}

		if !address.Match(address) {
			fmt.Println("Failed to match address to itself!")
			os.Exit(1)
		}
	}

}

func ProcessAssetRefTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark AssetRef Tests Output\n")

	for true {
		inputLine := getInputLine(scanner)
		if inputLine == "" {
			break
		}

		assetRef := new(coinspark.CoinSparkAssetRef)
		if assetRef.Decode(inputLine) {
			fmt.Print(assetRef)
		} else {
			fmt.Println("Failed to decode AssetRef: " + inputLine)
			os.Exit(1)
		}
		encodedBytes := assetRef.Encode()
		if encodedBytes == nil {
			fmt.Println("Encode returned nil for " + inputLine)
			os.Exit(1)
		}
		encoded := string(encodedBytes)
		if encoded != inputLine {
			fmt.Println("Encode AssetRef mismatch: " + encoded + " should be " + inputLine)
			os.Exit(1)
		}

		if !assetRef.Match(assetRef) {
			fmt.Println("Failed to match assetRef to itself!")
			os.Exit(1)
		}
	}
}

func ProcessAssetHashTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark AssetHash Tests Output\n")

	for true {
		inputLines := getInputLines(scanner, 10)
		if inputLines[0] == "" {
			break
		}

		var name, issuer, description, units, issueDate, expiryDate, interestRate, multiple, contractContent, dummy string
		unpackLines(inputLines, &name, &issuer, &description, &units, &issueDate, &expiryDate, &interestRate, &multiple, &contractContent, &dummy)

		contractBytes := []byte(contractContent)
		interestRateF64, _ := strconv.ParseFloat(interestRate, 64)
		multipleF64, _ := strconv.ParseFloat(multiple, 64)
		hash := coinspark.CoinSparkCalcAssetHash(name, issuer, description, units, issueDate, expiryDate, interestRateF64, multipleF64, contractBytes)

		fmt.Println(strings.ToUpper(hex.EncodeToString(hash[:])))

	}
}

func ProcessMessageHashTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark MessageHash Tests Output\n")
	for true {
		inputLines := getInputLines(scanner, 2)
		if inputLines[0] == "" {
			break
		}

		var saltString, countPartsString string
		unpackLines(inputLines, &saltString, &countPartsString)

		saltBytes := []byte(saltString)
		countParts16, _ := strconv.ParseInt(countPartsString, 10, 16)
		countParts := int(countParts16)

		numLinesToRead := 3*countParts + 1
		inputLines = getInputLines(scanner, numLinesToRead)
		if inputLines[0] == "" {
			break
		}

		messageParts := make([]coinspark.CoinSparkMessagePart, 0)
		index := 0
		for len(messageParts) < countParts && index < numLinesToRead {
			var part coinspark.CoinSparkMessagePart
			part.MimeType = inputLines[index]
			index++
			part.FileName = inputLines[index]
			index++
			part.Content = []byte(inputLines[index])
			index++
			messageParts = append(messageParts, part)
		}

		hash := coinspark.CoinSparkCalcMessageHash(saltBytes, messageParts)
		hashHexString := strings.ToUpper(hex.EncodeToString(hash))
		fmt.Println(hashHexString)
	}
}

func ProcessTransferTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark Transfer Tests Output\n")

	for true {
		inputLines := getInputLines(scanner, 8)
		if inputLines[0] == "" {
			break
		}

		var genesisMetadataHex, assetRefString, transfersMetadataHex, inputBalancesString, outputsSatoshisString, outputsRegularString, feeSatoshis, dummy string
		unpackLines(inputLines, &genesisMetadataHex, &assetRefString, &transfersMetadataHex, &inputBalancesString, &outputsSatoshisString, &outputsRegularString, &feeSatoshis, &dummy)
		genesis := new(coinspark.CoinSparkGenesis)
		genesisMetadataBytes, _ := hex.DecodeString(genesisMetadataHex)
		if !genesis.Decode(genesisMetadataBytes) {
			fmt.Println("Failed to decode genesis metadata: " + genesisMetadataHex)
			os.Exit(1)
		}
		assetRef := new(coinspark.CoinSparkAssetRef)
		if !assetRef.Decode(assetRefString) {
			fmt.Println("Failed to decode asset reference: " + assetRefString)
		}

		inputBalances := splitToCoinSparkAssetQty(inputBalancesString, intToCoinSparkAssetQty)
		outputsSatoshis := splitToCoinSparkSatoshiQty(outputsSatoshisString, intToCoinSparkSatoshiQty)
		outputsRegular := splitToBool(outputsRegularString, intToBool)

		countInputs := len(inputBalances)
		countOutputs := len(outputsSatoshis)

		metadata, _ := hex.DecodeString(transfersMetadataHex)
		transfers := new(coinspark.CoinSparkTransferList)
		if transfers.Decode(metadata, countInputs, countOutputs) == 0 {
			fmt.Println("Failed to decode transfers metadata: " + transfersMetadataHex)
			os.Exit(1)
		}

		validFeeSatoshis := transfers.CalcMinFee(countInputs, outputsSatoshis, outputsRegular)

		// Perform the transfer calculation and get default flags
		var outputBalances []coinspark.CoinSparkAssetQty
		var outputsDefault []bool

		if intToCoinSparkSatoshiQty(feeSatoshis) >= validFeeSatoshis {
			outputBalances = transfers.Apply(assetRef, genesis, inputBalances, outputsRegular)
		} else {
			outputBalances = transfers.ApplyNone(inputBalances, outputsRegular)
		}
		outputsDefault = transfers.DefaultOutputs(countInputs, outputsRegular)

		// Output the results

		fmt.Printf("%d # transaction fee satoshis to be valid\n", validFeeSatoshis)

		buffer := bytes.Buffer{}
		for index, outputBalance := range outputBalances {
			buffer.WriteString(fmt.Sprintf("%d", outputBalance))
			if index+1 != len(outputBalances) {
				buffer.WriteString(",")
			}
		}
		buffer.WriteString(" # units of this asset in each output\n")

		for index, outputDefault := range outputsDefault {
			if outputDefault == true {
				buffer.WriteString("1")
			} else {
				buffer.WriteString("0")
			}
			if index+1 != len(outputBalances) {
				buffer.WriteString(",")
			}
		}
		buffer.WriteString(" # boolean flags whether each output is in a default route\n")
		fmt.Println(buffer.String())

		// Test the net and gross calculations using the input balances as example net values

		for _, inputBalance := range inputBalances {
			testGrossBalance := genesis.CalcGross(inputBalance)
			testNetBalance := genesis.CalcNet(testGrossBalance)

			if inputBalance != testNetBalance {
				fmt.Printf("Net to gross to net mismatch: %d -> %d -> %d !\n", inputBalance, testGrossBalance, testNetBalance)
				os.Exit(1)
			}
		}

	}
}

func ProcessScriptTests(scanner *bufio.Scanner) {
	fmt.Println("CoinSpark Script Tests Output\n")

	for true {
		inputLines := getInputLines(scanner, 4)
		if inputLines[0] == "" {
			break
		}

		var countInputsString, countOutputsString, scriptPubKeyHex, dummy string
		unpackLines(inputLines, &countInputsString, &countOutputsString, &scriptPubKeyHex, &dummy)
		tmp, _ := strconv.ParseInt(countInputsString, 10, 64)
		countInputs := int(tmp)
		tmp, _ = strconv.ParseInt(countOutputsString, 10, 64)
		countOutputs := int(tmp)

		metadata := coinspark.ScriptToMetadata(scriptPubKeyHex, true)
		if metadata == nil {
			fmt.Println("Could not decode script metadata: ", scriptPubKeyHex)
			os.Exit(1)
		}

		//# Read in the different types of metadata
		genesis := coinspark.CoinSparkGenesis{}
		hasGenesis := genesis.Decode(metadata)

		var paymentRef coinspark.CoinSparkPaymentRef

		hasPaymentRef := paymentRef.Decode(metadata)

		transfers := coinspark.CoinSparkTransferList{}
		hasTransfers := transfers.Decode(metadata, countInputs, countOutputs) >= 1

		message := coinspark.CoinSparkMessage{}
		hasMessage := message.Decode(metadata, countOutputs)

		// Output the toString()s

		if hasGenesis {
			fmt.Print(genesis.String())
		}
		if hasPaymentRef {
			fmt.Print(paymentRef.String())
		}
		if hasTransfers {
			fmt.Print(transfers.String())
		}
		if hasMessage {
			fmt.Print(message.String())
		}

		// Re-encode

		var testMetadata []byte
		testMetadataMaxLen := len(metadata)
		var nextMetadata []byte
		nextMetadataMaxLen := testMetadataMaxLen

		encodeOrder := []string{"genesis", "paymentRef", "transfers", "message"}

		for _, encodeField := range encodeOrder {
			triedNextMetadata := false

			if encodeField == "genesis" {
				if hasGenesis {
					_, nextMetadata = genesis.Encode(nextMetadataMaxLen)
					triedNextMetadata = true
				}
			} else if encodeField == "paymentRef" {
				if hasPaymentRef {
					nextMetadata = paymentRef.Encode(nextMetadataMaxLen)
					triedNextMetadata = true
				}
			} else if encodeField == "transfers" {
				if hasTransfers {
					nextMetadata = transfers.Encode(countInputs, countOutputs, nextMetadataMaxLen)
					triedNextMetadata = true
				}
			} else if encodeField == "message" {
				if hasMessage {
					nextMetadata = message.Encode(countOutputs, nextMetadataMaxLen)
					triedNextMetadata = true
				}
			}

			if triedNextMetadata {
				if nextMetadata == nil {
					fmt.Println("Failed to reencode ", encodeField, " metadata!")
					os.Exit(1)
				}
				if len(testMetadata) > 0 {
					testMetadata = coinspark.MetadataAppend(testMetadata, testMetadataMaxLen, nextMetadata)
					if len(testMetadata) > testMetadataMaxLen {
						fmt.Println("Insufficient space to append ", encodeField, " metadata!")
						os.Exit(1)
					}
				} else {
					testMetadata = nextMetadata
				}

				nextMetadataMaxLen = coinspark.MetadataMaxAppendLen(testMetadata, testMetadataMaxLen)
			}
		}

		// Test other library functions while we are here

		
		if hasGenesis {
			if !genesis.Match(&genesis, true) {
				fmt.Println("Failed to match genesis to itself!")
				os.Exit(1)
			}
				
			if genesis.CalcHashLen(len(metadata))!=genesis.GetHashLen() {
				// assumes that metadata only contains genesis
				fmt.Println("Failed to calculate matching hash length!")
				fmt.Println("genesis.GetHashLen() = ", genesis.GetHashLen() )
				fmt.Println("genesis.CalcHashLen(len(metadata)) = ", genesis.CalcHashLen(len(metadata)))
				fmt.Println("len(metadata) = ", len(metadata))
				os.Exit(1)
			}
				
			testGenesis:=coinspark.CoinSparkGenesis{}
			testGenesis.Decode(metadata)
			
			rounding:=rand.Intn(3)-1 // random is 0..2, so minus 1 for -1,0 or 1
			
			testGenesis.SetQty(0, 0)
			testGenesis.SetQty(genesis.GetQty(), rounding)
			
			testGenesis.SetChargeFlat(0, 0)
			testGenesis.SetChargeFlat(genesis.GetChargeFlat(), rounding)
			
			if !genesis.Match(&testGenesis, false) {
				fmt.Println("Mismatch on genesis rounding!")
				os.Exit(1)
			}
		}
				
		if hasPaymentRef {
			if !paymentRef.Match(&paymentRef) {
				fmt.Println("Failed to match paymentRef to itself!")
				os.Exit(1)
			}
		}
				
		if hasTransfers {
			if !transfers.Match(&transfers, true) {
				fmt.Println("Failed to strictly match transfers to itself!")
				os.Exit(1)
			}

			if !transfers.Match(&transfers, false) {
				fmt.Println("Failed to leniently match transfers to itself!")
				os.Exit(1)
			}
		}

		if hasMessage {
			if !message.Match(&message, true) {
				fmt.Println("Failed to strictly match message to itself!")
				os.Exit(1)
			}
				
			if !message.Match(&message, false) {
				fmt.Println("Failed to leniently match message to itself!")
				os.Exit(1)
			}

			messageEncode:=message.Encode(countOutputs, len(metadata)) // encode on its own to check calcHashLen()
			
			if message.CalcHashLen(countOutputs, len(messageEncode))!=message.GetHashLen() {
				fmt.Println("Failed to calculate matching message hash length!")
				os.Exit(1)
			}
		}

				
		// Compare to the original

		encoded := coinspark.MetadataToScript(testMetadata, true)

		if encoded != scriptPubKeyHex {
			fmt.Println("Encode metadata mismatch: " + encoded + " should be " + scriptPubKeyHex)
			os.Exit(1)
		}

		checkMetadata := coinspark.ScriptToMetadata(coinspark.MetadataToScript(testMetadata, false), false)

		if bytes.Compare(checkMetadata, testMetadata) != 0 {
			fmt.Println("Binary metadata to/from script mismatch!")
			os.Exit(1)
		}
	}
}

func main() {
	numArgs := len(os.Args)
	if numArgs <= 1 {
		fmt.Println("The input filename is missing")
		os.Exit(1)
	}

	ProcessInput(os.Args[1])

}

