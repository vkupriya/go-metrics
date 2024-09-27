package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
)

const filePermissions fs.FileMode = 0o600
const rsaPrivateKeyBits = 2048

var path = "../../keys/"

func main() {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaPrivateKeyBits)
	if err != nil {
		panic(err)
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	err = os.WriteFile(path+"private.pem", privateKeyPEM, filePermissions)
	if err != nil {
		panic(err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	err = os.WriteFile(path+"public.pem", publicKeyPEM, filePermissions)
	if err != nil {
		panic(err)
	}

	fmt.Println("key pair generated successfully.")
}
