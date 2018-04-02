package hfw

import (
	"bytes"
	"errors"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/hfw2/common"
)

func TomlSave(file string, c interface{}) error {
	var buf bytes.Buffer
	e := toml.NewEncoder(&buf)
	err := e.Encode(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

func TomlLoad(file string, c interface{}) (err error) {
	if common.IsExist(file) {
		_, err := toml.DecodeFile(file, c)
		return err
	}

	return errors.New(file + " not exist")
}
