package encoding

import (
	"bytes"
	"encoding/gob"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

func init() {
	//https://blog.csdn.net/impressionw/article/details/74731888
	extra.RegisterFuzzyDecoders()
}

//Codec 参考https://github.com/golang/appengine/blob/master/memcache/memcache.go
type Codec struct {
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}

var (
	// Gob is a Codec that uses the gob package.
	Gob = Codec{gobMarshal, gobUnmarshal}
	// JSON is a Codec that uses the json package.
	// JSON = Codec{json.Marshal, json.Unmarshal}
	JSON = jsoniter.ConfigCompatibleWithStandardLibrary
)

func gobMarshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobUnmarshal(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

type CodecIO struct {
	Marshal   func(io.Writer, interface{}) error
	Unmarshal func(io.Reader, interface{}) error
}

var (
	GobIO  = CodecIO{gobWriterMarshal, gobReaderUnmarshal}
	JSONIO = CodecIO{jsonWriterMarshal, jsonReaderUnmarshal}
)

//以下直接针对r/w操作
func gobWriterMarshal(w io.Writer, data interface{}) (err error) {
	return gob.NewEncoder(w).Encode(data)
}

func gobReaderUnmarshal(r io.Reader, data interface{}) (err error) {
	return gob.NewDecoder(r).Decode(data)
}

func jsonWriterMarshal(w io.Writer, data interface{}) (err error) {
	enc := jsoniter.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}
func jsonReaderUnmarshal(r io.Reader, data interface{}) (err error) {
	return jsoniter.NewDecoder(r).Decode(data)
}
