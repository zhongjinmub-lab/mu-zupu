package payment

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"sort"
	"strings"
)

// parseRSAPrivateKey 支持 PKCS1 与 PKCS8 的 PEM,以及无 PEM 头的裸 base64(可能含换行)。
func parseRSAPrivateKey(s string) (*rsa.PrivateKey, error) {
	der, err := decodeKeyMaterial(s)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, errors.New("invalid RSA private key")
	}
	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

// parseRSAPublicKey 支持 PKIX 与 PKCS1 的 PEM,以及无 PEM 头的裸 base64。
func parseRSAPublicKey(s string) (*rsa.PublicKey, error) {
	der, err := decodeKeyMaterial(s)
	if err != nil {
		return nil, err
	}
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		key, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return key, nil
	}
	if key, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return key, nil
	}
	return nil, errors.New("invalid RSA public key")
}

func decodeKeyMaterial(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty key material")
	}
	if strings.Contains(s, "-----BEGIN") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, errors.New("invalid PEM block")
		}
		return block.Bytes, nil
	}
	clean := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "").Replace(s)
	der, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return nil, errors.New("invalid base64 key material")
	}
	return der, nil
}

// signRSA2 使用 SHA256withRSA(PKCS1v15)签名,返回 base64 字符串。
func signRSA2(priv *rsa.PrivateKey, content string) (string, error) {
	digest := sha256.Sum256([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// verifyRSA2 校验 SHA256withRSA(PKCS1v15)签名。
func verifyRSA2(pub *rsa.PublicKey, content, signatureB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil {
		return ErrInvalidSignature
	}
	digest := sha256.Sum256([]byte(content))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return ErrInvalidSignature
	}
	return nil
}

// buildSignContent 按 key 字典序拼接 "k=v&k=v",跳过空值与 sign。
// excludeSignType 为 true 时同时跳过 sign_type(用于支付宝异步通知/响应验签);
// 为 false 时保留 sign_type(用于支付宝请求签名)。
func buildSignContent(params map[string]string, excludeSignType bool) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" {
			continue
		}
		if excludeSignType && k == "sign_type" {
			continue
		}
		if strings.TrimSpace(v) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

// decryptAESGCM 使用 AES-256-GCM 解密(微信支付 v3 回调资源体的 AEAD_AES_256_GCM)。
// key 必须为 32 字节;nonce 为 12 字节;ciphertext 为密文与 16 字节认证标签拼接后的字节(已 base64 解码);
// associatedData 为附加认证数据(可为空)。
func decryptAESGCM(key, nonce, ciphertext, associatedData []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("aes-256-gcm key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid aes-gcm nonce size")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, ErrInvalidNotify
	}
	return plaintext, nil
}
