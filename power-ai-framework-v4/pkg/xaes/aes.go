package xaes

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

// EncryptCBC 加密
func EncryptCBC(content, secretKey string) (s string, err error) {
	return EncryptCBCByte([]byte(content), secretKey)
}

// EncryptCBCByte 加密
func EncryptCBCByte(origData []byte, secretKey string) (s string, err error) {

	key, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	origData = pkcs5Padding(origData, blockSize)                // 补全码
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize]) // 加密模式
	encrypted := make([]byte, len(origData))                    // 创建数组
	blockMode.CryptBlocks(encrypted, origData)                  // 加密
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptCBC 解密
func DecryptCBC(content string, secretKey string) (decrypted []byte, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("UnKnow panic")
			}
		}
	}()

	key, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		return nil, err
	}
	encrypted, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key) // 分组秘钥
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()                              // 获取秘钥块的长度
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize]) // 加密模式
	decrypted = make([]byte, len(encrypted))                    // 创建数组
	blockMode.CryptBlocks(decrypted, encrypted)
	decrypted = pkcs5UnPadding(decrypted, blockSize) // 去除补全码
	return decrypted, nil
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	bys := make([]byte, padding)
	for i := range bys {
		bys[i] = 0
	}
	return append(ciphertext, bys...)
}
func pkcs5UnPadding(origData []byte, blockSize int) []byte {
	length := len(origData)
	for i := length - 1; i >= 0; i-- {
		if origData[i] == 0 {
			origData = origData[:len(origData)-1]
		} else {
			break
		}
	}
	return origData
}
