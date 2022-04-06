package glitterboot

import (
	thttp "github.com/tendermint/tendermint/rpc/client/http"
)

func NewTMClient(tendermintAddr string) (*thttp.HTTP, error) {
	return thttp.New(tendermintAddr, "/websocket")
}

type TendermintClient = thttp.HTTP
