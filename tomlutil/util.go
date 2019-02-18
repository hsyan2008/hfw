package tomlutil

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

func Save(file string, c interface{}) error {
	var buf bytes.Buffer
	e := toml.NewEncoder(&buf)
	err := e.Encode(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

func Load(file string, c interface{}) (err error) {
	_, err = toml.DecodeFile(file, c)
	return
}
