package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := cli(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func cli(ctx context.Context) error {
	g, err := gdb.Remote(ctx, "localhost", 1234)
	if err != nil {
		return err
	}
	defer g.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		g.Interrupt(ctx)
	}()

	if err := g.Continue(ctx); err != nil {
		return err
	}

	if err := g.Wait(ctx); err != nil {
		return err
	}

	return nil
}
