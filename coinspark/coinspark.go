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

// Package coinspark ports the CoinSpark library v2.1 to Golang.
// More information can be found here: http://coinspark.org
package coinspark

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

//#define TRUE 1
//#define FALSE 0

// Return smaller of two ints
func COINSPARK_MIN(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// Return smaller of two int16s
func COINSPARK_MIN16(a int16, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

// Return smaller of two CoinSparkAssetQtys
func COINSPARK_MINASSETQTY(a CoinSparkAssetQty, b CoinSparkAssetQty) CoinSparkAssetQty {
	if a < b {
		return a
	}
	return b
}

// Return smaller of two CoinSparkSatoshiQtys
func COINSPARK_MINSATOSHIQTY(a CoinSparkSatoshiQty, b CoinSparkSatoshiQty) CoinSparkSatoshiQty {
	if a < b {
		return a
	}
	return b
}

// Return bigger of two ints
func COINSPARK_MAX(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// CoinSpark constants and other macros
const (
	COINSPARK_SATOSHI_QTY_MAX = 2100000000000000
	COINSPARK_ASSET_QTY_MAX   = 100000000000000
	COINSPARK_PAYMENT_REF_MAX = 0xFFFFFFFFFFFFF // 2^52-1

	COINSPARK_GENESIS_QTY_MANTISSA_MIN                    = 1
	COINSPARK_GENESIS_QTY_MANTISSA_MAX                    = 1000
	COINSPARK_GENESIS_QTY_EXPONENT_MIN                    = 0
	COINSPARK_GENESIS_QTY_EXPONENT_MAX                    = 11
	COINSPARK_GENESIS_CHARGE_FLAT_MAX                     = 5000
	COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MIN            = 0
	COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX            = 100
	COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX_IF_EXP_MAX = 50
	COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MIN            = 0
	COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MAX            = 2
	COINSPARK_GENESIS_CHARGE_BASIS_POINTS_MIN             = 0
	COINSPARK_GENESIS_CHARGE_BASIS_POINTS_MAX             = 250
	COINSPARK_GENESIS_DOMAIN_NAME_MAX_LEN                 = 32
	COINSPARK_GENESIS_PAGE_PATH_MAX_LEN                   = 24
	COINSPARK_GENESIS_HASH_MIN_LEN                        = 12
	COINSPARK_GENESIS_HASH_MAX_LEN                        = 32

	COINSPARK_ASSETREF_BLOCK_NUM_MAX   = 4294967295
	COINSPARK_ASSETREF_TX_OFFSET_MAX   = 4294967295
	COINSPARK_ASSETREF_TXID_PREFIX_LEN = 2

	COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE = -1 // magic number for a default route

	COINSPARK_IO_INDEX_MAX = 65535

	COINSPARK_ADDRESS_FLAG_ASSETS        = 1
	COINSPARK_ADDRESS_FLAG_PAYMENT_REFS  = 2
	COINSPARK_ADDRESS_FLAG_TEXT_MESSAGES = 4
	COINSPARK_ADDRESS_FLAG_FILE_MESSAGES = 8
	COINSPARK_ADDRESS_FLAG_MASK          = 0x7FFFFF // 23 bits are currently usable
)

// CoinSpark type definitions
type CoinSparkSatoshiQty int64
type CoinSparkAssetQty int64
type CoinSparkIOIndex int
type CoinSparkAddressFlags int32

type CoinSparkPaymentRef struct {
	ref uint64
}

type CoinSparkAddress struct {
	bitcoinAddress string
	addressFlags   CoinSparkAddressFlags
	paymentRef     CoinSparkPaymentRef
}

type CoinSparkGenesis struct {
	qtyMantissa        int16
	qtyExponent        int16
	chargeFlatMantissa int16
	chargeFlatExponent int16
	chargeBasisPoints  int16 // one hundredths of a percent
	useHttps           bool
	domainName         string
	usePrefix          bool   // prefix coinspark/ in asset web page URL path
	pagePath           string // Max len should be COINSPARK_GENESIS_PAGE_PATH_MAX_LEN + 1
	assetHash          []byte // Max len should be COINSPARK_GENESIS_HASH_MAX_LEN
	assetHashLen       int    // number of bytes in assetHash that are valid for comparison
}

type CoinSparkAssetRef struct {
	blockNum   int64                                    // block in which genesis transaction is confirmed
	txOffset   int64                                    // byte offset within that block
	txIDPrefix [COINSPARK_ASSETREF_TXID_PREFIX_LEN]byte // first bytes of genesis transaction id
}

type CoinSparkIORange struct {
	first CoinSparkIOIndex
	count CoinSparkIOIndex
}

type CoinSparkTransfer struct {
	assetRef     CoinSparkAssetRef
	inputs       CoinSparkIORange
	outputs      CoinSparkIORange
	qtyPerOutput CoinSparkAssetQty
}

type CoinSparkTransferList struct {
	transfers []CoinSparkTransfer
}

type CoinSparkMessage struct {
	useHttps     bool
	serverHost   string             // max len COINSPARK_MESSAGE_SERVER_HOST_MAX_LEN+1
	usePrefix    bool               // prefix coinspark/ in server path
	serverPath   string             // max len COINSPARK_MESSAGE_SERVER_PATH_MAX_LEN+1
	isPublic     bool               // is the message publicly viewable
	outputRanges []CoinSparkIORange // array of output ranges
	hash         []byte
	hashLen      int // number of bytes in hash that are valid for comparison/encoding
}

type CoinSparkMessagePart struct {
	MimeType string
	FileName string
	Content  []byte
}

/*
type CoinSparkDomainPath struct {
	domainName string
	path string
	useHttps bool
	usePrefix bool
}
*/

// Constants used internally

const (
	COINSPARK_UNSIGNED_BYTE_MAX    = 0xFF
	COINSPARK_UNSIGNED_2_BYTES_MAX = 0xFFFF
	COINSPARK_UNSIGNED_3_BYTES_MAX = 0xFFFFFF
	COINSPARK_UNSIGNED_4_BYTES_MAX = 0xFFFFFFFF

	COINSPARK_METADATA_IDENTIFIER     = "SPK"
	COINSPARK_METADATA_IDENTIFIER_LEN = 3
	COINSPARK_LENGTH_PREFIX_MAX       = 96
	COINSPARK_GENESIS_PREFIX          = 'g'
	COINSPARK_TRANSFERS_PREFIX        = 't'
	COINSPARK_PAYMENTREF_PREFIX       = 'r'
	COINSPARK_MESSAGE_PREFIX          = 'm'
	COINSPARK_DUMMY_PREFIX            = '?' // for none

	COINSPARK_FEE_BASIS_MAX_SATOSHIS = 1000

	COINSPARK_GENESIS_QTY_FLAGS_LENGTH              = 2
	COINSPARK_GENESIS_QTY_MASK                      = 0x3FFF
	COINSPARK_GENESIS_QTY_EXPONENT_MULTIPLE         = 1001
	COINSPARK_GENESIS_FLAG_CHARGE_FLAT              = 0x4000
	COINSPARK_GENESIS_FLAG_CHARGE_BPS               = 0x8000
	COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MULTIPLE = 101
	COINSPARK_GENESIS_CHARGE_FLAT_LENGTH            = 1
	COINSPARK_GENESIS_CHARGE_BPS_LENGTH             = 1

	COINSPARK_DOMAIN_PACKING_PREFIX_MASK         = 0xC0
	COINSPARK_DOMAIN_PACKING_PREFIX_SHIFT        = 6
	COINSPARK_DOMAIN_PACKING_SUFFIX_MASK         = 0x3F
	COINSPARK_DOMAIN_PACKING_SUFFIX_MAX          = 61
	COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4_NO_PATH = 62 // messages only
	COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4         = 63
	COINSPARK_DOMAIN_PACKING_IPv4_HTTPS          = 0x40
	COINSPARK_DOMAIN_PACKING_IPv4_NO_PATH_PREFIX = 0x80

	COINSPARK_DOMAIN_PATH_ENCODE_BASE    = 40
	COINSPARK_DOMAIN_PATH_FALSE_END_CHAR = '<'
	COINSPARK_DOMAIN_PATH_TRUE_END_CHAR  = '>'

	COINSPARK_PACKING_GENESIS_MASK      = 0xC0
	COINSPARK_PACKING_GENESIS_PREV      = 0x00
	COINSPARK_PACKING_GENESIS_3_3_BYTES = 0x40 // 3 bytes for block index, 3 for txn offset
	COINSPARK_PACKING_GENESIS_3_4_BYTES = 0x80 // 3 bytes for block index, 4 for txn offset
	COINSPARK_PACKING_GENESIS_4_4_BYTES = 0xC0 // 4 bytes for block index, 4 for txn offset

	COINSPARK_PACKING_INDICES_MASK    = 0x38
	COINSPARK_PACKING_INDICES_0P_0P   = 0x00 // input 0 only or previous, output 0 only or previous
	COINSPARK_PACKING_INDICES_0P_1S   = 0x08 // input 0 only or previous, output 1 only or subsequent single
	COINSPARK_PACKING_INDICES_0P_ALL  = 0x10 // input 0 only or previous, all outputs
	COINSPARK_PACKING_INDICES_1S_0P   = 0x18 // input 1 only or subsequent single, output 0 only or previous
	COINSPARK_PACKING_INDICES_ALL_0P  = 0x20 // all inputs, output 0 only or previous
	COINSPARK_PACKING_INDICES_ALL_1S  = 0x28 // all inputs, output 1 only or subsequent single
	COINSPARK_PACKING_INDICES_ALL_ALL = 0x30 // all inputs, all outputs
	COINSPARK_PACKING_INDICES_EXTEND  = 0x38 // use second byte for more extensive information

	COINSPARK_PACKING_EXTEND_INPUTS_SHIFT  = 3
	COINSPARK_PACKING_EXTEND_OUTPUTS_SHIFT = 0

	COINSPARK_PACKING_EXTEND_MASK = 0x07
	COINSPARK_PACKING_EXTEND_0P   = 0x00 // index 0 only or previous

	COINSPARK_PACKING_EXTEND_PUBLIC = 0x00 // this is public (messages only)

	COINSPARK_PACKING_EXTEND_1S        = 0x01 // index 1 only or subsequent single
	COINSPARK_PACKING_EXTEND_BYTE      = 0x02 // 1 byte for single index
	COINSPARK_PACKING_EXTEND_2_BYTES   = 0x03 // 2 bytes for single index
	COINSPARK_PACKING_EXTEND_0_1_BYTE  = 0x01 // starting at 0, 1 byte for count (messages only)
	COINSPARK_PACKING_EXTEND_1_0_BYTE  = 0x02 // 1 byte for single index, count is 1
	COINSPARK_PACKING_EXTEND_2_0_BYTES = 0x03 // 2 bytes for single index, count is 1

	COINSPARK_PACKING_EXTEND_1_1_BYTES = 0x04 // 1 byte for first index, 1 byte for count
	COINSPARK_PACKING_EXTEND_2_1_BYTES = 0x05 // 2 bytes for first index, 1 byte for count
	COINSPARK_PACKING_EXTEND_2_2_BYTES = 0x06 // 2 bytes for first index, 2 bytes for count
	COINSPARK_PACKING_EXTEND_ALL       = 0x07 // all inputs|outputs

	COINSPARK_PACKING_QUANTITY_MASK    = 0x07
	COINSPARK_PACKING_QUANTITY_1P      = 0x00 // quantity=1 or previous
	COINSPARK_PACKING_QUANTITY_1_BYTE  = 0x01
	COINSPARK_PACKING_QUANTITY_2_BYTES = 0x02
	COINSPARK_PACKING_QUANTITY_3_BYTES = 0x03
	COINSPARK_PACKING_QUANTITY_4_BYTES = 0x04
	COINSPARK_PACKING_QUANTITY_6_BYTES = 0x05
	COINSPARK_PACKING_QUANTITY_FLOAT   = 0x06
	COINSPARK_PACKING_QUANTITY_MAX     = 0x07 // transfer all quantity across

	COINSPARK_TRANSFER_QTY_FLOAT_LENGTH            = 2
	COINSPARK_TRANSFER_QTY_FLOAT_MANTISSA_MAX      = 1000
	COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MAX      = 11
	COINSPARK_TRANSFER_QTY_FLOAT_MASK              = 0x3FFF
	COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MULTIPLE = 1001

	COINSPARK_ADDRESS_PREFIX              = 's'
	COINSPARK_ADDRESS_FLAG_CHARS_MULTIPLE = 10
	COINSPARK_ADDRESS_CHAR_INCREMENT      = 13

	COINSPARK_OUTPUTS_MORE_FLAG     = 0x80
	COINSPARK_OUTPUTS_RESERVED_MASK = 0x60
	COINSPARK_OUTPUTS_TYPE_MASK     = 0x18
	COINSPARK_OUTPUTS_TYPE_SINGLE   = 0x00 // one output index (0...7)
	COINSPARK_OUTPUTS_TYPE_FIRST    = 0x08 // first (0...7) outputs
	COINSPARK_OUTPUTS_TYPE_UNUSED   = 0x10 // for future use
	COINSPARK_OUTPUTS_TYPE_EXTEND   = 0x18 // "extend", including public/all
	COINSPARK_OUTPUTS_VALUE_MASK    = 0x07
	COINSPARK_OUTPUTS_VALUE_MAX     = 7

	COINSPARK_MESSAGE_SERVER_HOST_MAX_LEN = 32
	COINSPARK_MESSAGE_SERVER_PATH_MAX_LEN = 24
	COINSPARK_MESSAGE_HASH_MIN_LEN        = 12
	COINSPARK_MESSAGE_HASH_MAX_LEN        = 32
	COINSPARK_MESSAGE_MAX_IO_RANGES       = 16
)

// Type definitions and constants used internally

type PackingType int

// options to use in order of priority
const (
	_0P PackingType = iota
	_1S
	_ALL
	_BYTE
	_2_BYTES
	_1_1_BYTES
	_2_1_BYTES
	_2_2_BYTES
	countPackingTypes
	firstPackingType = 0 // iota is reset to 0 at start of const
)

type OutputRangePacking struct {
	packing    int
	firstBytes int
	countBytes int
}

// Map is unordered so we have explicit order to range over
var packingExtendMapOrder = []string{"_0P", "_1S", "_ALL", "_1_0_BYTE", "_0_1_BYTE", "_2_0_BYTES", "_1_1_BYTES", "_2_1_BYTES", "_2_2_BYTES"}
var packingExtendMap = map[string]byte{
	"_0P":        COINSPARK_PACKING_EXTEND_0P,
	"_1S":        COINSPARK_PACKING_EXTEND_1S,
	"_ALL":       COINSPARK_PACKING_EXTEND_ALL,
	"_1_0_BYTE":  COINSPARK_PACKING_EXTEND_1_0_BYTE,
	"_0_1_BYTE":  COINSPARK_PACKING_EXTEND_0_1_BYTE,
	"_2_0_BYTES": COINSPARK_PACKING_EXTEND_2_0_BYTES,
	"_1_1_BYTES": COINSPARK_PACKING_EXTEND_1_1_BYTES,
	"_2_1_BYTES": COINSPARK_PACKING_EXTEND_2_1_BYTES,
	"_2_2_BYTES": COINSPARK_PACKING_EXTEND_2_2_BYTES}

// last two characters are end markers, < means false, > means true
var domainPathChars = []byte("0123456789abcdefghijklmnopqrstuvwxyz-.<>")

var integerToBase58 = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

var base58Minus49ToInteger = [...]int{
	0, 1, 2, 3, 4, 5, 6, 7, 8, -1, -1, -1, -1, -1, -1, -1,
	9, 10, 11, 12, 13, 14, 15, 16, -1, 17, 18, 19, 20, 21, -1, 22,
	23, 24, 25, 26, 27, 28, 29, 30, 31, 32, -1, -1, -1, -1, -1, -1,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, -1, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57}

var domainNamePrefixes = [...]string{
	"",
	"www."}

var domainNameSuffixes = [...]string{
	// leave space for 3 more in future
	"",
	// most common suffixes based on Alexa's top 1M sites as of 10 June 2014, with some manual adjustments

	".at",
	".au",
	".be",
	".biz",
	".br",
	".ca",
	".ch",
	".cn",
	".co.jp",
	".co.kr",
	".co.uk",
	".co.za",
	".co",
	".com.ar",
	".com.au",
	".com.br",
	".com.cn",
	".com.mx",
	".com.tr",
	".com.tw",
	".com.ua",
	".com",
	".cz",
	".de",
	".dk",
	".edu",
	".es",
	".eu",
	".fr",
	".gov",
	".gr",
	".hk",
	".hu",
	".il",
	".in",
	".info",
	".ir",
	".it",
	".jp",
	".kr",
	".me",
	".mx",
	".net",
	".nl",
	".no",
	".org",
	".pl",
	".ps",
	".ro",
	".ru",
	".se",
	".sg",
	".tr",
	".tv",
	".tw",
	".ua",
	".uk",
	".us",
	".vn"}

type PackingByteCounts struct {
	blockNumBytes     int
	txOffsetBytes     int
	txIDPrefixBytes   int
	firstInputBytes   int
	countInputsBytes  int
	firstOutputBytes  int
	countOutputsBytes int
	quantityBytes     int
}

// returns -1 if invalid
func Base58ToInteger(base58Character byte) int {

	if (base58Character < 49) || (base58Character > 122) {
		return -1
	}
	return base58Minus49ToInteger[base58Character-49]
}

// Set all fields in address to their default/zero values, which are not necessarily valid.
func (p *CoinSparkAddress) Clear() {
	p.bitcoinAddress = ""
	p.addressFlags = 0
	p.paymentRef = CoinSparkPaymentRef{0}
}

// Returns true if all values in the address are in their permitted ranges, false otherwise.
func (p *CoinSparkAddress) IsValid() bool {
	if p.bitcoinAddress == "" {
		return false
	}
	if (p.addressFlags & COINSPARK_ADDRESS_FLAG_MASK) != p.addressFlags {
		return false
	}

	return p.paymentRef.IsValid()
}

// Returns true if the two CoinSparkAddress structures are identical.
func (p *CoinSparkAddress) Match(other *CoinSparkAddress) bool {
	//	a1 := p.bitcoinAddress
	//	a2 := address.bitcoinAddress
	//	s1 := a1[:bytes.Index(a1[:], []byte{0x00})]
	//	s2 := a2[:bytes.Index(a2[:], []byte{0x00})]

	return (p.bitcoinAddress == other.bitcoinAddress && p.addressFlags == other.addressFlags && p.paymentRef == other.paymentRef)
}

// Decodes the CoinSpark address string into the fields in address.
// Returns true if the address could be successfully read, otherwise false.
func (p *CoinSparkAddress) Decode(sparkAddress string) bool {

	var bitcoinAddressLen, halfLength int
	var charIndex, charValue, addressFlagChars, paymentRefChars, extraDataChars int
	var multiplier uint64
	var stringBase58 [1024]byte
	bufBase58 := bytes.Buffer{}

	input := []byte(sparkAddress)
	inputLen := len(input)

	//  Check for basic validity

	if (inputLen < 2) || (inputLen > len(stringBase58)) {
		goto cannotDecodeAddress
	}

	if input[0] != COINSPARK_ADDRESS_PREFIX {
		goto cannotDecodeAddress
	}
	//  Convert from base 58

	for charIndex = 1; charIndex < inputLen; charIndex++ { // exclude first character
		charValue = Base58ToInteger(input[charIndex])
		if charValue < 0 {
			goto cannotDecodeAddress
		}
		stringBase58[charIndex] = byte(charValue)
	}

	//  De-obfuscate first half of address using second half

	halfLength = (inputLen + 1) / 2
	for charIndex = 1; charIndex < halfLength; charIndex++ { // exclude first character
		stringBase58[charIndex] = (stringBase58[charIndex] + 58 - stringBase58[inputLen-charIndex]) % 58
	}

	//  Get length of extra data

	charValue = int(stringBase58[1])
	addressFlagChars = charValue / COINSPARK_ADDRESS_FLAG_CHARS_MULTIPLE // keep as integer
	paymentRefChars = charValue % COINSPARK_ADDRESS_FLAG_CHARS_MULTIPLE
	extraDataChars = addressFlagChars + paymentRefChars

	if inputLen < (2 + extraDataChars) {
		goto cannotDecodeAddress
	}

	//  Check we have sufficient length for the decoded address

	bitcoinAddressLen = inputLen - 2 - extraDataChars
	//  Read the extra data for address flags

	p.addressFlags = 0
	multiplier = 1

	for charIndex = 0; charIndex < addressFlagChars; charIndex++ {
		charValue = int(stringBase58[2+charIndex])
		p.addressFlags += CoinSparkAddressFlags(uint64(charValue) * multiplier)
		multiplier *= 58
	}

	//  Read the extra data for payment reference

	p.paymentRef = CoinSparkPaymentRef{0}
	multiplier = 1

	for charIndex = 0; charIndex < paymentRefChars; charIndex++ {
		charValue = int(stringBase58[2+addressFlagChars+charIndex])
		p.paymentRef.ref += uint64(charValue) * multiplier
		multiplier *= 58
	}
	//  Convert the bitcoin address

	for charIndex = 0; charIndex < bitcoinAddressLen; charIndex++ {
		charValue = int(stringBase58[2+extraDataChars+charIndex])
		charValue += 58*2 - COINSPARK_ADDRESS_CHAR_INCREMENT // avoid worrying about the result of modulo on negative numbers in any language

		if extraDataChars > 0 {
			charValue -= int(stringBase58[2+charIndex%extraDataChars])
		}

		bufBase58.WriteByte(integerToBase58[charValue%58])
		//        p.bitcoinAddress[charIndex]=integerToBase58[charValue%58]
	}

	//p.bitcoinAddress[bitcoinAddressLen]=0 // C terminator byte

	p.bitcoinAddress = bufBase58.String()

	return p.IsValid()

cannotDecodeAddress:
	return false
}

// Encodes the fields in address to a byte array
// Returns the size of the byte array if successful, otherwise 0.
func (p *CoinSparkAddress) Encode() []byte {
	var bitcoinAddressLen, stringLen, halfLength int
	var charIndex, charValue, addressFlagChars, paymentRefChars, extraDataChars int
	var testAddressFlags CoinSparkAddressFlags
	var testPaymentRef uint64
	var stringBase58 [1024]byte

	buf := bytes.Buffer{}

	if !p.IsValid() {
		goto cannotEncodeAddress
	}

	//  Build up extra data for address flags

	addressFlagChars = 0
	testAddressFlags = p.addressFlags

	for testAddressFlags > 0 {
		stringBase58[2+addressFlagChars] = byte(testAddressFlags % 58)
		testAddressFlags /= 58 // keep as integer
		addressFlagChars++
	}

	//  Build up extra data for payment reference

	paymentRefChars = 0
	testPaymentRef = p.paymentRef.ref

	for testPaymentRef > 0 {
		stringBase58[2+addressFlagChars+paymentRefChars] = byte(testPaymentRef % 58)
		testPaymentRef /= 58 // keep as integer
		paymentRefChars++
	}

	//  Calculate/encode extra length and total length required

	extraDataChars = addressFlagChars + paymentRefChars
	bitcoinAddressLen = len(p.bitcoinAddress)
	stringLen = bitcoinAddressLen + 2 + extraDataChars

	stringBase58[1] = byte(addressFlagChars*COINSPARK_ADDRESS_FLAG_CHARS_MULTIPLE + paymentRefChars)

	//  Convert the bitcoin address

	for charIndex = 0; charIndex < bitcoinAddressLen; charIndex++ {
		charValue = Base58ToInteger(p.bitcoinAddress[charIndex])
		if charValue < 0 {
			fmt.Println("invalid base58 char")
			return nil //0 // invalid base58 character
		}

		charValue += COINSPARK_ADDRESS_CHAR_INCREMENT

		if extraDataChars > 0 {
			charValue += int(stringBase58[2+charIndex%extraDataChars])
		}

		stringBase58[2+extraDataChars+charIndex] = byte(charValue % 58)
	}

	//  Obfuscate first half of address using second half to prevent common prefixes

	halfLength = (stringLen + 1) / 2
	for charIndex = 1; charIndex < halfLength; charIndex++ { // exclude first character
		stringBase58[charIndex] = (stringBase58[charIndex] + stringBase58[stringLen-charIndex]) % 58
	}

	//  Convert to base 58 and add prefix and terminator
	buf.WriteByte(COINSPARK_ADDRESS_PREFIX)
	//    input[0]=COINSPARK_ADDRESS_PREFIX
	for charIndex = 1; charIndex < stringLen; charIndex++ {
		//        input[charIndex]=integerToBase58[stringBase58[charIndex]]
		buf.WriteByte(integerToBase58[stringBase58[charIndex]])
	}
	//    input[stringLen]=0

	return buf.Bytes()

cannotEncodeAddress:
	return nil
}

// Internal use only
type FlagToString struct {
	flag  CoinSparkAddressFlags
	label string
}

// Outputs the address to a string for debugging.
func (p *CoinSparkAddress) String() string {

	var flagOutput bool

	buffer := bytes.Buffer{} //NewBuffer(0) //make([]byte, 1024))

	flagsToString := []FlagToString{
		{COINSPARK_ADDRESS_FLAG_ASSETS, "assets"},
		{COINSPARK_ADDRESS_FLAG_PAYMENT_REFS, "payment references"},
		{COINSPARK_ADDRESS_FLAG_TEXT_MESSAGES, "text messages"},
		{COINSPARK_ADDRESS_FLAG_FILE_MESSAGES, "file messages"},
	}
	buffer.WriteString("COINSPARK ADDRESS\n")
	buffer.WriteString(fmt.Sprintf("  Bitcoin address: %s\n", p.bitcoinAddress))
	buffer.WriteString(fmt.Sprintf("    Address flags: %d", p.addressFlags))

	flagOutput = false

	for _, f := range flagsToString {
		if p.addressFlags&f.flag > 0 {
			if flagOutput {
				buffer.WriteString(", ")
			} else {
				buffer.WriteString(" [")
			}
			buffer.WriteString(f.label) //fmt.Sprintf("%s%s", flagOutput ? ", " : " [", f.label)
			flagOutput = true
		}
	}

	if flagOutput {
		buffer.WriteString("]")
	}
	buffer.WriteString("\n")

	buffer.WriteString(fmt.Sprintf("Payment reference: %d\n", p.paymentRef.ref))
	buffer.WriteString(fmt.Sprintf("END COINSPARK ADDRESS\n\n"))
	return buffer.String()
}

// Convenience constructor
func NewCoinSparkAddress(address string, flags CoinSparkAddressFlags, paymentRef CoinSparkPaymentRef) *CoinSparkAddress {
	p := new(CoinSparkAddress)
	p.bitcoinAddress = address
	p.addressFlags = flags
	p.paymentRef = paymentRef
	return p
}

// Set all fields in assetRef to their default/zero values, which are not necessarily valid.
func (p *CoinSparkAssetRef) Clear() {
	p.blockNum = 0
	p.txOffset = 0
	var x [COINSPARK_ASSETREF_TXID_PREFIX_LEN]byte
	p.txIDPrefix = x
}

func (p *CoinSparkAssetRef) StringInner(headers bool) string {

	buffer := bytes.Buffer{}

	//char buffer[1024], hex[17], *bufferPtr;
	//size_t bufferLength, copyLength;
	//bufferPtr=buffer;

	if headers {
		buffer.WriteString("COINSPARK ASSET REFERENCE\n")
	}

	var buf []byte = make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(p.blockNum))
	s := hex.EncodeToString(buf[0:4])

	buffer.WriteString(fmt.Sprintf("Genesis block index: %d (small endian hex %s)\n", p.blockNum, strings.ToUpper(s)))

	binary.LittleEndian.PutUint32(buf, uint32(p.txOffset))
	s = hex.EncodeToString(buf[0:4])

	buffer.WriteString(fmt.Sprintf(" Genesis txn offset: %d (small endian hex %s)\n", p.txOffset, strings.ToUpper(s)))

	s = hex.EncodeToString(p.txIDPrefix[:])
	buffer.WriteString(fmt.Sprintf("Genesis txid prefix: %s\n", strings.ToUpper(s)))

	if headers {
		buffer.WriteString("END COINSPARK ASSET REFERENCE\n\n")
	}

	return buffer.String()
}

// Outputs the assetRef to a string for debugging.
func (p *CoinSparkAssetRef) String() string {
	return p.StringInner(true)
}

// Returns true if all values in the asset reference are in their permitted ranges, false otherwise.
func (p *CoinSparkAssetRef) IsValid() bool {
	if p.blockNum != COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE {
		if p.blockNum < 0 || p.blockNum > COINSPARK_ASSETREF_BLOCK_NUM_MAX {
			goto assetRefIsInvalid
		}
		if p.txOffset < 0 || p.txOffset > COINSPARK_ASSETREF_TX_OFFSET_MAX {
			goto assetRefIsInvalid
		}
	}

	return true

assetRefIsInvalid:
	return false
}

// Returns true if the two CoinSparkAssetRef structures are identical
func (p *CoinSparkAssetRef) Match(other *CoinSparkAssetRef) bool {
	return (p.txIDPrefix == other.txIDPrefix &&
		p.txOffset == other.txOffset) &&
		(p.blockNum == other.blockNum)
}

// Encodes the assetRef to a byte array
func (p *CoinSparkAssetRef) Encode() []byte {
	buffer := bytes.Buffer{}
	var txIDPrefixInteger int

	if !p.IsValid() {
		goto cannotEncodeAssetRef
	}
	txIDPrefixInteger = 256*int(p.txIDPrefix[1]) + int(p.txIDPrefix[0])
	buffer.WriteString(fmt.Sprintf("%d-%d-%d", p.blockNum, p.txOffset, txIDPrefixInteger))
	return buffer.Bytes()

cannotEncodeAssetRef:
	return nil
}

// Decodes the CoinSpark asset reference string into assetRef.
// Returns true if the asset reference could be successfully read, otherwise false.
func (p *CoinSparkAssetRef) Decode(assetRef string) bool {
	var blockNum, txOffset, txIDPrefixInteger int
	n, err := fmt.Sscanf(assetRef, "%d-%d-%d", &blockNum, &txOffset, &txIDPrefixInteger)
	if n != 3 || err != nil {
		return false
	}

	if (txIDPrefixInteger < 0) || (txIDPrefixInteger > 0xFFFF) {
		return false
	}

	p.blockNum = int64(blockNum)
	p.txOffset = int64(txOffset)
	p.txIDPrefix = [2]byte{byte(txIDPrefixInteger % 256), byte(txIDPrefixInteger / 256)}
	return p.IsValid()
}

func NewCoinSparkAssetRef(blockNum int64, txOffset int64, txIDPrefix []byte) *CoinSparkAssetRef {
	p := new(CoinSparkAssetRef)
	p.blockNum = blockNum
	p.txOffset = txOffset
	p.txIDPrefix = [2]byte{txIDPrefix[0], txIDPrefix[1]}
	return p
}

// Compare two CoinSparkAssetRef objects, useful for sorting from lower to higher asset refereneces.
func (p *CoinSparkAssetRef) Compare(otherAssetRef *CoinSparkAssetRef) int {
	// -1 if this<otherAssetRef, 1 if otherAssetRef>this, 0 otherwise

	if p.blockNum != otherAssetRef.blockNum {
		if p.blockNum < otherAssetRef.blockNum {
			return -1
		} else {
			return 1
		}
	} else if p.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE { // # in this case don't compare other fields
		return 0
	} else if p.txOffset != otherAssetRef.txOffset {
		if p.txOffset < otherAssetRef.txOffset {
			return -1
		} else {
			return 1
		}
	} else {
		thisTxIDPrefixLower := strings.ToLower(hex.EncodeToString(p.txIDPrefix[:]))
		otherTxIDPrefixLower := strings.ToLower(hex.EncodeToString(otherAssetRef.txIDPrefix[:]))
		if thisTxIDPrefixLower != otherTxIDPrefixLower { // # comparing hex gives same order as comparing bytes
			if thisTxIDPrefixLower < otherTxIDPrefixLower {
				return -1
			} else {
				return 1
			}
		} else {
			return 0
		}
	}
}

// Calculates the assetHash for the key information from a CoinSpark asset web page JSON specification.
// All string parameters except contractContent must be passed using UTF-8 encoding.
// You may pass nil for any parameter which was not in the JSON.
// Note that you need to pass in the contract *content*, not its URL.
func CoinSparkCalcAssetHash(name string,
	issuer string,
	description string,
	units string,
	issueDate string,
	expiryDate string,
	interestRate float64,
	multiple float64,
	contractContent []byte) [sha256.Size]byte {

	buffer := bytes.Buffer{}

	name = strings.TrimSpace(name)
	issuer = strings.TrimSpace(issuer)
	description = strings.TrimSpace(description)
	units = strings.TrimSpace(units)
	issueDate = strings.TrimSpace(issueDate)
	expiryDate = strings.TrimSpace(expiryDate)

	buffer.WriteString(name)
	buffer.WriteByte(0x00)
	buffer.WriteString(issuer)
	buffer.WriteByte(0x00)
	buffer.WriteString(description)
	buffer.WriteByte(0x00)
	buffer.WriteString(units)
	buffer.WriteByte(0x00)
	buffer.WriteString(issueDate)
	buffer.WriteByte(0x00)
	buffer.WriteString(expiryDate)
	buffer.WriteByte(0x00)

	interestRateToHash := int64(interestRate*1000000.0 + 0.5)
	if multiple == 0.0 {
		multiple = 1
	}
	multipleToHash := int64(multiple*1000000.0 + 0.5)

	buffer.WriteString(fmt.Sprintf("%d", interestRateToHash))
	buffer.WriteByte(0x00)
	buffer.WriteString(fmt.Sprintf("%d", multipleToHash))
	buffer.WriteByte(0x00)
	//    bufferPtr+=1+sprintf(bufferPtr, "%lld", (long long)interestRateToHash);
	//    bufferPtr+=1+sprintf(bufferPtr, "%lld", (long long)multipleToHash);
	buffer.Write(contractContent)
	buffer.WriteByte(0x00)

	hash := sha256.Sum256(buffer.Bytes())
	return hash
}

// Calculates the messageHash for a CoinSpark message containing the given messageParts array.
// Pass in a random string in salt (length saltLen), that should be sent to the message server along with the content.
func CoinSparkCalcMessageHash(salt []byte, messageParts []CoinSparkMessagePart) []byte {
	buffer := bytes.Buffer{}
	buffer.Write(salt)
	buffer.WriteByte(0x00)
	for _, part := range messageParts {
		buffer.WriteString(part.MimeType)
		buffer.WriteByte(0x00)
		buffer.WriteString(part.FileName)
		buffer.WriteByte(0x00)
		buffer.Write(part.Content)
		buffer.WriteByte(0x00)
	}
	hash := sha256.Sum256(buffer.Bytes())
	return hash[:]
}

func MantissaExponentToQty(mantissa int16, exponent int16) int64 {
	var quantity int64
	quantity = int64(mantissa)
	for ; exponent > 0; exponent-- {
		quantity *= 10
	}
	return quantity
}

func QtyToMantissaExponent(quantity CoinSparkAssetQty, rounding int, mantissaMax int16, exponentMax int16) (qty int64, mantissa int16, exponent int16) {
	var roundOffset int64

	if rounding < 0 {
		roundOffset = 0
	} else if rounding > 0 {
		roundOffset = 9
	} else {
		roundOffset = 4
	}

	exponent = 0

	for quantity > CoinSparkAssetQty((mantissaMax)) {
		quantity = (quantity + CoinSparkAssetQty(roundOffset)) / 10
		exponent++
	}

	mantissa = int16(quantity)
	exponent = COINSPARK_MIN16(exponent, exponentMax)

	qty = MantissaExponentToQty(mantissa, exponent)
	return qty, mantissa, exponent
}

// Can't sort as index matters
/*
type ByLength []string

func (s ByLength) Len() int {
    return len(s)
}
func (s ByLength) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s ByLength) Less(i, j int) bool {
    return len(s[i]) < len(s[j])
}
//		sort.Sort(sort.Reverse(ByLength(slice)))
*/

//static size_t ShrinkLowerDomainName(const char* fullDomainName, size_t fullDomainNameLen, char* shortDomainName, size_t shortDomainNameMaxLen, char* packing)
func ShrinkLowerDomainName(fullDomainName string) (shortDomainName string, packing byte) {
	//char sourceDomainName[256];
	//int charIndex, bestPrefixLen, prefixIndex, prefixLen, bestPrefixIndex, bestSuffixLen, suffixIndex, suffixLen, bestSuffixIndex;
	//size_t sourceDomainLen;

	//  Check we have things in range

	//    if (len(fullDomainName>=sizeof(sourceDomainName)) // >= because of null terminator
	//        return 0;

	if len(fullDomainName) <= 0 {
		return "", 0 // nothing there
	}

	//  Convert to lower case and C-terminated string

	//    sourceDomainLen=fullDomainNameLen;

	//    for (charIndex=0; charIndex<sourceDomainLen; charIndex++)
	//        sourceDomainName[charIndex]=tolower(fullDomainName[charIndex]);

	//    sourceDomainName[sourceDomainLen]=0;

	sourceDomainName := strings.ToLower(fullDomainName)

	//  Search for prefixes
	//var bestPrefix string
	var bestPrefixLen int = -1
	var bestPrefixIndex int
	for n, prefix := range domainNamePrefixes {
		prefixLen := len(prefix)
		if prefixLen > bestPrefixLen && strings.HasPrefix(sourceDomainName, prefix) {
			bestPrefixLen = prefixLen
			bestPrefixIndex = n
		}
	}

	if bestPrefixLen > 0 {
		sourceDomainName = strings.TrimPrefix(sourceDomainName, domainNamePrefixes[bestPrefixIndex])
	}

	//  Search for suffixes
	var bestSuffixLen int = -1
	var bestSuffixIndex int

	// Optimisation: sort suffixes into descending order
	//	sortedSuffixes :=  domainNameSuffixes[:]
	//	sort.Sort(sort.Reverse(ByLength(sortedSuffixes)))

	for n, suffix := range domainNameSuffixes {
		suffixLen := len(suffix)
		if suffixLen > bestSuffixLen && strings.HasSuffix(sourceDomainName, suffix) {
			bestSuffixLen = suffixLen
			bestSuffixIndex = n
			//break // Optimsation: break since first suffix found is the longest
		}
	}

	if bestSuffixLen > 0 {
		sourceDomainName = strings.TrimSuffix(sourceDomainName, domainNameSuffixes[bestSuffixIndex])
	}

	//  Output and return

	shortDomainName = sourceDomainName

	//    strcpy(shortDomainName, sourceDomainName);
	packingInt := ((bestPrefixIndex << COINSPARK_DOMAIN_PACKING_PREFIX_SHIFT) & COINSPARK_DOMAIN_PACKING_PREFIX_MASK) |
		(bestSuffixIndex & COINSPARK_DOMAIN_PACKING_SUFFIX_MASK)

	packing = byte(packingInt)
	return shortDomainName, packing
}

func EncodeDomainPathTriplets(path string) []byte {
	metadata := bytes.Buffer{}
	stringLen := len(path)
	stringTriplet := 0
	lowerPath := strings.ToLower(path)

	for stringPos, char := range lowerPath {

		encodeValue := strings.Index(string(domainPathChars), string(char))
		if encodeValue == -1 {

			goto cannotEncodeTriplets // invalid character found
		}

		switch stringPos % 3 {
		case 0:
			stringTriplet = encodeValue
		case 1:
			stringTriplet += encodeValue * COINSPARK_DOMAIN_PATH_ENCODE_BASE
		case 2:
			stringTriplet += encodeValue * COINSPARK_DOMAIN_PATH_ENCODE_BASE * COINSPARK_DOMAIN_PATH_ENCODE_BASE
		}

		if ((stringPos % 3) == 2) || (stringPos == (stringLen - 1)) { // write out 2 bytes if we've collected 3 chars, or if we're finishing
			//      if ((metadataPtr+2)<=metadataEnd) {
			buf := make([]byte, 2)
			binary.LittleEndian.PutUint16(buf, uint16(stringTriplet))
			n, _ := metadata.Write(buf)

			if n != 2 {
				goto cannotEncodeTriplets
			}
		}
	}

	return metadata.Bytes()

cannotEncodeTriplets:
	return nil
}

// maybe return size
func EncodeDomainAndOrPath(domainName string, useHttps bool, pagePath string, usePrefix bool, forMessages bool) []byte {
	//static size_t EncodeDomainAndOrPath(const char* domainName, bool useHttps, const char* pagePath, bool usePrefix,
	//                                      char* _metadataPtr, const char* metadataEnd)
	//   size_t encodeStringLen, pagePathLen, encodeLen;
	//   char *metadataPtr, packing, encodeString[256];
	//   u_int8_t octets[4];
	//   var octets [4]byte

	//   metadataPtr=_metadataPtr;
	//   encodeStringLen=0;

	//  Domain name
	metadata := bytes.Buffer{}
	buffer := bytes.Buffer{}
	skipEmptyPagePath := false

	if domainName != "" {
		theIP := net.ParseIP(domainName)
		if theIP != nil {
			theIP = theIP.To4() // could return 16 byte slice
		}
		if theIP != nil && len(theIP) == 4 { // special space-saving encoding for IPv4 addresses
			var c byte
			if forMessages && pagePath == "" {
				c = COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4_NO_PATH
				if usePrefix {
					c |= COINSPARK_DOMAIN_PACKING_IPv4_NO_PATH_PREFIX
				}
				skipEmptyPagePath = true //	pagePath=None # skip encoding the empty page path
			} else {
				c = COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4
			}

			if useHttps {
				c |= COINSPARK_DOMAIN_PACKING_IPv4_HTTPS
			}

			metadata.WriteByte(c)
			metadata.Write(theIP)
		} else { // otherwise shrink the domain name and prepare it for encoding
			shortDomainName, packing := ShrinkLowerDomainName(domainName)

			if shortDomainName == "" {
				goto cannotEncodeDomainAndPath
			}
			buffer.WriteString(shortDomainName)
			if useHttps {
				buffer.WriteByte(COINSPARK_DOMAIN_PATH_TRUE_END_CHAR)
			} else {
				buffer.WriteByte(COINSPARK_DOMAIN_PATH_FALSE_END_CHAR)
			}

			metadata.WriteByte(packing)
		}
	}

	//  Page path

	if pagePath != "" || pagePath == "" && skipEmptyPagePath == false {
		if pagePath != "" {
			buffer.WriteString(pagePath)
		}
		if usePrefix {
			buffer.WriteByte(COINSPARK_DOMAIN_PATH_TRUE_END_CHAR)
		} else {
			buffer.WriteByte(COINSPARK_DOMAIN_PATH_FALSE_END_CHAR)
		}
	}

	//  Encode whatever is required as triplets

	if buffer.Len() > 0 {
		encoded := EncodeDomainPathTriplets(buffer.String())
		if encoded == nil {
			goto cannotEncodeDomainAndPath
		}
		metadata.Write(encoded)
	}

	return metadata.Bytes()

cannotEncodeDomainAndPath:
	return nil
}

// Go doesn't have a rounding function.
// https://gist.github.com/DavidVaini/10308388
// Public domain / open-source.i
// We want to replicate C math library function round
// --> round(0.5) is 1.0, and round(-0.5) is -1.0.
func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func WriteSmallEndianUnsigned(value int64, numBytes int) []byte {
	if value < 0 {
		return []byte{} //nil
	}
	if numBytes == 0 {
		return []byte{}
	}

	buf := bytes.Buffer{}

	for x := 0; x < numBytes; x++ {
		buf.WriteByte(byte(value % 256))
		value = int64(math.Floor(float64(value) / 256.0))
	}
	if value > 0 {
		return nil
	}
	return buf.Bytes()
}

func UnsignedToSmallEndianHex(value int64, numBytes int) string {
	buffer := bytes.Buffer{}
	if numBytes > 0 {
		for x := 0; x < numBytes; x++ {
			hexString := fmt.Sprintf("%02X", value%256)
			buffer.WriteString(hexString)
			value = int64(math.Floor(float64(value) / 256.0))
		}
	}
	return buffer.String()
}

func GetLastRegularOutput(outputsRegular []bool) int {
	var outputIndex int
	countOutputs := len(outputsRegular)
	for outputIndex = countOutputs - 1; outputIndex >= 0; outputIndex-- {
		if outputsRegular[outputIndex] {
			return outputIndex
		}
	}

	//return countOutputs // indicates no regular ones were found
	return -1 // indicate no regular ones were found
}

func CountNonLastRegularOutputs(outputsRegular []bool) int {
	var countRegularOutputs, outputIndex int

	countRegularOutputs = 0
	countOutputs := len(outputsRegular)

	for outputIndex = 0; outputIndex < countOutputs; outputIndex++ {
		if outputsRegular[outputIndex] {
			countRegularOutputs++
		}
	}

	return COINSPARK_MAX(countRegularOutputs-1, 0)
}

func (p *CoinSparkGenesis) Clear() {
	p.qtyMantissa = 0
	p.qtyExponent = 0
	p.chargeFlatMantissa = 0
	p.chargeFlatExponent = 0
	p.chargeBasisPoints = 0
	p.useHttps = false
	p.domainName = "" //[0]=0x00
	p.usePrefix = true
	p.pagePath = "" //[0]=0x00
	p.assetHash = nil
	p.assetHashLen = 0
}

func (p *CoinSparkGenesis) GetHashLen() int {
	return p.assetHashLen
}

func (p *CoinSparkGenesis) GetChargeFlat() CoinSparkAssetQty {
	x := MantissaExponentToQty(p.chargeFlatMantissa, p.chargeFlatExponent)
	return CoinSparkAssetQty(x)
}

func (p *CoinSparkGenesis) SetChargeFlat(desiredChargeFlat CoinSparkAssetQty, rounding int) CoinSparkAssetQty {
	var chargeFlatMantissa, chargeFlatExponent int16
	_, chargeFlatMantissa, chargeFlatExponent = QtyToMantissaExponent(desiredChargeFlat, rounding, COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX, COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MAX)
	if chargeFlatExponent == COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MAX {
		chargeFlatMantissa = COINSPARK_MIN16(chargeFlatMantissa, COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX_IF_EXP_MAX)
	}
	p.chargeFlatMantissa = chargeFlatMantissa
	p.chargeFlatExponent = chargeFlatExponent
	return p.GetChargeFlat()
}

func (p *CoinSparkGenesis) GetQty() CoinSparkAssetQty {
	x := MantissaExponentToQty(p.qtyMantissa, p.qtyExponent)
	return CoinSparkAssetQty(x)
}

func (p *CoinSparkGenesis) SetQty(desiredQty CoinSparkAssetQty, rounding int) CoinSparkAssetQty {
	_, qtyMantissa, qtyExponent := QtyToMantissaExponent(desiredQty, rounding, COINSPARK_GENESIS_QTY_MANTISSA_MAX, COINSPARK_GENESIS_QTY_EXPONENT_MAX)
	p.qtyMantissa = qtyMantissa
	p.qtyExponent = qtyExponent
	return p.GetQty()
}

func (p *CoinSparkGenesis) String() string {
	//char buffer[1024], hex[128], *bufferPtr;
	//size_t bufferLength, copyLength, domainPathEncodeLen;
	var quantityEncoded, chargeFlatEncoded int
	var quantity, chargeFlat CoinSparkAssetQty
	//    char domainPathMetadata[64];

	buffer := bytes.Buffer{}

	quantity = p.GetQty()
	quantityEncoded = int((p.qtyExponent*COINSPARK_GENESIS_QTY_EXPONENT_MULTIPLE + p.qtyMantissa) & COINSPARK_GENESIS_QTY_MASK)
	chargeFlat = p.GetChargeFlat()
	chargeFlatEncoded = int(p.chargeFlatExponent*COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MULTIPLE + p.chargeFlatMantissa)
	domainPathMetadata := EncodeDomainAndOrPath(p.domainName, p.useHttps, p.pagePath, p.usePrefix, false)

	buffer.WriteString("COINSPARK GENESIS\n")
	buffer.WriteString(fmt.Sprintf("   Quantity mantissa: %d\n", p.qtyMantissa))
	buffer.WriteString(fmt.Sprintf("   Quantity exponent: %d\n", p.qtyExponent))

	buffer.WriteString(fmt.Sprintf("    Quantity encoded: %d (small endian hex %s)\n", quantityEncoded, UnsignedToSmallEndianHex(int64(quantityEncoded), COINSPARK_GENESIS_QTY_FLAGS_LENGTH)))
	buffer.WriteString(fmt.Sprintf("      Quantity value: %d\n", quantity))
	buffer.WriteString(fmt.Sprintf("Flat charge mantissa: %d\n", p.chargeFlatMantissa))
	buffer.WriteString(fmt.Sprintf("Flat charge exponent: %d\n", p.chargeFlatExponent))
	buffer.WriteString(fmt.Sprintf(" Flat charge encoded: %d (small endian hex %s)\n", chargeFlatEncoded, UnsignedToSmallEndianHex(int64(chargeFlatEncoded), COINSPARK_GENESIS_CHARGE_FLAT_LENGTH)))
	buffer.WriteString(fmt.Sprintf("   Flat charge value: %d\n", chargeFlat))
	buffer.WriteString(fmt.Sprintf(" Basis points charge: %d (hex %s)\n", p.chargeBasisPoints, UnsignedToSmallEndianHex(int64(p.chargeBasisPoints), COINSPARK_GENESIS_CHARGE_BPS_LENGTH)))

	httpMode := "http"
	if p.useHttps {
		httpMode = "https"
	}
	prefix := ""
	if p.usePrefix {
		prefix = "coinspark/"
	}
	pagePath := "[spent-txid]"
	if len(p.pagePath) > 0 {
		pagePath = p.pagePath
	}
	buffer.WriteString(fmt.Sprintf("           Asset URL: %s://%s/%s%s/ (length %d+%d encoded %s length %d)\n",
		httpMode, p.domainName,
		prefix, pagePath,
		len(p.domainName), len(p.pagePath),
		strings.ToUpper(hex.EncodeToString(domainPathMetadata)),
		len(domainPathMetadata)))

	buffer.WriteString(fmt.Sprintf("          Asset hash: %s (length %d)\n", strings.ToUpper(hex.EncodeToString(p.assetHash[0:p.assetHashLen])), p.assetHashLen))
	buffer.WriteString("END COINSPARK GENESIS\n\n")

	return buffer.String()
}

func (p *CoinSparkGenesis) IsValid() bool {
	if (p.qtyMantissa < COINSPARK_GENESIS_QTY_MANTISSA_MIN) || (p.qtyMantissa > COINSPARK_GENESIS_QTY_MANTISSA_MAX) {
		return false
	}

	if (p.qtyExponent < COINSPARK_GENESIS_QTY_EXPONENT_MIN) || (p.qtyExponent > COINSPARK_GENESIS_QTY_EXPONENT_MAX) {
		return false
	}

	if (p.chargeFlatExponent < COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MIN) || (p.chargeFlatExponent > COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MAX) {
		return false
	}

	if p.chargeFlatMantissa < COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MIN {
		return false
	}

	var tmp int16
	if p.chargeFlatExponent == COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MAX {
		tmp = COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX_IF_EXP_MAX
	} else {
		tmp = COINSPARK_GENESIS_CHARGE_FLAT_MANTISSA_MAX
	}
	if p.chargeFlatMantissa > tmp {
		return false
	}

	if (p.chargeBasisPoints < COINSPARK_GENESIS_CHARGE_BASIS_POINTS_MIN) || (p.chargeBasisPoints > COINSPARK_GENESIS_CHARGE_BASIS_POINTS_MAX) {
		return false
	}

	if len(p.domainName) > COINSPARK_GENESIS_DOMAIN_NAME_MAX_LEN {
		return false
	}

	if len(p.pagePath) > COINSPARK_GENESIS_PAGE_PATH_MAX_LEN {
		return false
	}

	if (p.assetHashLen < COINSPARK_GENESIS_HASH_MIN_LEN) || (p.assetHashLen > COINSPARK_GENESIS_HASH_MAX_LEN) {
		return false
	}

	return true
}

func (p *CoinSparkGenesis) Match(other *CoinSparkGenesis, strict bool) bool {
	var hashCompareLen int
	var floatQuantitiesMatch bool

	hashCompareLen = COINSPARK_MIN(p.assetHashLen, other.assetHashLen)
	hashCompareLen = COINSPARK_MIN(hashCompareLen, COINSPARK_GENESIS_HASH_MAX_LEN)

	if strict {
		floatQuantitiesMatch = (p.qtyMantissa == other.qtyMantissa) && (p.qtyExponent == other.qtyExponent) && (p.chargeFlatMantissa == other.chargeFlatMantissa) && (p.chargeFlatExponent == other.chargeFlatExponent)
	} else {
		floatQuantitiesMatch = p.GetQty() == other.GetQty() && p.GetChargeFlat() == other.GetChargeFlat()
	}

	return (floatQuantitiesMatch && (p.chargeBasisPoints == other.chargeBasisPoints) && p.useHttps == other.useHttps && strings.ToLower(p.domainName) == strings.ToLower(other.domainName) && p.usePrefix == other.usePrefix && strings.ToLower(p.pagePath) == strings.ToLower(other.pagePath) && bytes.Equal(p.assetHash[0:hashCompareLen], other.assetHash[0:hashCompareLen]))

}

func (p *CoinSparkGenesis) Apply(outputsRegular []bool) []CoinSparkAssetQty {
	countOutputs := len(outputsRegular)
	outputBalances := make([]CoinSparkAssetQty, countOutputs)
	var qtyPerOutput CoinSparkAssetQty

	lastRegularOutput := GetLastRegularOutput(outputsRegular)
	divideOutputs := CountNonLastRegularOutputs(outputsRegular)
	genesisQty := p.GetQty()

	if divideOutputs == 0 {
		qtyPerOutput = 0
	} else {
		qtyPerOutput = genesisQty / CoinSparkAssetQty(divideOutputs) // rounds down
	}

	extraFirstOutput := genesisQty - (qtyPerOutput * CoinSparkAssetQty(divideOutputs))
	for outputIndex := 0; outputIndex < countOutputs; outputIndex++ {
		outputBalances[outputIndex] = 0
		if outputsRegular[outputIndex] && (outputIndex != lastRegularOutput) {
			outputBalances[outputIndex] = qtyPerOutput + extraFirstOutput
			extraFirstOutput = 0 // so it will only contribute to the first
		} else {
			outputBalances[outputIndex] = 0
		}
	}

	return outputBalances
}

// Calculates the URL for the asset web page of genesis.
// Returns empty string if fail
func (p *CoinSparkGenesis) CalcAssetURL(firstSpentTxID string, firstSpentVout int) string {
	var protocol string
	if p.useHttps {
		protocol = "https"
	} else {
		protocol = "http"
	}

	var prefix string
	if p.usePrefix {
		prefix = "coinspark/"
	} else {
		prefix = ""
	}

	// suffix uses path but if not valid uses 16 bytes from txid starting at pos firstSpentVout % 64, and
	// wrap around to front of string. like a circular buffer.
	var suffix string = p.pagePath
	if suffix == "" {
		if firstSpentTxID == "" || len(firstSpentTxID) != 64 {
			return ""
		}
		buffer := firstSpentTxID + firstSpentTxID
		startPos := firstSpentVout % 64
		suffix = buffer[startPos : startPos+16] // slice works on ASCII string which we expect
	}

	s := fmt.Sprintf("%s://%s/%s%s/", protocol, p.domainName, prefix, suffix)
	return s
}

func (p *CoinSparkGenesis) CalcCharge(qtyGross CoinSparkAssetQty) CoinSparkAssetQty {
	charge := p.GetChargeFlat() + ((qtyGross*CoinSparkAssetQty(p.chargeBasisPoints) + 5000) / 10000) // rounds to nearest

	return COINSPARK_MINASSETQTY(qtyGross, charge)
}

func (p *CoinSparkGenesis) CalcHashLen(metadataMaxLen int) int {

	assetHashLen := metadataMaxLen - COINSPARK_METADATA_IDENTIFIER_LEN - 1 - COINSPARK_GENESIS_QTY_FLAGS_LENGTH

	if p.chargeFlatMantissa > 0 {
		assetHashLen -= COINSPARK_GENESIS_CHARGE_FLAT_LENGTH
	}

	if p.chargeBasisPoints > 0 {
		assetHashLen -= COINSPARK_GENESIS_CHARGE_BPS_LENGTH
	}

	domainPathLen := len(p.pagePath) + 1
	theIP := net.ParseIP(p.domainName)
	if theIP != nil {
		theIP = theIP.To4() // could return 16 byte slice
	}
	if theIP != nil {
		assetHashLen -= 5 // packing and IP octets
	} else {
		assetHashLen -= 1 // packing
		shortDomainName, _ := ShrinkLowerDomainName(p.domainName)
		domainPathLen += len(shortDomainName) + 1
	}

	assetHashLen -= 2 * ((domainPathLen + 2) / 3) // uses integer arithmetic

	return COINSPARK_MIN(assetHashLen, COINSPARK_GENESIS_HASH_MAX_LEN)
}

func (p *CoinSparkGenesis) CalcMinFee(outputsSatoshis []CoinSparkSatoshiQty, outputsRegular []bool) CoinSparkSatoshiQty {
	return CoinSparkSatoshiQty(CountNonLastRegularOutputs(outputsRegular)) * GetMinFeeBasis(outputsSatoshis, outputsRegular)
}

func (p *CoinSparkGenesis) CalcNet(qtyGross CoinSparkAssetQty) CoinSparkAssetQty {
	return qtyGross - p.CalcCharge(qtyGross)
}

func (p *CoinSparkGenesis) CalcGross(qtyNet CoinSparkAssetQty) CoinSparkAssetQty {
	var lowerGross CoinSparkAssetQty

	if qtyNet <= 0 {
		return 0 // no point getting past charges if we end up with zero anyway
	}

	lowerGross = ((qtyNet + p.GetChargeFlat()) * 10000) / CoinSparkAssetQty(10000-p.chargeBasisPoints) // divides rounding down

	var result CoinSparkAssetQty
	if p.CalcNet(lowerGross) >= qtyNet {
		result = lowerGross
	} else {
		result = lowerGross + 1
	}
	return result
}

func (p *CoinSparkGenesis) Decode(buffer []byte) bool {
	metadata := LocateMetadataRange(buffer, COINSPARK_GENESIS_PREFIX)
	if metadata == nil {
		return false
	}

	// Quantity mantissa and exponent

	quantityEncoded := int(binary.LittleEndian.Uint16([]byte(metadata[:COINSPARK_GENESIS_QTY_FLAGS_LENGTH])))
	metadata = metadata[COINSPARK_GENESIS_QTY_FLAGS_LENGTH:]
	if quantityEncoded == 0 {
		return false
	}

	p.qtyMantissa = int16((quantityEncoded & COINSPARK_GENESIS_QTY_MASK) % COINSPARK_GENESIS_QTY_EXPONENT_MULTIPLE)
	p.qtyExponent = int16((quantityEncoded & COINSPARK_GENESIS_QTY_MASK) / COINSPARK_GENESIS_QTY_EXPONENT_MULTIPLE)

	// Charges - flat and basis points

	if quantityEncoded&COINSPARK_GENESIS_FLAG_CHARGE_FLAT > 0 {
		chargeEncoded := int(metadata[0])
		metadata = metadata[COINSPARK_GENESIS_CHARGE_FLAT_LENGTH:]

		p.chargeFlatMantissa = int16(chargeEncoded % COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MULTIPLE)
		p.chargeFlatExponent = int16(chargeEncoded / COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MULTIPLE)
	} else {
		p.chargeFlatMantissa = 0
		p.chargeFlatExponent = 0
	}

	if quantityEncoded&COINSPARK_GENESIS_FLAG_CHARGE_BPS > 0 {
		p.chargeBasisPoints = int16(metadata[0])
		metadata = metadata[COINSPARK_GENESIS_CHARGE_BPS_LENGTH:]
	} else {
		p.chargeBasisPoints = 0
	}

	//  Domain name and page path

	valid, result := DecodeDomainAndOrPath(string(metadata), true, true, false)
	if !valid {
		return false
	}

	metadata = metadata[result.decodedChars:]
	p.useHttps = result.useHttps
	p.domainName = result.domainName
	p.usePrefix = result.usePrefix
	p.pagePath = result.pagePath

	// Asset hash

	p.assetHashLen = COINSPARK_MIN(len(metadata), COINSPARK_GENESIS_HASH_MAX_LEN)
	p.assetHash = metadata[:p.assetHashLen]

	// Return validity

	return p.IsValid()
}

func (p *CoinSparkGenesis) Encode(metadataMaxLen int) (err error, metadata []byte) {
	metadata = nil

	if !p.IsValid() {
		return errors.New("invalid genesis"), metadata
	}

	buf := new(bytes.Buffer)
	buf.WriteString(COINSPARK_METADATA_IDENTIFIER)
	buf.WriteByte(COINSPARK_GENESIS_PREFIX)

	//  Quantity mantissa and exponent

	var quantityEncoded int = int((p.qtyExponent*COINSPARK_GENESIS_QTY_EXPONENT_MULTIPLE + p.qtyMantissa) & COINSPARK_GENESIS_QTY_MASK)
	if p.chargeFlatMantissa > 0 {
		quantityEncoded |= COINSPARK_GENESIS_FLAG_CHARGE_FLAT
	}
	if p.chargeBasisPoints > 0 {
		quantityEncoded |= COINSPARK_GENESIS_FLAG_CHARGE_BPS
	}

	// COINSPARK_GENESIS_QTY_FLAGS_LENGTH = 2
	err = binary.Write(buf, binary.LittleEndian, uint16(quantityEncoded))
	if err != nil {
		return err, metadata
	}

	//  Charges - flat and basis points

	if (quantityEncoded & COINSPARK_GENESIS_FLAG_CHARGE_FLAT) != 0 {
		chargeEncoded := p.chargeFlatExponent*COINSPARK_GENESIS_CHARGE_FLAT_EXPONENT_MULTIPLE + p.chargeFlatMantissa

		// COINSPARK_GENESIS_CHARGE_FLAT_LENGTH = 1
		buf.WriteByte(uint8(chargeEncoded))
	}

	// COINSPARK_GENESIS_CHARGE_BPS_LENGTH = 1
	if (quantityEncoded & COINSPARK_GENESIS_FLAG_CHARGE_BPS) != 0 {
		buf.WriteByte(uint8(p.chargeBasisPoints))
	}

	// Domain name and page path

	domainBuf := EncodeDomainAndOrPath(p.domainName, p.useHttps, p.pagePath, p.usePrefix, false)
	if domainBuf == nil {
		return errors.New("cannot write domain name/path"), metadata
	}

	buf.Write(domainBuf)

	// Asset hash
	buf.Write(p.assetHash[:p.assetHashLen])

	// Check the total length is within the specified limit

	if buf.Len() > metadataMaxLen {
		return errors.New("total length above limit"), metadata
	}

	// Return what we created
	metadata = buf.Bytes()
	return nil, metadata
}

type result_DecodeDomainAndOrPath struct {
	decodedChars int
	useHttps     bool
	domainName   string
	pagePath     string
	usePrefix    bool
}

func DecodeDomainAndOrPath(metadata string, doDomainName bool, doPagePath bool, forMessages bool) (bool, result_DecodeDomainAndOrPath) {
	startLength := len(metadata)
	metadataParts := 0
	result := result_DecodeDomainAndOrPath{}
	var isIpAddress bool = false
	var packing int
	// Domain name

	if doDomainName {

		// Get packing byte
		if len(metadata) < 1 {
			return false, result
		}

		packingChar := metadata[0]
		metadata = metadata[1:]

		packing = int(packingChar)

		//# Extract IP address if present

		packingSuffix := packing & COINSPARK_DOMAIN_PACKING_SUFFIX_MASK
		isIpAddress = ((packingSuffix == COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4) ||
			(forMessages && (packingSuffix == COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4_NO_PATH)))

		if isIpAddress {
			result.useHttps = (packing & COINSPARK_DOMAIN_PACKING_IPv4_HTTPS) > 0
			if len(metadata) <= 4 {
				return false, result
			}

			octetChars := metadata[:4]
			metadata = metadata[4:]
			result.domainName = fmt.Sprintf("%d.%d.%d.%d", int(octetChars[0]), int(octetChars[1]), int(octetChars[2]), int(octetChars[3]))

			if doPagePath && forMessages && packingSuffix == COINSPARK_DOMAIN_PACKING_SUFFIX_IPv4_NO_PATH {
				result.pagePath = ""
				result.usePrefix = (packing & COINSPARK_DOMAIN_PACKING_IPv4_NO_PATH_PREFIX) > 0

				doPagePath = false // skip decoding the empty page path
			}
		} else {
			metadataParts += 1
		}
	}

	// Convert remaining metadata to string

	if doPagePath {
		metadataParts += 1
	}

	if metadataParts > 0 {
		decodeString, decodedCharsPos := DecodeDomainPathTriplets(metadata, metadataParts)
		if decodeString == "" {
			return false, result
		}

		metadata = metadata[decodedCharsPos:]

		// Extract domain name if IP address was not present
		if doDomainName && !isIpAddress {
			replacedDecodeString := strings.Replace(decodeString, string(COINSPARK_DOMAIN_PATH_FALSE_END_CHAR), string(COINSPARK_DOMAIN_PATH_TRUE_END_CHAR), -1)
			endCharPos := strings.IndexRune(replacedDecodeString, COINSPARK_DOMAIN_PATH_TRUE_END_CHAR)
			if endCharPos < 0 {
				return false, result // should never happen
			}
			result.domainName = ExpandDomainName(decodeString[0:endCharPos], packing)
			if result.domainName == "" {
				return false, result
			}
			result.useHttps = decodeString[endCharPos] == COINSPARK_DOMAIN_PATH_TRUE_END_CHAR
			decodeString = decodeString[endCharPos+1:]
		}

		// Extract page path

		if doPagePath {
			replacedDecodeString := strings.Replace(decodeString, string(COINSPARK_DOMAIN_PATH_FALSE_END_CHAR), string(COINSPARK_DOMAIN_PATH_TRUE_END_CHAR), 1)
			endCharPos := strings.IndexRune(replacedDecodeString, COINSPARK_DOMAIN_PATH_TRUE_END_CHAR)

			if endCharPos < 0 {
				return false, result // should never happen
			}

			result.usePrefix = (decodeString[endCharPos] == COINSPARK_DOMAIN_PATH_TRUE_END_CHAR)
			result.pagePath = decodeString[0:endCharPos]
			decodeString = decodeString[endCharPos+1:]
		}
	}
	// Finish and return
	result.decodedChars = startLength - len(metadata)

	return true, result
}

func DecodeDomainPathTriplets(metadata string, parts int) (result string, numDecodedChars int) {
	startLength := len(metadata)
	result = ""
	stringPos := 0
	var stringTriplet int
	for parts > 0 {

		if (stringPos % 3) == 0 {
			if len(metadata) < 2 {
				return "", 0
			}

			stringTriplet = int(binary.LittleEndian.Uint16([]byte(metadata[:2])))
			metadata = metadata[2:]

			if stringTriplet >= (COINSPARK_DOMAIN_PATH_ENCODE_BASE * COINSPARK_DOMAIN_PATH_ENCODE_BASE * COINSPARK_DOMAIN_PATH_ENCODE_BASE) {
				return "", 0 //invalid value
			}
		}

		stringPosMod3 := stringPos % 3

		var decodeValue int

		switch stringPosMod3 {
		case 0:
			decodeValue = stringTriplet % COINSPARK_DOMAIN_PATH_ENCODE_BASE
		case 1:
			decodeValue = int(math.Floor(float64(stringTriplet)/float64(COINSPARK_DOMAIN_PATH_ENCODE_BASE))) % COINSPARK_DOMAIN_PATH_ENCODE_BASE
		case 2:
			decodeValue = int(math.Floor(float64(stringTriplet) / float64(COINSPARK_DOMAIN_PATH_ENCODE_BASE*COINSPARK_DOMAIN_PATH_ENCODE_BASE)))
		}

		var decodeChar byte = domainPathChars[decodeValue]
		result = result + string(decodeChar)
		stringPos += 1

		if string(decodeChar) == string(COINSPARK_DOMAIN_PATH_TRUE_END_CHAR) || string(decodeChar) == string(COINSPARK_DOMAIN_PATH_FALSE_END_CHAR) {
			parts -= 1
		}
	}

	return result, startLength - len(metadata)
}

func ExpandDomainName(domainName string, packing int) string {

	// Prefix

	prefixIndex := (packing & COINSPARK_DOMAIN_PACKING_PREFIX_MASK) >> COINSPARK_DOMAIN_PACKING_PREFIX_SHIFT
	if prefixIndex >= len(domainNamePrefixes) {
		return ""
	}

	prefix := domainNamePrefixes[prefixIndex]

	// Suffix

	suffixIndex := packing & COINSPARK_DOMAIN_PACKING_SUFFIX_MASK
	if suffixIndex >= len(domainNameSuffixes) {
		return ""
	}

	suffix := domainNameSuffixes[suffixIndex]

	return prefix + domainName + suffix
}

func GetMinFeeBasis(outputsSatoshis []CoinSparkSatoshiQty, outputsRegular []bool) CoinSparkSatoshiQty {
	var smallestOutputSatoshis CoinSparkSatoshiQty
	var outputIndex int
	countOutputs := len(outputsRegular)

	smallestOutputSatoshis = COINSPARK_SATOSHI_QTY_MAX

	for outputIndex = 0; outputIndex < countOutputs; outputIndex++ {
		if outputsRegular[outputIndex] == true {
			smallestOutputSatoshis = COINSPARK_MINSATOSHIQTY(smallestOutputSatoshis, outputsSatoshis[outputIndex])
		}
	}

	return COINSPARK_MINSATOSHIQTY(COINSPARK_FEE_BASIS_MAX_SATOSHIS, smallestOutputSatoshis)
}

func LocateMetadataRange(metadata []byte, desiredPrefix byte) []byte {
	metadataLen := len(metadata)

	if metadataLen < (COINSPARK_METADATA_IDENTIFIER_LEN + 1) {
		// check for 4 bytes at least
		return nil
	}

	if string(metadata[0:COINSPARK_METADATA_IDENTIFIER_LEN]) != COINSPARK_METADATA_IDENTIFIER {
		// check it starts 'SPK'
		return nil
	}

	position := COINSPARK_METADATA_IDENTIFIER_LEN // start after 'SPK'

	for position < metadataLen {
		foundPrefix := metadata[position] // read the next prefix

		position += 1
		foundPrefixOrd := int(foundPrefix)

		if (desiredPrefix != 0 && foundPrefix == desiredPrefix) ||
			(desiredPrefix == COINSPARK_DUMMY_PREFIX && foundPrefixOrd > COINSPARK_LENGTH_PREFIX_MAX) {
			// it's our data from here to the end (if desiredPrefix is None, it matches the last one whichever it is)
			return metadata[position:]
		}

		if foundPrefixOrd > COINSPARK_LENGTH_PREFIX_MAX {
			// it's some other type of data from here to end
			return nil
		}

		// if we get here it means we found a length byte

		if position+foundPrefixOrd > metadataLen {
			// something went wrong - length indicated is longer than that available
			return nil
		}

		if position >= metadataLen {
			// something went wrong - that was the end of the input data
			return nil
		}

		if metadata[position] == desiredPrefix {
			// it's the length of our part
			return metadata[position+1 : position+foundPrefixOrd]
		} else {
			position += foundPrefixOrd // skip over this many bytes
		}
	}
	return nil
}

func (p *CoinSparkPaymentRef) Clear() *CoinSparkPaymentRef {
	p.ref = 0
	return p
}

func (p *CoinSparkPaymentRef) String() string {
	buffer := bytes.Buffer{}
	buffer.WriteString("COINSPARK PAYMENT REFERENCE\n")
	buffer.WriteString(fmt.Sprintf("%d (small endian hex %s)\n", p.ref, UnsignedToSmallEndianHex(int64(p.ref), 8)))
	buffer.WriteString("END COINSPARK PAYMENT REFERENCE\n\n")

	return buffer.String()
}

func (p *CoinSparkPaymentRef) IsValid() bool {
	return p.ref >= 0 && p.ref <= COINSPARK_PAYMENT_REF_MAX
}

func (p *CoinSparkPaymentRef) Match(other *CoinSparkPaymentRef) bool {
	return p.ref == other.ref
}

func (p *CoinSparkPaymentRef) Randomize() *CoinSparkPaymentRef {
	return NewRandomCoinSparkPaymentRef()
}

func NewRandomCoinSparkPaymentRef() *CoinSparkPaymentRef {
	rand.Seed(time.Now().UnixNano())
	return &CoinSparkPaymentRef{uint64(rand.Int63n(COINSPARK_PAYMENT_REF_MAX))}
}

func (p *CoinSparkPaymentRef) Encode(metadataMaxLen int) []byte {
	if !p.IsValid() {
		return nil
	}

	// 4-character identifier
	buf := bytes.Buffer{}
	buf.WriteString(COINSPARK_METADATA_IDENTIFIER)
	buf.WriteByte(COINSPARK_PAYMENTREF_PREFIX)

	// The payment reference

	bytes := 0
	paymentLeft := p.ref
	for paymentLeft > 0 {
		bytes += 1
		paymentLeft = uint64(math.Floor(float64(paymentLeft) / 256))
	}

	s := UnsignedToSmallEndianHex(int64(p.ref), bytes)
	hexBytes, _ := hex.DecodeString(s)
	buf.Write(hexBytes)

	// Check the total length is within the specified limit
	if buf.Len() > metadataMaxLen {
		return nil
	}

	// Return what we created
	return buf.Bytes()

}

func (p *CoinSparkPaymentRef) Decode(buffer []byte) bool {
	metadata := LocateMetadataRange(buffer, COINSPARK_PAYMENTREF_PREFIX)
	if metadata == nil {
		return false
	}

	// The payment reference

	finalMetadataLen := len(metadata)
	if finalMetadataLen > 8 {
		return false
	}

	_, v := ShiftLittleEndianBytesToInt(&metadata, finalMetadataLen)
	p.ref = uint64(v)

	// Return validity
	return p.IsValid()
}

func (p *CoinSparkIORange) Clear() {
	p.count = 0
	p.first = 0
}

func (p *CoinSparkIORange) IsValid() bool {
	if (p.first < 0) || (p.first > COINSPARK_IO_INDEX_MAX) {
		return false
	}
	if (p.count < 0) || (p.count > COINSPARK_IO_INDEX_MAX) {
		return false
	}
	return true
}

func NewCoinSparkIORange() *CoinSparkIORange {
	p := new(CoinSparkIORange)
	p.Clear()
	return p
}

func (p *CoinSparkIORange) Match(other *CoinSparkIORange) bool {
	return p.first == other.first && p.count == other.count
}

func (p *CoinSparkTransfer) Clear() {
	p.assetRef = CoinSparkAssetRef{}
	p.inputs = CoinSparkIORange{}
	p.outputs = CoinSparkIORange{}
	p.qtyPerOutput = CoinSparkAssetQty(0)
}

func (p *CoinSparkTransfer) IsValid() bool {
	if !(p.assetRef.IsValid() && p.inputs.IsValid() && p.outputs.IsValid()) {
		return false
	}
	if p.qtyPerOutput < 0 || p.qtyPerOutput > COINSPARK_ASSET_QTY_MAX {
		return false
	}
	return true
}

func (p *CoinSparkTransfer) Match(other *CoinSparkTransfer) bool {
	if p.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE {
		return other.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE && p.inputs.Match(&other.inputs) && p.outputs.first == other.outputs.first
	}
	return p.assetRef.Match(&other.assetRef) && p.inputs.Match(&other.inputs) && p.outputs.Match(&other.outputs) && p.qtyPerOutput == other.qtyPerOutput
}

func DecodePackingExtend(packingExtend byte, forMessages bool) (bool, string) {

	for _, packingType := range packingExtendMapOrder {
		packingExtendTest := packingExtendMap[packingType]
		if packingExtend == packingExtendTest {
			packingTypeTest := "_0_1_BYTE"
			if forMessages == true {
				packingTypeTest = "_1S"
			}

			if packingType != packingTypeTest {
				return true, packingType
			}
		}
	}

	return false, ""
}

func PackingTypeToValues(packingType string, previousRange *CoinSparkIORange, countInputOutputs int) CoinSparkIORange {
	var r CoinSparkIORange

	if packingType == "_0P" {
		if previousRange != nil {
			r.first = previousRange.first
			r.count = previousRange.count
		} else {
			r.first = 0
			r.count = 1
		}
	} else if packingType == "_1S" {
		if previousRange != nil {
			r.first = previousRange.first + previousRange.count
		} else {
			r.first = 1
		}
		r.count = 1
	} else if packingType == "_0_1_BYTE" {
		r.first = 0
	} else if (packingType == "_1_0_BYTE") || (packingType == "_2_0_BYTES") {
		r.count = 1
	} else if packingType == "_ALL" {
		r.first = 0
		r.count = CoinSparkIOIndex(countInputOutputs)
	}
	return r
}

func (p *CoinSparkTransfer) Decode(metadata []byte, previousTransfer *CoinSparkTransfer, countInputs int, countOutputs int) int {

	startLength := len(metadata)

	// Extract packing
	packing := int(metadata[0])

	metadata = metadata[1:]

	var inputPackingType, outputPackingType string
	var success bool
	packingExtend := 0

	// Packing for genesis reference

	if (packing & COINSPARK_PACKING_GENESIS_MASK) == COINSPARK_PACKING_GENESIS_PREV {
		if previousTransfer != nil {
			p.assetRef = previousTransfer.assetRef
		} else {
			// it's for a default route
			p.assetRef.blockNum = COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE
			p.assetRef.txOffset = 0
			var emptyPrefix [COINSPARK_ASSETREF_TXID_PREFIX_LEN]byte
			p.assetRef.txIDPrefix = emptyPrefix
		}
	}

	// Packing for input and output indices

	if (packing & COINSPARK_PACKING_INDICES_MASK) == COINSPARK_PACKING_INDICES_EXTEND {
		// we're using second packing metadata byte
		packingExtend = int(metadata[0])
		metadata = metadata[1:]
		if packingExtend == 0 {
			return 0
		}

		success, inputPackingType = DecodePackingExtend(byte((packingExtend>>COINSPARK_PACKING_EXTEND_INPUTS_SHIFT)&COINSPARK_PACKING_EXTEND_MASK), false)
		if success == false {
			return 0
		}
		success, outputPackingType = DecodePackingExtend(byte((packingExtend>>COINSPARK_PACKING_EXTEND_OUTPUTS_SHIFT)&COINSPARK_PACKING_EXTEND_MASK), false)
		if success == false {
			return 0
		}
	} else {
		// not using second packing metadata byte

		packingIndices := packing & COINSPARK_PACKING_INDICES_MASK

		// input packing
		if (packingIndices == COINSPARK_PACKING_INDICES_0P_0P) ||
			(packingIndices == COINSPARK_PACKING_INDICES_0P_1S) ||
			(packingIndices == COINSPARK_PACKING_INDICES_0P_ALL) {
			inputPackingType = "_0P"
		} else if packingIndices == COINSPARK_PACKING_INDICES_1S_0P {
			inputPackingType = "_1S"
		} else if (packingIndices == COINSPARK_PACKING_INDICES_ALL_0P) ||
			(packingIndices == COINSPARK_PACKING_INDICES_ALL_1S) ||
			(packingIndices == COINSPARK_PACKING_INDICES_ALL_ALL) {
			inputPackingType = "_ALL"
		}

		// output packing

		if (packingIndices == COINSPARK_PACKING_INDICES_0P_0P) ||
			(packingIndices == COINSPARK_PACKING_INDICES_1S_0P) ||
			(packingIndices == COINSPARK_PACKING_INDICES_ALL_0P) {
			outputPackingType = "_0P"
		} else if (packingIndices == COINSPARK_PACKING_INDICES_0P_1S) ||
			(packingIndices == COINSPARK_PACKING_INDICES_ALL_1S) {
			outputPackingType = "_1S"
		} else if (packingIndices == COINSPARK_PACKING_INDICES_0P_ALL) ||
			(packingIndices == COINSPARK_PACKING_INDICES_ALL_ALL) {
			outputPackingType = "_ALL"
		}
	}

	// Final stage of packing for input and output indices
	var prevInputs, prevOutputs *CoinSparkIORange
	if previousTransfer != nil {
		prevInputs = &previousTransfer.inputs
		prevOutputs = &previousTransfer.outputs
	}

	p.inputs = PackingTypeToValues(inputPackingType, prevInputs, countInputs)
	p.outputs = PackingTypeToValues(outputPackingType, prevOutputs, countOutputs)

	// Read in the fields as appropriate

	counts := p.PackingToByteCounts(byte(packing), byte(packingExtend))

	// copy metadata slice where it can be modified
	metadataArray := make([]byte, len(metadata))
	copy(metadataArray, metadata)
	//metadataArray=[metadataChar for metadataChar in metadata] # split into array of characters for next bit

	var result int
	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.blockNumBytes)
	if !success {
		return 0
	} else if counts.blockNumBytes > 0 {
		p.assetRef.blockNum = int64(result)
	}

	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.txOffsetBytes)
	if !success {
		return 0
	} else if counts.txOffsetBytes > 0 {
		p.assetRef.txOffset = int64(result)
	}

	txIDPrefixBytes := counts.txIDPrefixBytes
	if txIDPrefixBytes > 0 {
		if len(metadataArray) < txIDPrefixBytes {
			return 0
		}
		var prefix [COINSPARK_ASSETREF_TXID_PREFIX_LEN]byte
		copy(prefix[:], metadataArray[:txIDPrefixBytes])
		p.assetRef.txIDPrefix = prefix
		metadataArray = metadataArray[txIDPrefixBytes:]
	}
	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.firstInputBytes)
	if !success {
		return 0
	} else if counts.firstInputBytes > 0 {
		p.inputs.first = CoinSparkIOIndex(result)
	}

	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.countInputsBytes)
	if !success {
		return 0
	} else if counts.countInputsBytes > 0 {
		p.inputs.count = CoinSparkIOIndex(result)
	}

	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.firstOutputBytes)
	if !success {
		return 0
	} else if counts.firstOutputBytes > 0 {
		p.outputs.first = CoinSparkIOIndex(result)
	}

	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.countOutputsBytes)
	if !success {
		return 0
	} else if counts.countOutputsBytes > 0 {
		p.outputs.count = CoinSparkIOIndex(result)
	}

	success, result = ShiftLittleEndianBytesToInt(&metadataArray, counts.quantityBytes)
	if !success {
		return 0
	} else if counts.quantityBytes > 0 {
		p.qtyPerOutput = CoinSparkAssetQty(result)
	}

	metadata = metadataArray // use remaining characters

	// Finish up reading in quantity

	packingQuantity := packing & COINSPARK_PACKING_QUANTITY_MASK

	if packingQuantity == COINSPARK_PACKING_QUANTITY_1P {
		if previousTransfer != nil {
			p.qtyPerOutput = previousTransfer.qtyPerOutput
		} else {
			p.qtyPerOutput = 1
		}
	} else if packingQuantity == COINSPARK_PACKING_QUANTITY_MAX {
		p.qtyPerOutput = COINSPARK_ASSET_QTY_MAX
	} else if packingQuantity == COINSPARK_PACKING_QUANTITY_FLOAT {
		decodeQuantity := p.qtyPerOutput & COINSPARK_TRANSFER_QTY_FLOAT_MASK
		p.qtyPerOutput = CoinSparkAssetQty(MantissaExponentToQty(int16(decodeQuantity%COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MULTIPLE),
			int16(math.Floor(float64(decodeQuantity)/float64(COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MULTIPLE)))))
	}

	// Return bytes used

	if p.IsValid() == false {
		return 0
	}

	return startLength - len(metadata)
}

func (p *CoinSparkTransfer) Encode(previousTransfer *CoinSparkTransfer, metadataMaxLen int, countInputs int, countOutputs int) []byte {
	if p.IsValid() == false {
		return nil
	}

	var packing, packingExtend byte
	isDefaultRoute := (p.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE)

	// Packing for genesis reference

	if isDefaultRoute {
		if previousTransfer != nil && (previousTransfer.assetRef.blockNum != COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE) {
			return nil // default route transfers have to come at the start
		}

		packing |= COINSPARK_PACKING_GENESIS_PREV

	} else {
		if previousTransfer != nil && p.assetRef.Match(&previousTransfer.assetRef) {
			packing |= COINSPARK_PACKING_GENESIS_PREV
		} else if p.assetRef.blockNum <= COINSPARK_UNSIGNED_3_BYTES_MAX {
			if p.assetRef.txOffset <= COINSPARK_UNSIGNED_3_BYTES_MAX {
				packing |= COINSPARK_PACKING_GENESIS_3_3_BYTES
			} else if p.assetRef.txOffset <= COINSPARK_UNSIGNED_4_BYTES_MAX {
				packing |= COINSPARK_PACKING_GENESIS_3_4_BYTES
			} else {
				return nil
			}
		} else if (p.assetRef.blockNum <= COINSPARK_UNSIGNED_4_BYTES_MAX) && (p.assetRef.txOffset <= COINSPARK_UNSIGNED_4_BYTES_MAX) {
			packing |= COINSPARK_PACKING_GENESIS_4_4_BYTES
		} else {
			return nil
		}
	}

	// Packing for input and output indices
	inputPackingOptions := map[string]bool{}
	outputPackingOptions := map[string]bool{}
	if previousTransfer != nil {
		inputPackingOptions = GetPackingOptions(&previousTransfer.inputs, &p.inputs, countInputs, false)
		outputPackingOptions = GetPackingOptions(&previousTransfer.outputs, &p.outputs, countOutputs, false)
	} else {
		inputPackingOptions = GetPackingOptions(nil, &p.inputs, countInputs, false)
		outputPackingOptions = GetPackingOptions(nil, &p.outputs, countOutputs, false)
	}

	if inputPackingOptions["_0P"] && outputPackingOptions["_0P"] {
		packing |= COINSPARK_PACKING_INDICES_0P_0P
	} else if inputPackingOptions["_0P"] && outputPackingOptions["_1S"] {
		packing |= COINSPARK_PACKING_INDICES_0P_1S
	} else if inputPackingOptions["_0P"] && outputPackingOptions["_ALL"] {
		packing |= COINSPARK_PACKING_INDICES_0P_ALL
	} else if inputPackingOptions["_1S"] && outputPackingOptions["_0P"] {
		packing |= COINSPARK_PACKING_INDICES_1S_0P
	} else if inputPackingOptions["_ALL"] && outputPackingOptions["_0P"] {
		packing |= COINSPARK_PACKING_INDICES_ALL_0P
	} else if inputPackingOptions["_ALL"] && outputPackingOptions["_1S"] {
		packing |= COINSPARK_PACKING_INDICES_ALL_1S
	} else if inputPackingOptions["_ALL"] && outputPackingOptions["_ALL"] {
		packing |= COINSPARK_PACKING_INDICES_ALL_ALL
	} else {
		//: # we need the second (extended) packing byte
		packing |= COINSPARK_PACKING_INDICES_EXTEND

		success, packingExtendInput := EncodePackingExtend(inputPackingOptions)
		if !success {
			return nil
		}
		success, packingExtendOutput := EncodePackingExtend(outputPackingOptions)
		if !success {
			return nil
		}

		packingExtend = (packingExtendInput << COINSPARK_PACKING_EXTEND_INPUTS_SHIFT) | (packingExtendOutput << COINSPARK_PACKING_EXTEND_OUTPUTS_SHIFT)
	}

	// Packing for quantity

	encodeQuantity := p.qtyPerOutput

	if (previousTransfer != nil && p.qtyPerOutput == previousTransfer.qtyPerOutput) ||
		previousTransfer == nil && p.qtyPerOutput == 1 {
		packing |= COINSPARK_PACKING_QUANTITY_1P
	} else if p.qtyPerOutput >= COINSPARK_ASSET_QTY_MAX {
		packing |= COINSPARK_PACKING_QUANTITY_MAX
	} else if p.qtyPerOutput <= COINSPARK_UNSIGNED_BYTE_MAX {
		packing |= COINSPARK_PACKING_QUANTITY_1_BYTE
	} else if p.qtyPerOutput <= COINSPARK_UNSIGNED_2_BYTES_MAX {
		packing |= COINSPARK_PACKING_QUANTITY_2_BYTES
	} else {
		quantityPerOutput, mantissa, exponent := QtyToMantissaExponent(p.qtyPerOutput, 0,
			COINSPARK_TRANSFER_QTY_FLOAT_MANTISSA_MAX, COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MAX)
		if CoinSparkAssetQty(quantityPerOutput) == p.qtyPerOutput {
			packing |= COINSPARK_PACKING_QUANTITY_FLOAT
			encodeQuantity = CoinSparkAssetQty((exponent*COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MULTIPLE + mantissa) & COINSPARK_TRANSFER_QTY_FLOAT_MASK)
		} else if p.qtyPerOutput <= COINSPARK_UNSIGNED_3_BYTES_MAX {
			packing |= COINSPARK_PACKING_QUANTITY_3_BYTES
		} else if p.qtyPerOutput <= COINSPARK_UNSIGNED_4_BYTES_MAX {
			packing |= COINSPARK_PACKING_QUANTITY_4_BYTES
		} else {
			packing |= COINSPARK_PACKING_QUANTITY_6_BYTES
		}
	}

	// Write out the actual data

	counts := p.PackingToByteCounts(packing, packingExtend)

	buf := bytes.Buffer{}
	buf.WriteByte(packing)

	if (packing & COINSPARK_PACKING_INDICES_MASK) == COINSPARK_PACKING_INDICES_EXTEND {
		buf.WriteByte(packingExtend)
	}

	valbuf := WriteSmallEndianUnsigned(p.assetRef.blockNum, counts.blockNumBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	valbuf = WriteSmallEndianUnsigned(p.assetRef.txOffset, counts.txOffsetBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	buf.Write(p.assetRef.txIDPrefix[:counts.txIDPrefixBytes])
	padding := counts.txIDPrefixBytes - len(p.assetRef.txIDPrefix) // ensure right length
	for i := 0; i < padding; i++ {
		buf.WriteByte(0x00)
	}

	valbuf = WriteSmallEndianUnsigned(int64(p.inputs.first), counts.firstInputBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	valbuf = WriteSmallEndianUnsigned(int64(p.inputs.count), counts.countInputsBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	valbuf = WriteSmallEndianUnsigned(int64(p.outputs.first), counts.firstOutputBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	valbuf = WriteSmallEndianUnsigned(int64(p.outputs.count), counts.countOutputsBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	valbuf = WriteSmallEndianUnsigned(int64(encodeQuantity), counts.quantityBytes)
	if valbuf == nil {
		return nil
	}
	if len(valbuf) > 0 {
		buf.Write(valbuf)
	}

	//# Check the total length is within the specified limit

	if buf.Len() > metadataMaxLen {
		return nil
	}

	// Return what we created

	return buf.Bytes()

}

// result of -1 means ignore it
func ShiftLittleEndianBytesToInt(metadataPtr *[]byte, count int) (bool, int) {
	metadata := *metadataPtr

	if count > len(metadata) {
		return false, 0
	}

	var result int

	if count == 1 {
		result = int(metadata[0])
	} else if count == 2 {
		x := binary.LittleEndian.Uint16(metadata[0:count])
		result = int(x)
	} else if count == 4 {
		x := binary.LittleEndian.Uint32(metadata[0:count])
		result = int(x)
	} else {
		var sum uint64
		for i := 0; i < count; i++ {
			n := uint64(metadata[i])
			for p := 0; p < i; p++ {
				n *= 256
			}
			sum = sum + n
		}
		result = int(sum)
	}

	*metadataPtr = metadata[count:]
	return true, result
}

func (p *CoinSparkTransfer) PackingToByteCounts(packing byte, packingExtend byte) PackingByteCounts {
	var counts PackingByteCounts

	// Packing for genesis reference

	packingGenesis := packing & COINSPARK_PACKING_GENESIS_MASK

	if packingGenesis == COINSPARK_PACKING_GENESIS_3_3_BYTES {
		counts.blockNumBytes = 3
		counts.txOffsetBytes = 3
		counts.txIDPrefixBytes = COINSPARK_ASSETREF_TXID_PREFIX_LEN
	} else if packingGenesis == COINSPARK_PACKING_GENESIS_3_4_BYTES {
		counts.blockNumBytes = 3
		counts.txOffsetBytes = 4
		counts.txIDPrefixBytes = COINSPARK_ASSETREF_TXID_PREFIX_LEN
	} else if packingGenesis == COINSPARK_PACKING_GENESIS_4_4_BYTES {
		counts.blockNumBytes = 4
		counts.txOffsetBytes = 4
		counts.txIDPrefixBytes = COINSPARK_ASSETREF_TXID_PREFIX_LEN
	}

	// Packing for input and output indices (relevant for extended indices only)

	if (packing & COINSPARK_PACKING_INDICES_MASK) == COINSPARK_PACKING_INDICES_EXTEND {
		counts.firstInputBytes, counts.countInputsBytes = PackingExtendAddByteCounts((packingExtend>>COINSPARK_PACKING_EXTEND_INPUTS_SHIFT)&COINSPARK_PACKING_EXTEND_MASK, counts.firstInputBytes, counts.countInputsBytes, false)

		counts.firstOutputBytes, counts.countOutputsBytes = PackingExtendAddByteCounts((packingExtend>>COINSPARK_PACKING_EXTEND_OUTPUTS_SHIFT)&COINSPARK_PACKING_EXTEND_MASK, counts.firstOutputBytes, counts.countOutputsBytes, false)
	}

	// Packing for quantity

	packingQuantity := packing & COINSPARK_PACKING_QUANTITY_MASK

	switch packingQuantity {
	case COINSPARK_PACKING_QUANTITY_1_BYTE:
		counts.quantityBytes = 1
	case COINSPARK_PACKING_QUANTITY_2_BYTES:
		counts.quantityBytes = 2
	case COINSPARK_PACKING_QUANTITY_3_BYTES:
		counts.quantityBytes = 3
	case COINSPARK_PACKING_QUANTITY_4_BYTES:
		counts.quantityBytes = 4
	case COINSPARK_PACKING_QUANTITY_6_BYTES:
		counts.quantityBytes = 6
	case COINSPARK_PACKING_QUANTITY_FLOAT:
		counts.quantityBytes = COINSPARK_TRANSFER_QTY_FLOAT_LENGTH
	}

	// Return the resulting array
	return counts
}

func PackingExtendAddByteCounts(packingExtend byte, firstBytes int, countBytes int, forMessages bool) (firstBytesOut int, countBytesOut int) {

	switch packingExtend {
	case COINSPARK_PACKING_EXTEND_0_1_BYTE:
		if forMessages { // otherwise it's really COINSPARK_PACKING_EXTEND_1S
			countBytes = 1
		}
	case COINSPARK_PACKING_EXTEND_1_0_BYTE:
		firstBytes = 1

	case COINSPARK_PACKING_EXTEND_2_0_BYTES:
		firstBytes = 2

	case COINSPARK_PACKING_EXTEND_1_1_BYTES:
		firstBytes = 1
		countBytes = 1
	case COINSPARK_PACKING_EXTEND_2_1_BYTES:
		firstBytes = 2
		countBytes = 1
	case COINSPARK_PACKING_EXTEND_2_2_BYTES:
		firstBytes = 2
		countBytes = 2
	}
	firstBytesOut = firstBytes
	countBytesOut = countBytes
	return firstBytesOut, countBytesOut
}

func (p *CoinSparkTransfer) String() string {
	return p.StringInner(true)
}

func (p *CoinSparkTransfer) StringInner(headers bool) string {
	buffer := bytes.Buffer{}
	if headers {
		buffer.WriteString("COINSPARK TRANSFER\n")
	}

	isDefaultRoute := p.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE
	if isDefaultRoute {
		buffer.WriteString("      Default route:\n")
	} else {
		buffer.WriteString(p.assetRef.StringInner(false))
		buffer.WriteString(fmt.Sprintf("    Asset reference: %s\n", p.assetRef.Encode()))
	}

	if p.inputs.count > 0 {
		if p.inputs.count > 1 {
			buffer.WriteString(fmt.Sprintf("             Inputs: %d - %d (count %d)", p.inputs.first, p.inputs.first+p.inputs.count-1, p.inputs.count))
		} else {
			buffer.WriteString(fmt.Sprintf("              Input: %d", p.inputs.first))
		}
	} else {
		buffer.WriteString("             Inputs: none")
	}

	buffer.WriteString(fmt.Sprintf(" (small endian hex: first %s count %s)\n", UnsignedToSmallEndianHex(int64(p.inputs.first), 2), UnsignedToSmallEndianHex(int64(p.inputs.count), 2)))

	if p.outputs.count > 0 {
		if (p.outputs.count > 1) && !isDefaultRoute {
			buffer.WriteString(fmt.Sprintf("            Outputs: %d - %d (count %d)", p.outputs.first, p.outputs.first+p.outputs.count-1, p.outputs.count))
		} else {
			buffer.WriteString(fmt.Sprintf("             Output: %d", p.outputs.first))
		}
	} else {
		buffer.WriteString("            Outputs: none")
	}

	buffer.WriteString(fmt.Sprintf(" (small endian hex: first %s count %s)\n", UnsignedToSmallEndianHex(int64(p.outputs.first), 2), UnsignedToSmallEndianHex(int64(p.outputs.count), 2)))

	if !isDefaultRoute {
		buffer.WriteString(fmt.Sprintf("     Qty per output: %d (small endian hex %s", p.qtyPerOutput, UnsignedToSmallEndianHex(int64(p.qtyPerOutput), 8)))

		quantityPerOutput, mantissa, exponent := QtyToMantissaExponent(p.qtyPerOutput, 0,
			COINSPARK_TRANSFER_QTY_FLOAT_MANTISSA_MAX, COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MAX)

		if quantityPerOutput == int64(p.qtyPerOutput) {
			encodeQuantity := (exponent*COINSPARK_TRANSFER_QTY_FLOAT_EXPONENT_MULTIPLE + mantissa) & COINSPARK_TRANSFER_QTY_FLOAT_MASK
			buffer.WriteString(fmt.Sprintf(", as float %s", UnsignedToSmallEndianHex(int64(encodeQuantity), COINSPARK_TRANSFER_QTY_FLOAT_LENGTH)))
		}

		buffer.WriteString(")\n")
	}

	if headers {
		buffer.WriteString("END COINSPARK TRANSFER\n\n")
	}

	return buffer.String()

}

func (p *CoinSparkTransferList) Clear() {
	p.transfers = make([]CoinSparkTransfer, 0)
}

func (p *CoinSparkTransferList) String() string {
	buffer := bytes.Buffer{}
	buffer.WriteString("COINSPARK TRANSFERS\n")
	for i, t := range p.transfers {
		if i > 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(t.StringInner(false))
	}
	buffer.WriteString("END COINSPARK TRANSFERS\n\n")

	return buffer.String()
}

func (p *CoinSparkTransferList) IsValid() bool {
	for _, t := range p.transfers {
		if !t.IsValid() {
			return false
		}
	}
	return true
}

func (p *CoinSparkTransferList) GroupOrdering() []int {
	countTransfers := len(p.transfers)
	ordering := make([]int, countTransfers)
	transferUsed := make([]bool, countTransfers)

	for orderIndex, _ := range p.transfers {
		bestTransferScore := 0
		bestTransferIndex := -1
		transferScore := 0

		for transferIndex, _ := range p.transfers {
			transfer := p.transfers[transferIndex]
			if !transferUsed[transferIndex] {
				if transfer.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE {
					transferScore = 3 // top priority to default routes, which must be first in the encoded list
				} else if orderIndex > 0 && transfer.assetRef.Match(&p.transfers[ordering[orderIndex-1]].assetRef) {
					transferScore = 2 // then next best is one which has same asset reference as previous
				} else {
					transferScore = 1 // otherwise any will do
				}

				if transferScore > bestTransferScore { // if it's clearly the best, take it
					bestTransferScore = transferScore
					bestTransferIndex = transferIndex
				} else if transferScore == bestTransferScore { // otherwise give priority to "lower" asset references
					if transfer.assetRef.Compare(&p.transfers[bestTransferIndex].assetRef) < 0 {
						bestTransferIndex = transferIndex
					}
				}
			}
		}

		ordering[orderIndex] = bestTransferIndex
		transferUsed[bestTransferIndex] = true
	}

	return ordering
}

func (p *CoinSparkTransferList) Match(other *CoinSparkTransferList, strict bool) bool {
	countTransfers := len(p.transfers)
	if countTransfers != len(other.transfers) {
		return false
	}

	if strict {
		for i, t := range p.transfers {
			if !other.transfers[i].Match(&t) {
				return false
			}
		}
	}

	if !strict {
		thisOrdering := p.GroupOrdering()
		otherOrdering := other.GroupOrdering()
		for i, _ := range p.transfers {
			if !p.transfers[thisOrdering[i]].Match(&other.transfers[otherOrdering[i]]) {
				return false
			}
		}
	}

	return true
}

func (p *CoinSparkTransferList) CalcMinFee(countInputs int, outputsSatoshis []CoinSparkSatoshiQty, outputsRegular []bool) CoinSparkSatoshiQty {

	countOutputs := len(outputsSatoshis)
	if countOutputs != len(outputsRegular) {
		return COINSPARK_SATOSHI_QTY_MAX // these two arrays must be the same size
	}
	transfersToCover := 0

	for _, transfer := range p.transfers {

		if (transfer.assetRef.blockNum != COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE) && // don't count default routes
			(transfer.inputs.count > 0) &&
			(int(transfer.inputs.first) < countInputs) { // only count if at least one valid input index
			outputIndex := COINSPARK_MAX(int(transfer.outputs.first), 0)
			lastOutputIndex := COINSPARK_MIN(int(transfer.outputs.first+transfer.outputs.count), countOutputs) - 1

			for i := outputIndex; i <= lastOutputIndex; i++ {
				if outputsRegular[i] {
					transfersToCover += 1
				}
			}
		}
	}
	return CoinSparkSatoshiQty(transfersToCover) * GetMinFeeBasis(outputsSatoshis, outputsRegular)
}

func (p *CoinSparkTransferList) ApplyNone(inputBalances []CoinSparkAssetQty, outputsRegular []bool) []CoinSparkAssetQty {
	countOutputs := len(outputsRegular)
	outputBalances := make([]CoinSparkAssetQty, countOutputs)
	outputIndex := GetLastRegularOutput(outputsRegular)
	// -1 means None.
	if outputIndex != -1 {
		for _, inputBalance := range inputBalances {
			outputBalances[outputIndex] += inputBalance
		}
	}
	return outputBalances
}

func (p *CoinSparkTransferList) Apply(assetRef *CoinSparkAssetRef, genesis *CoinSparkGenesis, inputBalances []CoinSparkAssetQty, outputsRegular []bool) []CoinSparkAssetQty {
	// copy since we will modify it, and cast to integers
	localInputBalances := make([]CoinSparkAssetQty, len(inputBalances))
	copy(localInputBalances, inputBalances)

	countInputs := len(inputBalances)
	countOutputs := len(outputsRegular)
	outputBalances := make([]CoinSparkAssetQty, countOutputs)

	// Perform explicit transfers (i.e. not default routes)
	for _, transfer := range p.transfers {
		if assetRef.Match(&transfer.assetRef) {
			inputIndex := COINSPARK_MAX(int(transfer.inputs.first), 0)
			outputIndex := COINSPARK_MAX(int(transfer.outputs.first), 0)
			lastInputIndex := COINSPARK_MIN(inputIndex+int(transfer.inputs.count), countInputs) - 1
			lastOutputIndex := COINSPARK_MIN(outputIndex+int(transfer.outputs.count), countOutputs) - 1
			for outputIndex <= lastOutputIndex {
				if outputsRegular[outputIndex] == true {
					transferRemaining := transfer.qtyPerOutput
					for inputIndex <= lastInputIndex {
						transferQuantity := COINSPARK_MINASSETQTY(transferRemaining, inputBalances[inputIndex])
						if transferQuantity > 0 {
							//  skip all this if nothing is to be transferred (branch not really necessary)
							inputBalances[inputIndex] -= transferQuantity
							transferRemaining -= transferQuantity
							outputBalances[outputIndex] += transferQuantity
						}

						if transferRemaining > 0 {
							inputIndex += 1 // move to next input since self one is drained
						} else {
							break // stop if we have nothing left to transfer
						}

					}
				}

				outputIndex += 1
			}
		}
	}

	// Apply payment charges to all quantities not routed by default

	for outputIndex := 0; outputIndex < countOutputs; outputIndex++ {
		if outputsRegular[outputIndex] == true {
			outputBalances[outputIndex] = genesis.CalcNet(outputBalances[outputIndex])
		}
	}

	// Send remaining quantities to default outputs

	inputDefaultOutput := p.GetDefaultRouteMap(countInputs, outputsRegular)
	for inputIndex := 0; inputIndex < len(inputDefaultOutput); inputIndex++ {
		outputIndex := inputDefaultOutput[inputIndex]
		if outputIndex != -1 {
			outputBalances[outputIndex] += inputBalances[inputIndex]
		}
	}

	// Return the result

	return outputBalances
}

func (p *CoinSparkTransferList) DefaultOutputs(countInputs int, outputsRegular []bool) []bool {
	countOutputs := len(outputsRegular)
	outputsDefault := make([]bool, countOutputs)

	inputDefaultOutput := p.GetDefaultRouteMap(countInputs, outputsRegular)
	for _, outputIndex := range inputDefaultOutput {
		if outputIndex != -1 {
			outputsDefault[outputIndex] = true
		}
	}

	return outputsDefault
}

func (p *CoinSparkTransferList) GetDefaultRouteMap(countInputs int, outputsRegular []bool) []int {
	countOutputs := len(outputsRegular)

	// Default to last output for all inputs
	lastRegularOutput := GetLastRegularOutput(outputsRegular)
	inputDefaultOutput := make([]int, countInputs)
	for i := 0; i < countInputs; i++ {
		inputDefaultOutput[i] = lastRegularOutput
	}

	// Apply any default route transfers in reverse order (since early ones take precedence)
	for i := len(p.transfers) - 1; i >= 0; i-- {
		transfer := p.transfers[i]
		if transfer.assetRef.blockNum == COINSPARK_TRANSFER_BLOCK_NUM_DEFAULT_ROUTE {
			outputIndex := int(transfer.outputs.first)
			if (outputIndex >= 0) && (outputIndex < countOutputs) {
				inputIndex := COINSPARK_MAX(int(transfer.inputs.first), 0)
				lastInputIndex := COINSPARK_MIN(inputIndex+int(transfer.inputs.count), countInputs) - 1

				for inputIndex <= lastInputIndex {
					inputDefaultOutput[inputIndex] = outputIndex
					inputIndex += 1
				}
			}
		}
	}

	// Return the result

	return inputDefaultOutput
}

func (p *CoinSparkTransferList) Decode(metadataIn []byte, countInputs int, countOutputs int) int {
	metadata := LocateMetadataRange(metadataIn, COINSPARK_TRANSFERS_PREFIX)
	if metadata == nil {
		return 0
	}

	// Iterate over list

	p.transfers = make([]CoinSparkTransfer, 0)
	var previousTransfer *CoinSparkTransfer
	//previousTransfer = nil

	for len(metadata) > 0 {
		transfer := *new(CoinSparkTransfer)
		transferBytesUsed := transfer.Decode(metadata, previousTransfer, countInputs, countOutputs)

		if transferBytesUsed > 0 {
			p.transfers = append(p.transfers, transfer)
			metadata = metadata[transferBytesUsed:]
			previousTransfer = &transfer
		} else {
			return 0 // something was invalid
		}
	}
	// Return count
	return len(p.transfers)
}

func (p *CoinSparkTransferList) Encode(countInputs int, countOutputs int, metadataMaxLen int) []byte {

	buf := bytes.Buffer{}
	// 4-character identifier

	buf.WriteString(COINSPARK_METADATA_IDENTIFIER)
	buf.WriteByte(COINSPARK_TRANSFERS_PREFIX)

	// Encode each transfer, grouping by asset reference, but preserving original order otherwise
	ordering := p.GroupOrdering()
	countTransfers := len(p.transfers)
	var previousTransfer *CoinSparkTransfer
	previousTransfer = nil

	for transferIndex := 0; transferIndex < countTransfers; transferIndex++ {
		thisTransfer := p.transfers[ordering[transferIndex]]

		written := thisTransfer.Encode(previousTransfer, metadataMaxLen-buf.Len(), countInputs, countOutputs)
		if written == nil {
			return nil
		}

		buf.Write(written)
		previousTransfer = &thisTransfer
	}

	// Extra length check (even though thisTransfer.encode() should be sufficient)

	if buf.Len() > metadataMaxLen {
		return nil
	}

	// Return what we created

	return buf.Bytes()
}

func (p *CoinSparkMessage) String() string {
	hostPathMetadata := EncodeDomainAndOrPath(p.serverHost, p.useHttps, p.serverPath, p.usePrefix, true)
	urlString := p.CalcServerURL()
	buffer := bytes.Buffer{}
	buffer.WriteString("COINSPARK MESSAGE\n")
	buffer.WriteString(fmt.Sprintf("    Server URL: %s (length %d+%d encoded %s length %d)\n", urlString, len(p.serverHost), len(p.serverPath), strings.ToUpper(hex.EncodeToString(hostPathMetadata)), len(hostPathMetadata)))
	buffer.WriteString("Public message: ")
	if p.isPublic {
		buffer.WriteString("yes\n")
	} else {
		buffer.WriteString("no\n")
	}
	for _, outputRange := range p.outputRanges {
		if outputRange.count > 0 {
			if outputRange.count > 1 {
				buffer.WriteString(fmt.Sprintf("       Outputs: %d - %d (count %d)", outputRange.first, outputRange.first+outputRange.count-1, outputRange.count))
			} else {
				buffer.WriteString(fmt.Sprintf("        Output: %d", outputRange.first))
			}
		} else {
			buffer.WriteString("       Outputs: none")
		}

		buffer.WriteString(fmt.Sprintf(" (small endian hex: first %s count %s)\n", UnsignedToSmallEndianHex(int64(outputRange.first), 2), UnsignedToSmallEndianHex(int64(outputRange.count), 2)))
	}

	buffer.WriteString(fmt.Sprintf("  Message hash: %s (length %d)\n", strings.ToUpper(hex.EncodeToString(p.hash[:p.hashLen])), p.hashLen))
	buffer.WriteString("END COINSPARK MESSAGE\n\n")

	return buffer.String()
}

func (p *CoinSparkMessage) IsValid() bool {
	if len(p.serverHost) > COINSPARK_MESSAGE_SERVER_HOST_MAX_LEN {
		return false
	}

	if len(p.serverPath) > COINSPARK_MESSAGE_SERVER_PATH_MAX_LEN {
		return false
	}

	if len(p.hash) < p.hashLen {
		// check we have at least as much data as specified by self.hashLen
		return false
	}

	if p.hashLen < COINSPARK_MESSAGE_HASH_MIN_LEN || p.hashLen > COINSPARK_MESSAGE_HASH_MAX_LEN {
		return false
	}

	if !p.isPublic && len(p.outputRanges) == 0 {
		// public or aimed at some outputs at least
		return false
	}

	if len(p.outputRanges) > COINSPARK_MESSAGE_MAX_IO_RANGES {
		return false
	}

	for _, outputRange := range p.outputRanges {
		if !outputRange.IsValid() {
			return false
		}
	}

	return true

}

func (p *CoinSparkMessage) Match(other *CoinSparkMessage, strict bool) bool {
	hashCompareLen := COINSPARK_MIN(p.hashLen, other.hashLen)
	hashCompareLen = COINSPARK_MIN(hashCompareLen, COINSPARK_MESSAGE_HASH_MAX_LEN)

	var thisRanges, otherRanges []CoinSparkIORange
	if strict {
		thisRanges = p.outputRanges
		otherRanges = other.outputRanges
	} else {
		thisRanges = NormalizeIORanges(p.outputRanges)
		otherRanges = NormalizeIORanges(other.outputRanges)
	}

	if len(thisRanges) != len(otherRanges) {
		return false
	}

	for index := 0; index < len(thisRanges); index++ {
		if !thisRanges[index].Match(&otherRanges[index]) {
			return false
		}
	}

	return (p.useHttps == other.useHttps &&
		strings.ToLower(p.serverHost) == strings.ToLower(other.serverHost) &&
		p.usePrefix == other.usePrefix &&
		strings.ToLower(p.serverPath) == strings.ToLower(other.serverPath) &&
		p.isPublic == other.isPublic &&
		0 == bytes.Compare(p.hash[:hashCompareLen], other.hash[:hashCompareLen]))

}

func (p *CoinSparkMessage) Encode(countOutputs int, metadataMaxLen int) []byte {
	if !p.IsValid() {
		return nil
	}

	//4-character identifier

	buf := bytes.Buffer{}
	buf.WriteString(COINSPARK_METADATA_IDENTIFIER)
	buf.WriteByte(COINSPARK_MESSAGE_PREFIX)

	// Server host and path

	written := EncodeDomainAndOrPath(p.serverHost, p.useHttps, p.serverPath, p.usePrefix, true)
	if written == nil {
		return nil
	}
	buf.Write(written)

	// Output ranges

	if p.isPublic {
		//add public indicator first
		var packing byte
		packing = COINSPARK_OUTPUTS_TYPE_EXTEND | COINSPARK_OUTPUTS_TYPE_EXTEND
		if len(p.outputRanges) > 0 {
			packing = packing | COINSPARK_OUTPUTS_MORE_FLAG
		}
		buf.WriteByte(packing)
	}

	for index := 0; index < len(p.outputRanges); index++ {
		// other output ranges
		outputRange := p.outputRanges[index]

		success, packingResult := GetOutputRangePacking(outputRange, countOutputs)
		if success == false {
			return nil
		}

		// The packing byte

		packing := packingResult.packing

		if (index + 1) < len(p.outputRanges) {
			packing |= COINSPARK_OUTPUTS_MORE_FLAG
		}
		buf.WriteByte(byte(packing))

		hexString := UnsignedToSmallEndianHex(int64(outputRange.first), int(packingResult.firstBytes))
		hexBytes, _ := hex.DecodeString(hexString)
		buf.Write(hexBytes)

		// The number of outputs, if necessary
		hexString = UnsignedToSmallEndianHex(int64(outputRange.count), int(packingResult.countBytes))
		hexBytes, _ = hex.DecodeString(hexString)
		buf.Write(hexBytes)
	}

	// Message hash
	buf.Write(p.hash[:p.hashLen])

	// Check the total length is within the specified limit

	if buf.Len() > metadataMaxLen {
		return nil
	}

	// Return what we created
	return buf.Bytes()
}

func (p *CoinSparkMessage) Decode(buffer []byte, countOutputs int) bool {
	metadata := LocateMetadataRange(buffer, COINSPARK_MESSAGE_PREFIX)
	if metadata == nil {
		return false
	}

	// Server host and path
	success, decoded := DecodeDomainAndOrPath(string(metadata), true, true, true)
	if !success {
		return false
	}

	metadata = metadata[decoded.decodedChars:]
	p.useHttps = decoded.useHttps
	p.serverHost = decoded.domainName
	p.usePrefix = decoded.usePrefix
	p.serverPath = decoded.pagePath

	// Output ranges
	p.isPublic = false
	p.outputRanges = make([]CoinSparkIORange, 0)

	var outputRange CoinSparkIORange

	readAnotherRange := true

	for readAnotherRange == true {
		success, packing := ShiftLittleEndianBytesToInt(&metadata, 1)
		//Read the next packing byte and check reserved bits are zero
		if success == false {
			return false
		}

		if packing&COINSPARK_OUTPUTS_RESERVED_MASK > 0 {
			return false
		}

		readAnotherRange = packing&COINSPARK_OUTPUTS_MORE_FLAG > 0
		packingType := packing & COINSPARK_OUTPUTS_TYPE_MASK
		packingValue := packing & COINSPARK_OUTPUTS_VALUE_MASK

		if (packingType == COINSPARK_OUTPUTS_TYPE_EXTEND) && (packingValue == COINSPARK_PACKING_EXTEND_PUBLIC) {
			p.isPublic = true
			//special case for public messages
		} else {
			// Create a new output range
			if len(p.outputRanges) >= COINSPARK_MESSAGE_MAX_IO_RANGES {
				// too many output ranges
				return false
			}

			firstBytes := 0
			countBytes := 0

			// Decode packing byte

			if packingType == COINSPARK_OUTPUTS_TYPE_SINGLE {
				// inline single input
				outputRange = CoinSparkIORange{}
				outputRange.first = CoinSparkIOIndex(packingValue)
				outputRange.count = 1

			} else if packingType == COINSPARK_OUTPUTS_TYPE_FIRST {
				// inline first few outputs
				outputRange = CoinSparkIORange{}
				outputRange.first = 0
				outputRange.count = CoinSparkIOIndex(packingValue)

			} else if packingType == COINSPARK_OUTPUTS_TYPE_EXTEND {
				// we'll be taking additional bytes
				success, extendPackingType := DecodePackingExtend(byte(packingValue), true)
				if !success {
					return false
				}

				outputRange = PackingTypeToValues(extendPackingType, nil, countOutputs)

				firstBytes, countBytes = PackingExtendAddByteCounts(byte(packingValue), firstBytes, countBytes, true)

			} else {
				return false
				//will be self.COINSPARK_OUTPUTS_TYPE_UNUSED
			}

			// The index of the first output and number of outputs, if necessary

			success, v := ShiftLittleEndianBytesToInt(&metadata, firstBytes)
			if !success {
				return false
			} else if firstBytes > 0 {
				outputRange.first = CoinSparkIOIndex(v)
			}

			success, v = ShiftLittleEndianBytesToInt(&metadata, countBytes)
			if !success {
				return false
			} else if countBytes > 0 {
				outputRange.count = CoinSparkIOIndex(v)
			}

			// Add on the new output range

			p.outputRanges = append(p.outputRanges, outputRange)

		}
	}

	// Message hash
	p.hashLen = COINSPARK_MIN(len(metadata), COINSPARK_MESSAGE_HASH_MAX_LEN)
	p.hash = metadata[:p.hashLen] // insufficient length will be caught by isValid()

	// Return validity
	return p.IsValid()

}

func (p *CoinSparkMessage) GetHashLen() int {
	return p.hashLen
}

func (p *CoinSparkMessage) CalcHashLen(countOutputs int, metadataMaxLen int) int {
	hashLen := metadataMaxLen - COINSPARK_METADATA_IDENTIFIER_LEN - 1
	hostPathLen := len(p.serverPath) + 1
	theIP := net.ParseIP(p.serverHost)
	if theIP != nil {
		theIP = theIP.To4() // could return 16 byte slice
	}
	if theIP != nil {
		hashLen -= 5 // packing and IP octets
		if hostPathLen == 1 {
			hostPathLen = 0 // will skip server path in this case
		}
	} else {
		hashLen -= 1 // packing
		shortDomainName, _ := ShrinkLowerDomainName(p.serverHost)
		hostPathLen += len(shortDomainName) + 1
	}

	hashLen -= 2 * int((hostPathLen+2)/3) // uses integer arithmetic

	if p.isPublic {
		hashLen -= 1
	}

	for _, outputRange := range p.outputRanges {
		success, packingResult := GetOutputRangePacking(outputRange, countOutputs)
		if success {
			hashLen -= 1 + packingResult.firstBytes + packingResult.countBytes
		}
	}

	return COINSPARK_MIN(COINSPARK_MAX(hashLen, 0), COINSPARK_MESSAGE_HASH_MAX_LEN)
}

func (p *CoinSparkMessage) CalcServerURL() string {
	buffer := bytes.Buffer{}
	if p.useHttps {
		buffer.WriteString("https://")
	} else {
		buffer.WriteString("http://")
	}
	buffer.WriteString(p.serverHost)
	buffer.WriteString("/")
	if p.usePrefix {
		buffer.WriteString("coinspark/")
	}
	if len(p.serverPath) > 0 {
		buffer.WriteString(p.serverPath)
		buffer.WriteString("/")
	}
	return strings.ToLower(buffer.String())
}

func EncodePackingExtend(packingOptions map[string]bool) (bool, byte) {
	for _, packingType := range packingExtendMapOrder {
		if packingOptions[packingType] {
			return true, packingExtendMap[packingType]
		}
	}
	return false, 0
}

func NormalizeIORanges(inRanges []CoinSparkIORange) []CoinSparkIORange {
	countRanges := len(inRanges)
	if countRanges == 0 {
		return inRanges
	}

	rangeUsed := make([]bool, countRanges)
	var outRanges []CoinSparkIORange
	countRemoved := 0

	var lowestRangeFirst, lowestRangeIndex, lastRangeEnd CoinSparkIOIndex

	for orderIndex := 0; orderIndex < countRanges; orderIndex++ {
		lowestRangeFirst = 0
		lowestRangeIndex = -1

		for rangeIndex := 0; rangeIndex < countRanges; rangeIndex++ {
			if rangeUsed[rangeIndex] == false {
				if lowestRangeIndex == -1 || inRanges[rangeIndex].first < lowestRangeFirst {
					lowestRangeFirst = inRanges[rangeIndex].first
					lowestRangeIndex = CoinSparkIOIndex(rangeIndex)
				}
			}
		}

		if orderIndex > 0 && inRanges[lowestRangeIndex].first <= lastRangeEnd {
			// we can combine two adjacent ranges
			countRemoved += 1
			thisRangeEnd := inRanges[lowestRangeIndex].first + inRanges[lowestRangeIndex].count
			outRanges[orderIndex-countRemoved].count = CoinSparkIOIndex(COINSPARK_MAX(int(lastRangeEnd), int(thisRangeEnd))) - outRanges[orderIndex-countRemoved].first
		} else {
			outRanges = append(outRanges, inRanges[lowestRangeIndex])
		}

		lastRangeEnd = outRanges[orderIndex-countRemoved].first + outRanges[orderIndex-countRemoved].count
		rangeUsed[lowestRangeIndex] = true
	}
	return outRanges
}

func GetOutputRangePacking(outputRange CoinSparkIORange, countOutputs int) (bool, OutputRangePacking) {
	packingOptions := GetPackingOptions(nil, &outputRange, countOutputs, true)

	var packing int
	firstBytes := 0
	countBytes := 0

	if packingOptions["_1_0_BYTE"] && (outputRange.first <= COINSPARK_OUTPUTS_VALUE_MAX) {
		//# inline single output
		packing = COINSPARK_OUTPUTS_TYPE_SINGLE | (int(outputRange.first) & COINSPARK_OUTPUTS_VALUE_MASK)
	} else if packingOptions["_0_1_BYTE"] && (outputRange.count <= COINSPARK_OUTPUTS_VALUE_MAX) {
		// inline first few outputs
		packing = COINSPARK_OUTPUTS_TYPE_FIRST | (int(outputRange.count) & COINSPARK_OUTPUTS_VALUE_MASK)
	} else {
		// we'll be taking additional bytes
		success, packingExtend := EncodePackingExtend(packingOptions)
		if !success {
			return false, OutputRangePacking{}
		}

		firstBytes, countBytes = PackingExtendAddByteCounts(packingExtend, firstBytes, countBytes, true)

		packing = COINSPARK_OUTPUTS_TYPE_EXTEND | (int(packingExtend) & COINSPARK_OUTPUTS_VALUE_MASK)
	}

	var result OutputRangePacking
	result.packing = packing
	result.firstBytes = firstBytes
	result.countBytes = countBytes
	return true, result
}

func GetPackingOptions(previousRange *CoinSparkIORange, r *CoinSparkIORange, countInputsOutputs int, forMessages bool) map[string]bool {
	packingOptions := map[string]bool{}

	firstZero := (r.first == 0)
	firstByte := (r.first <= COINSPARK_UNSIGNED_BYTE_MAX)
	first2Bytes := (r.first <= COINSPARK_UNSIGNED_2_BYTES_MAX)
	countOne := (r.count == 1)
	countByte := (r.count <= COINSPARK_UNSIGNED_BYTE_MAX)

	if forMessages {
		packingOptions["_0P"] = false
		packingOptions["_1S"] = false // these two options not used for messages
		packingOptions["_0_1_BYTE"] = firstZero && countByte
	} else {
		if previousRange != nil {
			packingOptions["_0P"] = (r.first == previousRange.first) && (r.count == previousRange.count)
			packingOptions["_1S"] = (r.first == (previousRange.first + previousRange.count)) && countOne
		} else {
			packingOptions["_0P"] = firstZero && countOne
			packingOptions["_1S"] = (r.first == 1) && countOne
		}
		packingOptions["_0_1_BYTE"] = false // this option not used for transfers
	}

	packingOptions["_1_0_BYTE"] = firstByte && countOne
	packingOptions["_2_0_BYTES"] = first2Bytes && countOne
	packingOptions["_1_1_BYTES"] = firstByte && countByte
	packingOptions["_2_1_BYTES"] = first2Bytes && countByte
	packingOptions["_2_2_BYTES"] = first2Bytes && (r.count <= COINSPARK_UNSIGNED_2_BYTES_MAX)
	packingOptions["_ALL"] = firstZero && (int(r.count) >= countInputsOutputs)

	return packingOptions

}

func ScriptToMetadata(scriptPubKey string, scriptIsHex bool) []byte {
	scriptPubKeyRaw := GetRawScript(scriptPubKey, scriptIsHex)
	scriptPubKeyRawLen := len(scriptPubKeyRaw)
	metadataLen := scriptPubKeyRawLen - 2

	if (scriptPubKeyRawLen > 2) &&
		(scriptPubKeyRaw[0] == 0x6a) &&
		(scriptPubKeyRaw[1] > 0) &&
		(scriptPubKeyRaw[1] <= 75) &&
		(int(scriptPubKeyRaw[1]) == metadataLen) {
		return scriptPubKeyRaw[2:]
	}
	return nil
}

func ScriptIsRegular(scriptPubKey string, scriptIsHex bool) bool {
	scriptPubKeyRaw := GetRawScript(scriptPubKey, scriptIsHex)
	return len(scriptPubKeyRaw) < 1 || scriptPubKeyRaw[0] != 0x6a
}

func GetRawScript(scriptPubKey string, scriptIsHex bool) []byte {
	if scriptIsHex {
		bytes, _ := hex.DecodeString(scriptPubKey)
		return bytes
	}
	return []byte(scriptPubKey)
}

func MetadataMaxAppendLen(metadata []byte, metadataMaxLen int) int {
	return COINSPARK_MAX(metadataMaxLen-(len(metadata)+1-COINSPARK_METADATA_IDENTIFIER_LEN), 0)
}

func MetadataAppend(metadata []byte, metadataMaxLen int, appendMetadata []byte) []byte {
	lastMetadata := LocateMetadataRange(metadata, COINSPARK_DUMMY_PREFIX) // check we can find last metadata
	if lastMetadata == nil {
		return nil
	}

	if len(appendMetadata) < (COINSPARK_METADATA_IDENTIFIER_LEN + 1) {
		// check there is enough to check the prefix
		return nil
	}

	if string(appendMetadata[:COINSPARK_METADATA_IDENTIFIER_LEN]) != COINSPARK_METADATA_IDENTIFIER {
		// then check the prefix
		return nil
	}

	// we don't check the character after the prefix in appendMetadata because it could itself be composite

	needLength := len(metadata) + len(appendMetadata) - COINSPARK_METADATA_IDENTIFIER_LEN + 1 // check there is enough space
	if metadataMaxLen < needLength {
		return nil
	}

	lastMetadataLen := len(lastMetadata) + 1 // include prefix
	lastMetadataPos := len(metadata) - lastMetadataLen

	buf := bytes.Buffer{}
	buf.Write(metadata[:lastMetadataPos])
	buf.WriteByte(byte(lastMetadataLen))
	buf.Write(metadata[lastMetadataPos:])
	buf.Write(appendMetadata[COINSPARK_METADATA_IDENTIFIER_LEN:])
	return buf.Bytes()
}

func MetadataToScript(metadata []byte, toHexScript bool) string {
	if len(metadata) <= 75 {
		scriptPubKey := bytes.Buffer{}
		scriptPubKey.WriteByte(0x6a)
		scriptPubKey.WriteByte(byte(len(metadata)))
		scriptPubKey.Write(metadata)
		if toHexScript {
			return strings.ToUpper(hex.EncodeToString(scriptPubKey.Bytes()))
		}
		return string(scriptPubKey.Bytes())
	}
	return ""
}
