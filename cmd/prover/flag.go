package main

import (
	"github.com/urfave/cli"
)

var (
	GRPCAddr = cli.StringFlag{
		Name:   "grpc.addr",
		Usage:  "GRPC server listening address",
		Value:  "localhost",
		EnvVar: "GRPC_ADDR",
	}
	GRPCPort = cli.IntFlag{
		Name:   "grpc.port",
		Usage:  "GRPC server listening port",
		Value:  6000,
		EnvVar: "GRPC_PORT",
	}
	ProofBaseDir = cli.StringFlag{
		Name:   "proof.base-dir",
		Usage:  "A directory to temporarily store the generated proof",
		Value:  "./proof",
		EnvVar: "PROOF_BASE_DIR",
	}
	AwsRegion = cli.StringFlag{
		Name:   "aws.region",
		Value:  "ap-northeast-2",
		EnvVar: "AWS_REGION",
	}
	AwsProverInstanceId = cli.StringFlag{
		Name:     "aws.prover-instance-id",
		Usage:    "EC instance ID to generate the proof",
		EnvVar:   "AWS_PROVER_INSTANCE_ID",
		Required: true,
	}
)

func AllFlags() []cli.Flag {
	return []cli.Flag{
		GRPCAddr,
		GRPCPort,
		ProofBaseDir,
		AwsRegion,
		AwsProverInstanceId,
	}
}
