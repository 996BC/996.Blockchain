package crypto

import (
	"log"
	"os"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

var pkeyTestVar = &struct {
	runningDir string
	savedStdin *os.File
	password   string
}{
	savedStdin: os.Stdin,
	password:   "test_password",
}

func init() {
	var err error
	if pkeyTestVar.runningDir, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}
}

func cleanup() {
	tv := pkeyTestVar

	os.Remove(tv.runningDir + "/" + SealKey)
	os.Remove(tv.runningDir + "/" + PlainKey)
	os.Stdin = tv.savedStdin
}

func TestNewPKey(t *testing.T) {
	defer cleanup()

	_, err := NewPKey(pkeyTestVar.runningDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenSKey(t *testing.T) {
	defer cleanup()
	tv := pkeyTestVar

	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdin = pipeReader
	pipeWriter.WriteString(tv.password + "\n" + tv.password + "\n")
	sKey, err := NewSKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	pipeWriter.WriteString(tv.password + "\n")
	if err := OpenSKey(tv.runningDir, tv.runningDir); err != nil {
		t.Fatal(err)
	}

	pKey, err := RestorePKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("pKey from sKey", sKey.Serialize(), pKey.Serialize()); err != nil {
		t.Fatal(err)
	}
}

func TestRestorePKey(t *testing.T) {
	defer cleanup()
	tv := pkeyTestVar

	generatedKey, err := NewPKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	restoreKey, err := RestorePKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("restored pKey", generatedKey.Serialize(), restoreKey.Serialize()); err != nil {
		t.Fatal(err)
	}
}
