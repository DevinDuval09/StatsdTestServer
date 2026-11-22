package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	//setup listening hosts and ports
	//setup stats (sync.Map)
	//start StatsListener and CommandListener by passing host, port, map, and context
}
