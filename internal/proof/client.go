package proof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ProverClient interface {
	Prove(traceString string, proofType Type) (*ProveResponse, error)
	Spec() (*ProverSpecResponse, error)
}

func NewProverClient(address string) (ProverClient, error) {
	return &dialJsonRpcProverClient{address}, nil
}

type dialJsonRpcProverClient struct {
	address string
}

type request struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	Id      string `json:"id"`
}

type response[T any] struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  *T     `json:"result"`
	Error   any    `json:"error"`
	Id      string `json:"id"`
}

func (d dialJsonRpcProverClient) Prove(traceString string, proofType Type) (*ProveResponse, error) {
	return send[ProveResponse](d.address, "prove", []any{traceString, proofType})
}

func (d dialJsonRpcProverClient) Spec() (*ProverSpecResponse, error) {
	return send[ProverSpecResponse](d.address, "spec", nil)
}

func send[T any](address string, method string, params any) (*T, error) {
	request := request{"2.0", method, params, "0"}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Panicln(fmt.Errorf("failed to json.Marshal %w", err))
	}
	httpResponse, err := http.Post(address, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	jsonBytes, err = io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}
	var response response[T]
	if err = json.Unmarshal(jsonBytes, &response); err != nil {
		return nil, err
	}
	return response.Result, nil
}
