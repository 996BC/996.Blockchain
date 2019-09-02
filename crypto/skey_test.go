package crypto

import (
	"log"
	"os"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

var skeyTestVar = &struct {
	runningDir string
	password   string
}{
	password: "test_password",
}

func init() {
	var err error
	if skeyTestVar.runningDir, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}
}

func TestNewSKey(t *testing.T) {
	defer cleanup()
	tv := skeyTestVar

	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdin = pipeReader

	pipeWriter.WriteString(tv.password + "\n" + tv.password + "error" + "\n")
	_, err := NewSKey(tv.runningDir)
	if err == nil {
		t.Fatalf("Expect inconsistent input error")
	}

	pipeWriter.WriteString(tv.password + "\n" + tv.password + "\n")
	_, err = NewSKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSealPKey(t *testing.T) {
	defer cleanup()
	tv := skeyTestVar

	pKey, err := NewPKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdin = pipeReader
	pipeWriter.WriteString(tv.password + "\n" + tv.password + "\n")
	if err := SealPKey(tv.runningDir, tv.runningDir); err != nil {
		t.Fatal(err)
	}

	pipeWriter.WriteString(tv.password + "\n")
	sKey, err := RestoreSKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("SealPKey", pKey.Serialize(), sKey.Serialize()); err != nil {
		t.Fatal(err)
	}
}

func TestReNewSKey(t *testing.T) {
	defer cleanup()
	tv := skeyTestVar

	tmpDir := tv.runningDir + "/.tmp"
	if err := os.Mkdir(tmpDir, 0700); err != nil {
		t.Fatal(err)
	}
	defer func() { os.RemoveAll(tmpDir) }()

	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdin = pipeReader

	pipeWriter.WriteString(tv.password + "\n" + tv.password + "\n")
	oldPrivateKey, err := NewSKey(tv.runningDir)
	if err != nil {
		t.Error(err)
	}

	// generate new sKey from the existing sKey and restore it
	newPassWord := "new_test_password"
	pipeWriter.WriteString(tv.password + "\n")
	pipeWriter.WriteString(newPassWord + "\n" + newPassWord + "\n")
	err = ReNewSKey(tv.runningDir, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	pipeWriter.WriteString(newPassWord + "\n" + newPassWord + "\n")
	newPrivKey, err := RestoreSKey(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("ReNewSKey", oldPrivateKey.Serialize(), newPrivKey.Serialize()); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreSKey(t *testing.T) {
	defer cleanup()
	tv := skeyTestVar

	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdin = pipeReader
	pipeWriter.WriteString(tv.password + "\n" + tv.password + "\n")

	generatedKey, err := NewSKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	pipeWriter.WriteString(tv.password + "error" + "\n")
	_, err = RestoreSKey(tv.runningDir)
	if err == nil {
		t.Fatal("Expect restore failed cause error passphase input")
	}

	pipeWriter.WriteString(tv.password + "\n")
	restoredKey, err := RestoreSKey(tv.runningDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("RestoreSKey", generatedKey.Serialize(), restoredKey.Serialize()); err != nil {
		t.Fatal(err)
	}
}
