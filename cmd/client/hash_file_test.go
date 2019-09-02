package main

import (
	"testing"

	"github.com/996BC/996.Blockchain/core/merkle"
	"github.com/996BC/996.Blockchain/utils"
)

var hashFileTestVar = &struct {
	basePath string

	//////////////////////////////////////////////////////////////////////////////////////
	// empty.file.txt
	emptyFile    string
	sumEmptyFile string

	//////////////////////////////////////////////////////////////////////////////////////
	// empty_dir
	emptyDir string

	//////////////////////////////////////////////////////////////////////////////////////
	// normal_dir
	normalDir    string
	sumNormalDir string

	capitalistMd    string
	sumCapitalistMd string

	helloTxt    string
	sumHelloTxt string

	nestedDir    string
	sumNestedDir string

	antiTxt    string
	sumAntiTxt string

	peopleDaylyMd    string
	sumPeopleDaylyMd string

	/////////////////////////////////////////////////////////////////////////////////////
	// dir_with_empty_dir
	dirWithEmptyDir string

	/////////////////////////////////////////////////////////////////////////////////////
	// dir_with_empty_files
	dirWithEmptyFiles    string
	sumDirWithEmptyFiles string

	emptyFile1Txt string
	emptyFile2Txt string
}{
	basePath: "testdata/",

	emptyFile:    "empty_file.txt",
	sumEmptyFile: "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",

	emptyDir: "empty_dir",

	normalDir:        "normal_dir",
	capitalistMd:     "capitalist.md",
	sumCapitalistMd:  "775DF913A9E94EB84AFC4BB0483AC6CE58FB6ADD1C034EDFCFDC6B1801FA7284",
	helloTxt:         "hello.txt",
	sumHelloTxt:      "837B1E1FDD3AF39CE81B23D62A815F33F7389E427F83DD95951D82685685F286",
	nestedDir:        "nested_dir",
	antiTxt:          "anti.txt",
	sumAntiTxt:       "AA6AF54DD27F7BF9FBBCCB3B0787EC058291211560B2E5304712188816613397",
	peopleDaylyMd:    "people_daily.md",
	sumPeopleDaylyMd: "15DFAE3B586A39AC07BCFF01BACA3BCCF3BE19FA3B350E4359F9B7DDAFB0CF15",

	dirWithEmptyDir: "dir_with_empty_dir",

	dirWithEmptyFiles: "dir_with_empty_files",
	emptyFile1Txt:     "empty_file_1.txt",
	emptyFile2Txt:     "empty_file_2.txt",
}

func init() {
	tv := hashFileTestVar

	var netedDirSubHash []string
	netedDirSubHash = append(netedDirSubHash, tv.sumAntiTxt)
	netedDirSubHash = append(netedDirSubHash, tv.sumPeopleDaylyMd)
	tv.sumNestedDir = getDirHash(netedDirSubHash)

	var testHfDirSubHash []string
	testHfDirSubHash = append(testHfDirSubHash, tv.sumCapitalistMd)
	testHfDirSubHash = append(testHfDirSubHash, tv.sumHelloTxt)
	testHfDirSubHash = append(testHfDirSubHash, tv.sumNestedDir)
	tv.sumNormalDir = getDirHash(testHfDirSubHash)

	var testHfDirWithEmptyFileSubHash []string
	testHfDirWithEmptyFileSubHash = append(testHfDirWithEmptyFileSubHash, tv.sumEmptyFile)
	testHfDirWithEmptyFileSubHash = append(testHfDirWithEmptyFileSubHash, tv.sumEmptyFile)
	tv.sumDirWithEmptyFiles = getDirHash(testHfDirWithEmptyFileSubHash)
}

func getDirHash(hexSum []string) string {
	var ba [][]byte
	for _, h := range hexSum {
		b, _ := utils.FromHex(h)
		ba = append(ba, b)
	}

	resultB, _ := merkle.SortAndComputeRoot(ba)
	return utils.ToHex(resultB)
}

func hashCheck(t *testing.T, name string, expect string, result string) {
	prefix := name + " hash "
	if err := utils.TCheckString(prefix, expect, result); err != nil {
		t.Fatal(err)
	}
}

func dirCheck(t *testing.T, name string, expectedNum int, resultNum int, expectedHash string, resultHash string) {
	prefix := "number of sub hash of dir " + name + " "
	if err := utils.TCheckInt(prefix, expectedNum, resultNum); err != nil {
		t.Fatal(err)
	}

	prefix = "dif " + name + " hash "
	if err := utils.TCheckString(prefix, expectedHash, resultHash); err != nil {
		t.Fatal(err)
	}
}
func TestHfEmptyFile(t *testing.T) {
	tv := hashFileTestVar

	hf, err := newHashFile(tv.basePath+tv.emptyFile, true)
	if err != nil {
		t.Fatal(err)
	}

	if hf.Dir != nil {
		t.Fatalf("expect dir to be nil")
	}

	hashCheck(t, hf.Name, tv.sumEmptyFile, hf.Hash)
}

func TestHfEmptyDir(t *testing.T) {
	tv := hashFileTestVar

	_, err := newHashFile(tv.basePath+tv.emptyDir, true)
	if err == nil {
		t.Fatal("expect error\n")
	}
	if _, ok := err.(nothingForHash); !ok {
		t.Fatalf("unexpected err type %v\n", err)
	}
}

// test_df_dir
func TestHFDir(t *testing.T) {
	tv := hashFileTestVar

	hf, err := newHashFile(tv.basePath+tv.normalDir, true)
	if err != nil {
		t.Fatal(err)
	}

	dirCheck(t, hf.Name, 3, len(hf.Dir), tv.sumNormalDir, hf.Hash)
	for _, hf := range hf.Dir {
		if hf.Name == tv.capitalistMd {
			hashCheck(t, hf.Name, tv.sumCapitalistMd, hf.Hash)
		}

		if hf.Name == tv.helloTxt {
			hashCheck(t, hf.Name, tv.sumHelloTxt, hf.Hash)
		}

		if hf.Name == tv.nestedDir {
			dirCheck(t, tv.nestedDir, 2, len(hf.Dir), tv.sumNestedDir, hf.Hash)

			for _, hf := range hf.Dir {
				if hf.Name == tv.antiTxt {
					hashCheck(t, hf.Name, tv.sumAntiTxt, hf.Hash)
				}

				if hf.Name == tv.peopleDaylyMd {
					hashCheck(t, hf.Name, tv.sumPeopleDaylyMd, hf.Hash)
				}
			}

		}
	}
}

func TestHfDirWithEmptyDir(t *testing.T) {
	tv := hashFileTestVar

	_, err := newHashFile(tv.basePath+tv.dirWithEmptyDir, true)
	if err == nil {
		t.Fatal("expect error\n")
	}
	if _, ok := err.(nothingForHash); !ok {
		t.Fatalf("unexpected err type %v\n", err)
	}
}

func TestHfDirWithEmptyFile(t *testing.T) {
	tv := hashFileTestVar

	hf, err := newHashFile(tv.basePath+tv.dirWithEmptyFiles, true)
	if err != nil {
		t.Fatal(err)
	}

	dirCheck(t, hf.Name, 2, len(hf.Dir), tv.sumDirWithEmptyFiles, hf.Hash)
	for _, hf := range hf.Dir {
		if hf.Name == tv.emptyFile1Txt {
			hashCheck(t, hf.Name, tv.sumEmptyFile, hf.Hash)
		}

		if hf.Name == tv.emptyFile2Txt {
			hashCheck(t, hf.Name, tv.sumEmptyFile, hf.Hash)
		}
	}
}
