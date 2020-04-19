package main

import (
    "fmt"
    "github.com/go-co-op/gocron"
    "net/http"
    "strconv"
    "time"
)

func main() {
    interval, err := strconv.ParseUint(intervalVar, 10, 64)
    if err != nil {
        fmt.Printf("Invalid value for INTERVAL: %v", err)
        return
    }

    s1 := gocron.NewScheduler(time.UTC)
    s1.Every(interval).Minutes().Do(checkAndProcess)
    s1.Start()

    fmt.Println("Starting batch server on port 3000")
    _ = http.ListenAndServe(":3000", nil)
}