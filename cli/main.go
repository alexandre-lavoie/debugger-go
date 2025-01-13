package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandre-lavoie/debugger-go/core"
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

func debug(ctx context.Context, dbg core.StaticDebugger) error {
	fmt.Println("HERE!")

	return nil
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

	_, err = g.AddSoftwareBreakpoint(ctx, 0x12e62, debug)
	if err != nil {
		return err
	}

	if err := g.Continue(ctx); err != nil {
		return err
	}

	if err := g.Run(ctx); err != nil {
		return err
	}

	return nil
}
