package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

type keyConfig struct {
	Type int    `json:"type"`
	Path string `json:"path"`
}

type config struct {
	ServerIP     string     `json:"server_ip"`
	ServerPort   int        `json:"server_port"`
	Scheme       string     `json:"scheme"`
	IgnoreHidden int        `json:"ignore_hidden"`
	Key          *keyConfig `json:"key"`
	Difficulty   string     `json:"hash_diff"`
}

func parseConfig(file string) (*config, error) {
	if err := utils.AccessCheck(file); err != nil {
		return nil, err
	}

	confB, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read config file failed:%v", err)
	}

	conf := &config{}
	if err := json.Unmarshal(confB, conf); err != nil {
		return nil, fmt.Errorf("parse json format failed:%v", err)
	}

	if err := verifyConfig(conf); err != nil {
		return nil, fmt.Errorf("verify failed:%v", err)
	}

	return conf, nil
}

func verifyConfig(c *config) error {
	if ip := net.ParseIP(c.ServerIP); ip == nil {
		return fmt.Errorf("invald ip")
	}

	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid port")
	}

	if c.Scheme != "http" && c.Scheme != "https" {
		return fmt.Errorf("invalid protocol")
	}

	if c.Key.Type != crypto.SealKeyType && c.Key.Type != crypto.PlainKeyType {
		return fmt.Errorf("invalid key type")
	}

	if err := utils.AccessCheck(c.Key.Path); err != nil {
		return err
	}

	if len(c.Difficulty) == 0 {
		return fmt.Errorf("null difficulty")
	}

	return nil
}
