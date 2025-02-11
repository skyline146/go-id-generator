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
	Port int
}

func (s *HttpServer) Serve() error {
	http.HandleFunc("/get-unique-id", getUniqueId)

	log.Printf("http server listening at %v", s.Port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
	if err != nil {
		return fmt.Errorf("failed to start http server: %v", err)
	}

	return nil
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
