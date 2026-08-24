package release

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"

	protocolrelease "go.temporal.io/server/tools/umpire3/protocol/release"
)

func SignReceipt(
	receipt protocolrelease.QualificationReceipt,
	privateKey ed25519.PrivateKey,
) (protocolrelease.QualificationReceipt, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return protocolrelease.QualificationReceipt{}, errors.New("Ed25519 qualification signing key is required")
	}
	if err := receipt.ValidateUnsigned(); err != nil {
		return protocolrelease.QualificationReceipt{}, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	expectedPublicKey, err := base64.RawStdEncoding.DecodeString(receipt.Authority.PublicKey)
	if err != nil || len(expectedPublicKey) != ed25519.PublicKeySize {
		return protocolrelease.QualificationReceipt{}, errors.New("qualification authority public key is invalid")
	}
	if !bytes.Equal(publicKey, expectedPublicKey) {
		return protocolrelease.QualificationReceipt{}, errors.New("qualification signing key does not match the release authority")
	}
	payload, err := receipt.SigningPayload()
	if err != nil {
		return protocolrelease.QualificationReceipt{}, err
	}
	receipt.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	if err := receipt.Verify(receipt.Authority); err != nil {
		return protocolrelease.QualificationReceipt{}, err
	}
	return receipt, nil
}
