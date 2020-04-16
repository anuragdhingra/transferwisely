package main

import (
    "fmt"
    "github.com/go-co-op/gocron"
    "net/http"
    "os"
    "strconv"
    "time"
)

func main() {
    interval, _ := strconv.ParseUint(os.Getenv("INTERVAL"), 10, 64)

    s1 := gocron.NewScheduler(time.UTC)
    s1.Every(interval).Minutes().Do(checkAndProcess)
    s1.Start()

    fmt.Println("Starting batch server on port 3000")
    _ = http.ListenAndServe(":3000", nil)
}