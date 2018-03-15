package hfw

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
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
