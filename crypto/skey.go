package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/howeyc/gopass"
	"github.com/996BC/996.Blockchain/utils"
	"golang.org/x/crypto/scrypt"
)

/*
The skey is the sealed private key stored on the disk.
It's safer than plain key storage.Users can export a skey from an pkey and vice versa.
The aes key used to encrypt the ecc private key is derived by the scrypt.
*/

const (
	SealKeyType = 2
	SealKey     = ".sKey"

	version1   = 1
	kdfName    = "scrypt"
	dkLen      = 32
	scryptN    = 262144
	scryptP    = 1
	scryptR    = 8
	saltLen    = 32
	cryptoName = "aes-256-gcm"
)

type skeyJSON struct {
	Version    int         `json:"version"`
	KdfName    string      `json:"kdfName"`
	KDF        interface{} `json:"kdf"`
	CryptoName string      `json:"cryptoName"`
	Crypto     interface{} `json:"crypto"`
}

type scryptKDF struct {
	DkLen int    `json:"dkLen"`
	N     int    `json:"n"`
	P     int    `json:"p"`
	R     int    `json:"r"`
	Salt  string `json:"salt"`
}

type aes256GcmCrypto struct {
	CipherText string `json:"cipherText"`
	Nonce      string `json:"nonce"`
}

// NewSKey generates an key for users, then seals and saves it
func NewSKey(path string) (*btcec.PrivateKey, error) {
	keyFile := path + "/" + SealKey
	if err := checkBeforeNewKey(path, keyFile); err != nil {
		return nil, err
	}

	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}

	err = genSKeyAndSaveIt(privKey, keyFile)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// SealPKey seals a pkey and saves it
func SealPKey(pkeyPath string, outputPath string) error {
	keyFile := outputPath + "/" + SealKey
	if err := checkBeforeNewKey(outputPath, keyFile); err != nil {
		return err
	}

	privKey, err := RestorePKey(pkeyPath)
	if err != nil {
		return err
	}

	return genSKeyAndSaveIt(privKey, keyFile)
}

// ReNewSKey gennerates a new sKey from an existing old sKey
func ReNewSKey(oldKeyPath string, newKeyPath string) error {
	newKeyFile := newKeyPath + "/" + SealKey
	if err := utils.AccessCheck(newKeyPath); err != nil {
		return err
	}

	privKey, err := RestoreSKey(oldKeyPath)
	if err != nil {
		return err
	}

	return genSKeyAndSaveIt(privKey, newKeyFile)
}

// RestoreSKey restores private key from file
func RestoreSKey(path string) (*btcec.PrivateKey, error) {
	keyFile := path + "/" + SealKey
	jsonBytes, err := readKeyFile(keyFile)
	if err != nil {
		return nil, err
	}

	ks, kdf, aesCrypto, err := jsonUnMarshal(jsonBytes)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Input your passphrase to decrypt your key:")
	pass, err := gopass.GetPasswdMasked()
	if err != nil {
		return nil, fmt.Errorf("Get passphrase failed:%v", err)
	}

	return aesDecrypt(pass, ks, kdf, aesCrypto)
}

// three steps:
// 1. get user's passphrase
// 2. use passphrase to seal the private key
// 3. save the sealed content on the disk
func genSKeyAndSaveIt(privKey *btcec.PrivateKey, outputFile string) error {
	pass, err := getPassphrase()
	if err != nil {
		return err
	}

	sealedContent, err := seal(pass, privKey.Serialize())
	if err != nil {
		return err
	}

	if err := saveOnDisk(sealedContent, outputFile); err != nil {
		return err
	}

	return nil
}

func getPassphrase() ([]byte, error) {
	fmt.Printf("Input your passphrase(Please Remember it):")
	pass1, err := gopass.GetPasswdMasked()
	if err != nil {
		return nil, fmt.Errorf("Get passphrase failed:%v", err)
	} else if len(pass1) < 8 {
		return nil, fmt.Errorf("Password should be at least 8 characters")
	}
	fmt.Printf("Repeat it:")
	pass2, err := gopass.GetPasswdMasked()
	if err != nil {
		return nil, fmt.Errorf("Get passwor failed:%v", err)
	}
	if !bytes.Equal(pass1, pass2) {
		return nil, errors.New("Inconsistent input")
	}

	return pass1, nil
}

func seal(passphrase []byte, key []byte) ([]byte, error) {
	salt := make([]byte, saltLen)
	var err error
	var dk []byte
	if _, err = rand.Read(salt); err != nil {
		return nil, err
	}

	if dk, err = scrypt.Key(passphrase, salt, scryptN, scryptR, scryptP, dkLen); err != nil {
		return nil, err
	}

	// use AES-256-GCM to encrypt the private key
	nonce, cipherText, err := aesEncrypt(key, dk)
	if err != nil {
		return nil, err
	}

	sealedContent, err := jsonMarshal(utils.ToHex(nonce), utils.ToHex(cipherText), utils.ToHex(salt))
	if err != nil {
		return nil, err
	}

	return sealedContent, nil
}

func aesEncrypt(plaintext []byte, key []byte) (nonceRet, cipherTextRet []byte, err error) {
	if len(key) != 32 {
		return nil, nil, fmt.Errorf("AES key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	var aesgcm cipher.AEAD
	if aesgcm, err = cipher.NewGCM(block); err != nil {
		return nil, nil, err
	}

	cipherText := aesgcm.Seal(nil, nonce, plaintext, nil)
	return nonce, cipherText, nil
}

func jsonMarshal(nonce, cipherText, salt string) ([]byte, error) {
	kdf := &scryptKDF{
		DkLen: dkLen,
		N:     scryptN,
		P:     scryptP,
		R:     scryptR,
		Salt:  salt,
	}

	aesCrypto := &aes256GcmCrypto{
		CipherText: cipherText,
		Nonce:      nonce,
	}

	ks := skeyJSON{
		Version:    version1,
		KdfName:    kdfName,
		KDF:        kdf,
		CryptoName: cryptoName,
		Crypto:     aesCrypto,
	}

	var jsonBytes []byte
	var err error
	if jsonBytes, err = json.MarshalIndent(ks, "", "  "); err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func jsonUnMarshal(jsonBytes []byte) (*skeyJSON, *scryptKDF, *aes256GcmCrypto, error) {
	ks := &skeyJSON{}
	kdf := &scryptKDF{}
	aesCrypto := &aes256GcmCrypto{}
	ks.KDF = kdf
	ks.Crypto = aesCrypto
	if err := json.Unmarshal(jsonBytes, &ks); err != nil {
		return nil, nil, nil, err
	}
	if err := checkSealParams(ks, kdf, aesCrypto); err != nil {
		return nil, nil, nil, err
	}

	return ks, kdf, aesCrypto, nil
}

func checkSealParams(ks *skeyJSON, kdf *scryptKDF, aesCrypto *aes256GcmCrypto) error {
	if ks.Version != version1 {
		return fmt.Errorf("unrecognized version:%d", ks.Version)
	}
	if ks.KdfName != kdfName {
		return fmt.Errorf("unrecognized kdf:%s", ks.KdfName)
	}
	if ks.CryptoName != cryptoName {
		return fmt.Errorf("unrecognized crypto:%s", ks.CryptoName)
	}

	if kdf.DkLen != dkLen {
		return fmt.Errorf("unrecognized dkLen:%d", kdf.DkLen)
	}
	if kdf.N != scryptN {
		return fmt.Errorf("unrecognized n:%d", kdf.N)
	}
	if kdf.P != scryptP {
		return fmt.Errorf("unrecognized p:%d", kdf.P)
	}
	if kdf.R != scryptR {
		return fmt.Errorf("unrecognized r:%d", kdf.R)
	}
	if len(kdf.Salt) == 0 || len(aesCrypto.CipherText) == 0 ||
		len(aesCrypto.Nonce) == 0 {
		return fmt.Errorf("the essential content is missed")
	}
	return nil
}

func aesDecrypt(pass []byte, ks *skeyJSON, kdf *scryptKDF, aesCrypto *aes256GcmCrypto) (*btcec.PrivateKey, error) {
	var dk []byte
	var plainText []byte
	var block cipher.Block
	var aesgcm cipher.AEAD
	var err error

	salt, _ := utils.FromHex(kdf.Salt)
	if dk, err = scrypt.Key(pass, salt, kdf.N, kdf.R, kdf.P, kdf.DkLen); err != nil {
		return nil, err
	}

	if block, err = aes.NewCipher(dk); err != nil {
		return nil, err
	}

	if aesgcm, err = cipher.NewGCM(block); err != nil {
		return nil, err
	}

	nonce, _ := utils.FromHex(aesCrypto.Nonce)
	cipherText, _ := utils.FromHex(aesCrypto.CipherText)
	if plainText, err = aesgcm.Open(nil, nonce, cipherText, nil); err != nil {
		return nil, err
	}

	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), plainText)
	if privateKey == nil {
		return nil, fmt.Errorf("recover nil private key")
	}

	return privateKey, nil
}
