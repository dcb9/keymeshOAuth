package proxy

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dcb9/keymeshOAuth/crypto"
	"github.com/dcb9/keymeshOAuth/db"
)

var ErrEmptyEmail = errors.New("email could not be empty")

func HandlePutAccountInfo(requestBody string) (err error) {
	var info *db.AccountInfo
	err = json.Unmarshal([]byte(requestBody), &info)
	if err != nil {
		return
	}

	if info.Email == "" {
		return ErrEmptyEmail
	}

	if info.Sig != "" {
		info.ValidSig = crypto.VerifySig(info.UserAddress, info.Sig, []byte(info.Msg))
	}
	if info.UserAddress == "" {
		info.UserAddress = "-"
	}

	fmt.Printf("%#v\n", info)
	_, err = db.PutAccountInfo(*info)

	return
}
