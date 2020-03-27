package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

func main() {

	g := errgroup.Group{}

	for {

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
		c1 := 0
		g.Go(func() error {
			c1 = cpuLoop(ctx)
			if c1 < 0 {
				return fmt.Errorf("c1 failed")
			}
			return nil
		})

		c2 := 0
		g.Go(func() error {
			c2 = cpuLoop(ctx)
			if c2 < 0 {
				return fmt.Errorf("c2 failed")
			}
			return nil
		})

		c3 := 0
		g.Go(func() error {
			c3 = cpuLoop(ctx)
			if c3 < 0 {
				return fmt.Errorf("c3 failed: %d", c3)
			}
			return nil
		})

		c4 := 0
		g.Go(func() error {
			c4 = cpuLoop(ctx)
			if c4 < 0 {
				return fmt.Errorf("c3 failed: %d", c3)
			}
			return nil
		})

		if err := g.Wait(); err != nil {
			cancel()
			panic("Abort")
		}

		cancel()
		fmt.Println("Total count:", c1, c2, c3, c4, c1+c2+c3+c4)
		time.Sleep(1 * time.Second)

	}

}

func cpuLoop(ctx context.Context) int {
	i := -1

	for {
		select {
		case <-ctx.Done():
			return i
		default:
			i = i + 1
		}
	}

}
