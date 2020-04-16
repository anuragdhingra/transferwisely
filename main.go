package main

import (
    "encoding/json"
    "fmt"
    "github.com/go-chi/chi"
    "github.com/go-chi/chi/middleware"
    "log"
    "net/http"
)

func main() {
    r := chi.NewRouter()
    r.Use(middleware.Logger)

    fmt.Println("Starting server on port 3000")
    r.Get("/check", Check)
    _ = http.ListenAndServe(":3000", r)
}

func Check(w http.ResponseWriter, r *http.Request)  {
    transferId, err := checkAndProcess()
    w.Header().Set("Content-Type", "application/json")

    if err != nil {
        log.Println(err)
        w.WriteHeader(http.StatusInternalServerError)
        response := errorResponse{http.StatusInternalServerError, err.Error()}
        jsonResponse, _ := json.Marshal(response)
        _, _ = w.Write(jsonResponse)
        return
    }


    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, transferId)
}

type errorResponse struct {
    Code        int     `json:"code"`
    Message     string  `json:"message"`
}