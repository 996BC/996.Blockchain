package crypto

import (
	"encoding/base32"

	"github.com/btcsuite/btcd/btcec"
)

var (
	base32Codec = base32.StdEncoding.WithPadding(base32.NoPadding)
)

// PubKeyToID returns a peer id from the public key
func PubKeyToID(pubKey *btcec.PublicKey) string {
	pubKeyB := pubKey.SerializeCompressed()
	return base32Codec.EncodeToString(pubKeyB)
}

// PrivKeyToID returns a peer id from the private key
func PrivKeyToID(privKey *btcec.PrivateKey) string {
	pubKey := privKey.PubKey()
	return PubKeyToID(pubKey)
}

// IDToPubKey returns a public key from id; if error happens returns nil
func IDToPubKey(id string) *btcec.PublicKey {
	pubKeyB := IDToBytes(id)
	if pubKeyB == nil {
		return nil
	}

	pubKey, err := btcec.ParsePubKey(pubKeyB, btcec.S256())
	if err != nil {
		return nil
	}
	return pubKey
}

// IDToBytes returns a public key Serialize compressed bytes; if error happens returns nil
func IDToBytes(id string) []byte {
	pubKeyB, _ := base32Codec.DecodeString(id)
	return pubKeyB
}

// BytesToID return a peer id from the public key Serialize compressed bytes
func BytesToID(compressedKey []byte) string {
	return base32Codec.EncodeToString(compressedKey)
}
