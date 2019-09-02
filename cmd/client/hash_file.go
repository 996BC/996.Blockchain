package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/996BC/996.Blockchain/core/merkle"
	"github.com/996BC/996.Blockchain/utils"
)

type hashFile struct {
	Name string      `json:"name"`
	Hash string      `json:"hash"` // hex of hash
	Dir  []*hashFile `json:"dir"`
}

func generateHashFile(evidence string, ignore bool) error {
	if err := utils.AccessCheck(evidence); err != nil {
		return err
	}

	hf, err := newHashFile(evidence, ignore)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(hf, "", "  ")
	if err != nil {
		return err
	}

	var fileName string
	evidenceFileInfo, err := os.Stat(evidence)
	if err != nil {
		return fmt.Errorf("stat failed:%v", err)
	}
	if evidenceFileInfo.IsDir() {
		abs, _ := filepath.Abs(evidenceFileInfo.Name())
		fileName = filepath.Base(abs)
	} else {
		fileName = evidenceFileInfo.Name()
	}

	runningDir, err := os.Getwd()
	if err != nil {
		return err
	}
	outputFile := runningDir + "/hf-" + fileName + "-" + time.Now().Format("20060102150405")

	if err := ioutil.WriteFile(outputFile, jsonBytes, 0664); err != nil {
		return err
	}

	fmt.Printf(">>> generate hash file:%s\n", outputFile)
	return nil
}

func newHashFile(evidence string, ignore bool) (*hashFile, error) {

	fileInfo, err := os.Stat(evidence)
	if err != nil {
		return nil, fmt.Errorf("access %s failed:%v", evidence, err)
	}

	result := &hashFile{
		Name: fileInfo.Name(),
	}

	//1. directory
	if fileInfo.IsDir() {
		path, err := filepath.Abs(evidence)
		if err != nil {
			return nil, fmt.Errorf("get path of %s failed", path)
		}
		result.Name = filepath.Base(path)

		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read files in directory %s failed", path)
		}

		if len(files) == 0 {
			return nil, nothingForHash{name: fileInfo.Name()}
		}

		// recursively calculates and collects the hash to get merkle root
		var leafs merkle.MerkleLeafs
		for _, file := range files {
			if ignore && file.Name()[0] == '.' {
				continue
			}

			e := path + "/" + file.Name()
			hf, err := newHashFile(e, ignore)
			if err != nil {
				if _, ok := err.(nothingForHash); ok {
					continue
				}

				return nil, err
			}
			result.Dir = append(result.Dir, hf)

			hashB, err := utils.FromHex(hf.Hash)
			if err != nil {
				return nil, err
			}
			leafs = append(leafs, hashB)
		}

		if leafs.Len() == 0 {
			return nil, nothingForHash{name: fileInfo.Name()}
		}

		hash, err := merkle.SortAndComputeRoot(leafs)
		if err != nil {
			return nil, fmt.Errorf("get %s directory hash failed: %v",
				fileInfo.Name(), err)
		}
		result.Hash = utils.ToHex(hash)

		return result, nil
	}

	//2. file
	fileContent, err := ioutil.ReadFile(evidence)
	if err != nil {
		return nil, fmt.Errorf("read %s failed", evidence)
	}
	hash := utils.Hash(fileContent)
	result.Hash = utils.ToHex(hash)

	return result, nil
}

func verifyHashFile(file string) error {
	validStr := "\"" + file + "\"" + " is valid."
	invalidStr := "\"" + file + "\"" + " is invalid (%s).\n"

	if err := utils.AccessCheck(file); err != nil {
		return err
	}

	jsonBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read hash file failed:%v", err)
	}

	hf := &hashFile{}
	if err := json.Unmarshal(jsonBytes, hf); err != nil {
		fmt.Printf(invalidStr, "parse file failed")
		return nil
	}

	err = verify(hf, nil)
	if err == nil {
		fmt.Println(validStr)
	} else {
		fmt.Printf(invalidStr, err.Error())
	}
	return nil
}

func verify(hf *hashFile, parent *hashFile) error {
	var parentName string
	if parent != nil {
		parentName = " under \"" + parent.Name + "\""
	}
	fileName := "\"" + hf.Name + "\"" + parentName

	if len(hf.Name) == 0 {
		return fmt.Errorf("empty name hashFile " + parentName)
	}
	if len(hf.Hash) == 0 {
		return fmt.Errorf("empty hash of %s ", fileName)
	}
	_, err := utils.FromHex(hf.Hash)
	if err != nil {
		return fmt.Errorf("hex decode %s failed for %s", hf.Hash, fileName)
	}

	if hf.Dir == nil {
		return nil
	}

	var leafs merkle.MerkleLeafs
	for _, subHf := range hf.Dir {
		if err := verify(subHf, hf); err != nil {
			return err
		}
		subHashB, _ := utils.FromHex(subHf.Hash)
		leafs = append(leafs, subHashB)
	}

	merkleRootB, err := merkle.SortAndComputeRoot(leafs)
	if err != nil {
		return fmt.Errorf("compute merkle root for %s failed", fileName)
	}
	merkleRoot := utils.ToHex(merkleRootB)
	if merkleRoot != hf.Hash {
		return fmt.Errorf("sub files/directories hash %s is not correspond to %s of %s", merkleRoot, hf.Hash, fileName)
	}
	return nil
}

type nothingForHash struct {
	name string
}

func (n nothingForHash) Error() string {
	return fmt.Sprintf("nothing in %s to hash", n.name)
}
