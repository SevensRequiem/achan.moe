package tests

// open 50 connections to the web server and send requests and print the response adn response time
import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func Test() {
	var wg sync.WaitGroup
	client := &http.Client{}

	// Open 50 connections
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := time.Now()
			resp, err := client.Get("https://dev123.achan.moe") // Change to your server URL
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()
			duration := time.Since(start)
			fmt.Printf("Response from server: %s, Duration: %v\n", resp.Status, duration)
		}(i)
	}

	wg.Wait()
}
