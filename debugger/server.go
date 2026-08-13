package debug

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	scada "github.com/rusl222/scada/types"
)

type Server struct {
	proxy  *Proxy
	mux    *http.ServeMux
	wwwDir string
}

func RunDebugger(br scada.Api, rootDir string, webport int) *Proxy {

	// Оборачиваем в debug-прокси
	prox := Wrap(br)

	log.Printf("-----------------\ndebug server: http://localhost:%d\n------------------\n", webport)

	s := &Server{
		proxy:  prox,
		mux:    http.NewServeMux(),
		wwwDir: rootDir,
	}
	s.routes()
	go func() { log.Fatal(s.Listen(fmt.Sprintf(":%d", webport))) }()
	return prox
}

func (s *Server) Listen(addr string) error {

	if s.proxy == nil {
		return errors.New("debug: proxy is nil")
	}

	server := &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	return server.ListenAndServe()
}

func (s *Server) routes() {

	// HTML
	s.mux.HandleFunc("/", s.handleIndex)

	// REST
	s.mux.HandleFunc("/api/list", s.handleList)
	s.mux.HandleFunc("/api/set", s.handleSet)
	s.mux.HandleFunc("/api/state", s.handleState)
}
