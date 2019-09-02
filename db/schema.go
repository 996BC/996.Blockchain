package db

import (
	"bytes"
	"encoding/binary"
)

var (
	// height is a byte array of a uint64 number
	headerPrefix         = []byte("h")     // headerPrefix + height + hash -> header
	hashSuffix           = []byte("n")     // headerPrefix + height + hashSuffix -> header hash
	headerHeightPrefix   = []byte("H")     // headerHeightPrefix + hash -> height
	blockPrefix          = []byte("b")     // blockPrefix + height + hash -> block
	evidencePrefix       = []byte("e")     // evidencePrefix + height + hash -> evidence
	evidenceHeightPrefix = []byte("E")     // evidenceHeightPrefix + hash -> height
	scoreSuffix          = []byte("Score") // pubKey + scoreSuffix -> score
	evidenceSuffix       = []byte("e")     // pubKey + evidenceSuffix + evidenceHash -> height

	// meta data key should begin with 'm'
	mLatestHeight = []byte("mLatestHeigh")
	mGenesis      = []byte("mGenesis")
)

func hbyte(height uint64) []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, height)
	return result
}

func byteh(data []byte) uint64 {
	var result uint64
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.BigEndian, &result)
	return result
}

// h..
func getHeaderKey(height uint64, hash []byte) []byte {
	return append(headerPrefix, append(hbyte(height), hash...)...)
}

// h..n
func getHashKey(height uint64) []byte {
	return append(headerPrefix, append(hbyte(height), hashSuffix...)...)
}

// H..
func getHeaderHeightKey(hash []byte) []byte {
	return append(headerHeightPrefix, hash...)
}

// b..
func getBlockKey(height uint64, hash []byte) []byte {
	return append(blockPrefix, append(hbyte(height), hash...)...)
}

// e..
func getEvidenceKey(height uint64, hash []byte) []byte {
	return append(evidencePrefix, append(hbyte(height), hash...)...)
}

// E..
func getEvidenceHeightKey(hash []byte) []byte {
	return append(evidenceHeightPrefix, hash...)
}

// ..Score
func getScoreKey(key []byte) []byte {
	return append(key, scoreSuffix...)
}

// ..e
func getAccountEvidenceKeyPrefix(key []byte) []byte {
	return append(key, evidenceSuffix...)
}
