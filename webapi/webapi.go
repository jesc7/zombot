package webapi

import (
	"context"
	"net/http"

	"github.com/jesc7/zombot/types"
)

type WebServer struct {
	srv *http.Server
}

func NewServer(ctx context.Context, cfg types.Config) (*WebServer, error) {
	mux := &http.ServeMux{}
	mux.HandleFunc("/call", fnCalls) //пропущенные звонки
	mux.HandleFunc("/zsrv", fnZSrv)  //сообщения от ZSrv
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8089",
	}

}
