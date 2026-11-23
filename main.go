package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Configs struct {
	StatsHost   string
	StatsPort   int
	CommandHost string
	CommandPort int
}

const defaultHost string = "127.0.0.1"
const defaultPort int = 60000

func main() {

	host := flag.String("--host", defaultHost, "host address for command and stats listener")
	statsport := flag.Int("--statsport", defaultPort, "port to listen for stats on")
	commandport := flag.Int("--commandport", defaultPort+1, "port to listen for commands on")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()

	//setup listening hosts and ports
	var configs = Configs{StatsHost: *host, StatsPort: *statsport, CommandHost: *host, CommandPort: *commandport}

	//setup stats (sync.Map)
	stats := &sync.Map{}

	//start StatsListener and CommandListener by passing host, port, map, and context
	go StatsListener(configs.StatsHost, configs.StatsPort, stats, ctx)
	go CommandListener(configs.CommandHost, configs.CommandPort, stats, ctx)

}
