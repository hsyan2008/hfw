package encoding

import "encoding/base64"

//Base64Encode 编码base64
func Base64Encode(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

//Base64Decode 解码base64
func Base64Decode(src string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(src)
}
