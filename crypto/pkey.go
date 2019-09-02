package crypto

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/utils"
)

/*
The pkey is the plain private key stored on the disk.
*/

const (
	PlainKeyType = 1
	PlainKey     = ".pKey"
)

// NewPKey generates a key for users, then saves it
func NewPKey(path string) (*btcec.PrivateKey, error) {
	keyFile := path + "/" + PlainKey
	if err := checkBeforeNewKey(path, keyFile); err != nil {
		return nil, err
	}

	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}

	hexPrivKey := utils.ToHex(privKey.Serialize())
	if err = saveOnDisk([]byte(hexPrivKey), keyFile); err != nil {
		return nil, err
	}
	return privKey, nil
}

// OpenSKey opens a skey and saves it
func OpenSKey(skeyPath string, outputPath string) error {
	keyFile := outputPath + "/" + PlainKey
	if err := checkBeforeNewKey(outputPath, keyFile); err != nil {
		return err
	}

	privKey, err := RestoreSKey(skeyPath)
	if err != nil {
		return err
	}

	hexPrivKey := utils.ToHex(privKey.Serialize())
	if err = saveOnDisk([]byte(hexPrivKey), keyFile); err != nil {
		return err
	}
	return nil
}

// RestorePKey restores private key from file
func RestorePKey(path string) (*btcec.PrivateKey, error) {
	keyFile := path + "/" + PlainKey
	hexPrivKey, err := readKeyFile(keyFile)
	if err != nil {
		return nil, err
	}

	bytePrivKey, err := utils.FromHex(string(hexPrivKey))
	if err != nil {
		return nil, err
	}

	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), bytePrivKey)
	if privKey == nil {
		return nil, errors.New("parse bytes to private key failed")
	}

	return privKey, nil
}

func checkBeforeNewKey(path string, file string) error {
	if err := utils.AccessCheck(path); err != nil {
		return err
	}

	if err := utils.AccessCheck(file); err == nil {
		return fmt.Errorf("File %s already exists."+
			"You should remove it before creating a new one in the same directory",
			file)
	}

	return nil
}

func readKeyFile(file string) ([]byte, error) {
	if err := utils.AccessCheck(file); err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	//trim the \n of content
	trimContent := strings.TrimSpace(string(content))
	return []byte(trimContent), nil
}

func saveOnDisk(content []byte, file string) error {
	err := ioutil.WriteFile(file, content, 0600)
	return err
}
