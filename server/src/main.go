package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	fmt.Println("hello,world?")
	ctx, cancel := context.WithCancel(context.TODO())
	for i := 1; i <= 1; i++ {
		go func(k int, ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("%d exited\n", k)
					return
				default:
					fmt.Printf("%d fuck!\n", k)
				}
			}
		}(i, ctx)
	}
	time.Sleep(5 * time.Second)
	cancel()
	time.Sleep(5 * time.Second)
}
