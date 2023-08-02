package main

import (
	"github.com/urfave/cli"
)

var (
	JsonRpcAddr = cli.StringFlag{
		Name:   "jsonrpc.addr",
		Usage:  "Json Rpc server listening address",
		Value:  "localhost",
		EnvVar: "JSONRPC_ADDR",
	}
	JsonRpcPort = cli.IntFlag{
		Name:   "jsonrpc.port",
		Usage:  "Json Rpc server listening port",
		Value:  6000,
		EnvVar: "JSONRPC_PORT",
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
	AwsProverAddressType = cli.StringFlag{
		Name:   "aws.prover-address-type",
		Usage:  "EC instance address type (private, public)",
		Value:  "private",
		EnvVar: "AWS_PROVER_ADDRESS_TYPE",
	}
	AwsProverUrlSchema = cli.StringFlag{
		Name:   "aws.prover-url-schema",
		Usage:  "http, https",
		Value:  "http",
		EnvVar: "AWS_PROVER_URL_SCHEMA",
	}
	AwsProverJsonRpcPort = cli.IntFlag{
		Name:   "aws.prover-jsonrpc-port",
		Usage:  "jsonrpc port",
		Value:  3030,
		EnvVar: "AWS_PROVER_JSONRPC_PORT",
	}
)

func AllFlags() []cli.Flag {
	return []cli.Flag{
		JsonRpcAddr,
		JsonRpcPort,
		ProofBaseDir,
		AwsRegion,
		AwsProverInstanceId,
		AwsProverAddressType,
		AwsProverUrlSchema,
		AwsProverJsonRpcPort,
	}
}
