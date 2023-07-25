package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kroma-network/kroma-prover-proxy/internal/ec2"
	"github.com/kroma-network/kroma-prover-proxy/internal/proof"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "kroma-proof-proxy"
	app.Version = "0.0.1"
	app.Flags = AllFlags()
	app.Action = proverProxy
	if err := app.Run(os.Args); err != nil {
		log.Panicln(fmt.Errorf("failed to start kroma proof proxy: %w", err))
	}
}

func proverProxy(ctx *cli.Context) {
	lis, err := net.Listen("tcp", net.JoinHostPort(ctx.String(JsonRpcAddr.Name), strconv.Itoa(ctx.Int(JsonRpcPort.Name))))
	if err != nil {
		log.Panicln(fmt.Errorf("failed to listen: %w", err))
	}
	proverServer := newServer(ctx)
	go func() {
		if err := http.Serve(lis, proverServer); err != nil {
			log.Panicln(fmt.Errorf("failed to serve: %w", err))
		}
	}()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel
	proverServer.Close()
	if err := lis.Close(); err != nil {
		log.Println(fmt.Errorf("failed to close tcp %w", err).Error())
	}
}

func newServer(ctx *cli.Context) *proof.Server {
	return proof.NewServer(
		proof.NewService(
			proof.NewDiskRepository(ctx.String(ProofBaseDir.Name)),
			ec2.MustNewController(
				ctx.String(AwsRegion.Name),
				ctx.String(AwsProverInstanceId.Name),
				ctx.String(AwsProverAddressType.Name),
			),
		),
	)
}
