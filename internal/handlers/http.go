package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"id-generator/internal/lib"
)

type HttpServer struct {
	Port   int
	server *http.Server
}

func (s *HttpServer) Serve() error {
	if s.server != nil {
		return fmt.Errorf("http server is already running")
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: s.getHandler(),
	}

	log.Printf("http server listening at %v", s.Port)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	} else {
		return fmt.Errorf("failed to start http server: %v", err)
	}
}

func (s *HttpServer) Stop(stopCh chan struct{}, done func()) {
	defer done()

	<-stopCh

	if s.server == nil {
		panic("can't stop non-exist http server")
	}

	err := s.server.Shutdown(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to shutdown http server: %v", err))
	}
}

func (s *HttpServer) getHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/get-unique-id", getUniqueId)

	return mux
}

func getUniqueId(res http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	sysType := query.Get("sys_type")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	newId, err := lib.GetUniqueId(ctx, sysType)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(fmt.Sprintf("error while generating new unique id: %v", err)))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(newId))
}
