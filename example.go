package main

import (
	"fmt"
	"limiter"
)

func main() {
	limiter := limiter.NewRateLimiter()

	//add the `limiter_element_1` limiter by 100 per second
	limiter.AddElement("limiter_element_1", 100)

	//add the `limiter_element_2` limiter by 200 per second
	limiter.AddElement("limiter_element_2", 200)

	//add the `limiter_element_3` limiter by 300 per second
	limiter.AddElement("limiter_element_3", 300)

	for i := 0; i < 200; i++ {
		if limiter.Limit("limiter_element_1") == false {
			fmt.Println("Over limiter")
		} else {
			fmt.Println("Aha")
		}
	}
}
