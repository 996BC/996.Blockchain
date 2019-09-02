package rpc

import (
	"context"
	"net/http"
	"strconv"

	"github.com/996BC/996.Blockchain/core"
	"github.com/996BC/996.Blockchain/utils"
)

var logger = utils.NewLogger("http")

const (
	// LocalHost "127.0.0.1"
	LocalHost = "127.0.0.1"
	// DefaultHTTPPort 23666
	DefaultHTTPPort = 23666

	version1Path  = "/v1"
	version2Path  = "/v2"
	GetRangeParam = "range"
	GetHashParam  = "hash"
	GetIDParam    = "id"
)

type Config struct {
	Port int
	C    *core.Core
}

// Server is a http server provides interfaces for querying,uploading evidence and so on;
// it only listens on 127.0.0.1
type Server struct {
	*http.Server
	c *core.Core
}

var globalSvr *Server

type HTTPHandlers = []struct {
	Path string
	F    func(http.ResponseWriter, *http.Request)
}

func NewServer(conf *Config) *Server {
	sMux := http.NewServeMux()
	// evidence
	for _, handler := range evidenceHandlers {
		sMux.HandleFunc(handler.Path, handler.F)
	}
	// block
	for _, handler := range blockHandler {
		sMux.HandleFunc(handler.Path, handler.F)
	}
	// account
	for _, handler := range accountHandlers {
		sMux.HandleFunc(handler.Path, handler.F)
	}

	//default handler
	sMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	globalSvr = &Server{
		&http.Server{
			Addr:    LocalHost + ":" + strconv.Itoa(conf.Port),
			Handler: sMux,
		},
		conf.C,
	}

	return globalSvr
}

func (s *Server) Start() {
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal("Http server listen failed:%v\n", err)
		}
	}()
}

func (s *Server) Stop() {
	if err := s.Shutdown(context.Background()); err != nil {
		logger.Warn("HTTP server shutdown err:%v\n", err)
	}
}
