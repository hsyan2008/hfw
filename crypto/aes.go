package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"

	"github.com/hsyan2008/hfw/encoding"
)

//AesCrypt aes加解密
type AesCrypt struct {
	//16、24、32长度
	key []byte
	//cbc、ecb两种模式
	model string
}

//NewAesCrypt 默认是cbc模式
func NewAesCrypt(key string) *AesCrypt {
	return &AesCrypt{
		key:   getKey(key),
		model: "cbc",
	}
}

//SetEcbModel ..
func (aesCrypt *AesCrypt) SetEcbModel() {
	aesCrypt.model = "ecb"
}

//Encrypt 加密
func (aesCrypt *AesCrypt) Encrypt(plantText []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesCrypt.key) //选择加密算法
	if err != nil {
		return nil, err
	}
	plantText = aesCrypt.PKCS7Padding(plantText, block.BlockSize())

	var blockModel cipher.BlockMode
	if aesCrypt.model == "cbc" {
		blockModel = cipher.NewCBCEncrypter(block, aesCrypt.key)
	} else if aesCrypt.model == "ecb" {
		blockModel = cipher.NewCBCEncrypter(block, bytes.Repeat([]byte{0}, block.BlockSize()))
	}

	ciphertext := make([]byte, len(plantText))

	blockModel.CryptBlocks(ciphertext, plantText)
	return ciphertext, nil
}

//Encrypt2Base64 加密结果转为base64
func (aesCrypt *AesCrypt) Encrypt2Base64(plantText string) (string, error) {
	ac, err := aesCrypt.Encrypt([]byte(plantText))
	if err != nil {
		return "", err
	}
	return encoding.Base64Encode(ac), nil
}

//PKCS7Padding ..
func (aesCrypt *AesCrypt) PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//Decrypt 解密
func (aesCrypt *AesCrypt) Decrypt(ciphertext []byte) ([]byte, error) {
	keyBytes := []byte(aesCrypt.key)
	block, err := aes.NewCipher(keyBytes) //选择加密算法
	if err != nil {
		return nil, err
	}
	var blockModel cipher.BlockMode
	if aesCrypt.model == "cbc" {
		blockModel = cipher.NewCBCDecrypter(block, keyBytes)
	} else if aesCrypt.model == "ecb" {
		blockModel = cipher.NewCBCDecrypter(block, bytes.Repeat([]byte{0}, block.BlockSize()))
	}
	plantText := make([]byte, len(ciphertext))
	blockModel.CryptBlocks(plantText, ciphertext)
	plantText = aesCrypt.PKCS7UnPadding(plantText, block.BlockSize())
	return plantText, nil
}

//Baes642Decrypt 解密base64格式的密文
func (aesCrypt *AesCrypt) Baes642Decrypt(ciphertext string) (string, error) {
	ubase, err := encoding.Base64Decode(ciphertext)
	if err != nil {
		return "", err
	}
	pass, err := aesCrypt.Decrypt(ubase)
	if err != nil {
		return "", err
	}
	return string(pass), nil
}

//PKCS7UnPadding ..
func (aesCrypt *AesCrypt) PKCS7UnPadding(plantText []byte, blockSize int) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	return plantText[:(length - unpadding)]
}

func getKey(key string) []byte {
	keyLen := len(key)
	if keyLen < 16 {
		panic("res key 长度不能小于16")
	}
	arrKey := []byte(key)
	if keyLen >= 32 {
		//取前32个字节
		return arrKey[:32]
	}
	if keyLen >= 24 {
		//取前24个字节
		return arrKey[:24]
	}
	//取前16个字节
	return arrKey[:16]
}
