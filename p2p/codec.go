package p2p

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"

	"github.com/btcsuite/btcd/btcec"
)

// codec is used to encrypt/decrypt message
type codec interface {
	encrypt(plainText []byte) ([]byte, error)
	decrypt(cipherText []byte) ([]byte, error)
}

// implement of 'codec'
type aesgcmCodec struct {
	aead  cipher.AEAD
	nonce []byte
}

func (aes *aesgcmCodec) encrypt(plainText []byte) ([]byte, error) {
	cipherText := aes.aead.Seal(nil, aes.nonce, plainText, nil)
	return cipherText, nil
}

func (aes *aesgcmCodec) decrypt(cipherText []byte) ([]byte, error) {
	plainText, err := aes.aead.Open(nil, aes.nonce, cipherText, nil)
	return plainText, err
}

func newAESGCMCodec(remotePubKey *btcec.PublicKey, randPrivKey *btcec.PrivateKey) (*aesgcmCodec, error) {

	sharedKey := sha512.Sum512(btcec.GenerateSharedSecret(randPrivKey, remotePubKey))

	block, err := aes.NewCipher(sharedKey[:32])
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &aesgcmCodec{
		aead:  aesgcm,
		nonce: sharedKey[32 : 32+12],
	}, nil
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
