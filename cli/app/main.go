package main

import (
	"context"
	"fmt"

	"github.com/alexandre-lavoie/debugger-go/cli"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := cli.CLI(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
