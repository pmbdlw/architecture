package authserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
)

// loadKey 读取 PKCS#8 / PKCS#1 PEM 私钥；未配置时生成临时密钥（仅限本地开发）。
func loadKey(file string) (*rsa.PrivateKey, bool, error) {
	if file == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		return key, true, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, false, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, false, errors.New("invalid PEM: " + file)
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, false, nil
		}
		return nil, false, errors.New("not an RSA key: " + file)
	}
	k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return k, false, err
}

// jwk 公钥的 JWK 表示（/oauth2/jwks）。
func jwk(pub *rsa.PublicKey, kid string) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
