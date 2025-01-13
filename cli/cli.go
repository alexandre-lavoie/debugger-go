package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alexandre-lavoie/debugger-go/core"
	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func debug(ctx context.Context, dbg core.StaticDebugger) error {
	regs, err := dbg.ReadRegisters(ctx)
	if err != nil {
		return err
	}

	parts := make([]string, len(regs))

	for i, reg := range regs {
		parts[i] = fmt.Sprintf("%s(%s): %s", reg.Definition.Name, reg.Definition.Type, reg.ToString())
	}

	fmt.Println(strings.Join(parts, ", "))

	return errors.New("done")
}

func CLI(ctx context.Context) error {
	g, err := gdb.Remote(ctx, "localhost", 1234)
	if err != nil {
		return err
	}
	defer g.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-c
			g.Interrupt(ctx)
		}
	}()

	_, err = g.AddInterruptBreakpoint(ctx, debug)
	if err != nil {
		return err
	}

	if err := g.Run(ctx); err != nil {
		return err
	}

	return nil
}
