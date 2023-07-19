package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"google.golang.org/grpc"

	"github.com/kroma-network/kroma-prover-grpc-proto/prover"
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
	s, proverServer := grpc.NewServer(), newServer(ctx)
	prover.RegisterProverServer(s, proverServer)
	lis, err := net.Listen("tcp", net.JoinHostPort(ctx.String(GRPCAddr.Name), strconv.Itoa(ctx.Int(GRPCPort.Name))))
	if err != nil {
		log.Panicln(fmt.Errorf("failed to listen: %w", err))
	}
	go func() {
		if err := s.Serve(lis); err != nil {
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
	s.Stop()
}

func newServer(ctx *cli.Context) *proof.Server {
	return proof.NewServer(
		proof.NewDiskRepository(ctx.String(ProofBaseDir.Name)),
		ec2.MustNewController(
			ctx.String(AwsRegion.Name),
			ctx.String(AwsProverInstanceId.Name),
		),
	)
}
