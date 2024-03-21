// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/Pond-International/go-ethereum/common"
	"github.com/Pond-International/go-ethereum/common/hexutil"
	"github.com/Pond-International/go-ethereum/common/math"
)

var _ = (*accountMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (a Account) MarshalJSON() ([]byte, error) {
	type Account struct {
		Code       hexutil.Bytes               `json:"code,omitempty"`
		Storage    map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance    *math.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce      math.HexOrDecimal64         `json:"nonce,omitempty"`
		PrivateKey hexutil.Bytes               `json:"secretKey,omitempty"`
	}
	var enc Account
	enc.Code = a.Code
	if a.Storage != nil {
		enc.Storage = make(map[storageJSON]storageJSON, len(a.Storage))
		for k, v := range a.Storage {
			enc.Storage[storageJSON(k)] = storageJSON(v)
		}
	}
	enc.Balance = (*math.HexOrDecimal256)(a.Balance)
	enc.Nonce = math.HexOrDecimal64(a.Nonce)
	enc.PrivateKey = a.PrivateKey
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *Account) UnmarshalJSON(input []byte) error {
	type Account struct {
		Code       *hexutil.Bytes              `json:"code,omitempty"`
		Storage    map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance    *math.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce      *math.HexOrDecimal64        `json:"nonce,omitempty"`
		PrivateKey *hexutil.Bytes              `json:"secretKey,omitempty"`
	}
	var dec Account
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Code != nil {
		a.Code = *dec.Code
	}
	if dec.Storage != nil {
		a.Storage = make(map[common.Hash]common.Hash, len(dec.Storage))
		for k, v := range dec.Storage {
			a.Storage[common.Hash(k)] = common.Hash(v)
		}
	}
	if dec.Balance == nil {
		return errors.New("missing required field 'balance' for Account")
	}
	a.Balance = (*big.Int)(dec.Balance)
	if dec.Nonce != nil {
		a.Nonce = uint64(*dec.Nonce)
	}
	if dec.PrivateKey != nil {
		a.PrivateKey = *dec.PrivateKey
	}
	return nil
}
