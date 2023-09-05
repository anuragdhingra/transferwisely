package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"net/http"
	"strconv"
	"time"
)

func main() {
	_, err := strconv.ParseUint(intervalVar, 10, 64)
	if err != nil {
		fmt.Printf("Invalid value for INTERVAL: %v", err)
		return
	}

	s1 := gocron.NewScheduler(time.UTC)
	_, err = s1.Every(2).Minute().Do(checkAndProcess)
	if err != nil {
		fmt.Println(err.Error())
		panic("couldn't initiate the checkAndProcess job")
	}
	//s1.Every(12).Hours().Do(sendExpiryReminderMail)
	s1.StartAsync()

	fmt.Println("Starting batch server on port 3000")
	_ = http.ListenAndServe(":3000", nil)
}
