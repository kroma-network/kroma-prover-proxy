package proof

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type Server struct {
	service *Service
}

func NewServer(service *Service) *Server {
	return &Server{service: service}
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	switch httpRequest.RequestURI {
	case "/":
		s.serveJsonRpc(writer, httpRequest)
	case "/health":
		response := map[string]interface{}{
			"status":               "ok",
			"ec2Running":           s.service.ec2.Running(),
			"generatingProofCount": len(s.service.inProgressProof),
		}
		err := json.NewEncoder(writer).Encode(response)
		if err != nil {
			http.Error(writer, "Failed to encode JSON response", http.StatusInternalServerError)
		}
	}
}

func (s *Server) serveJsonRpc(writer http.ResponseWriter, httpRequest *http.Request) {
	var request map[string]interface{}
	err := json.NewDecoder(httpRequest.Body).Decode(&request)
	if err != nil {
		http.Error(writer, "Failed to decode JSON request", http.StatusBadRequest)
		return
	}

	method, ok := request["method"].(string)
	if !ok {
		http.Error(writer, "Method not found in JSON request", http.StatusBadRequest)
		return
	}

	params, ok := request["params"]
	if !ok {
		http.Error(writer, "Params not found in JSON request", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
	}

	if result, err := s.callMethod(method, params); err != nil {
		rpcError := NewJsonRpcErrorFromErrorOrNil(err)
		if rpcError == nil {
			rpcError = NewJsonRpcErrorFromString(err.Error())
		}
		response["error"] = rpcError
	} else {
		response["result"] = result
	}

	err = json.NewEncoder(writer).Encode(response)
	if err != nil {
		http.Error(writer, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func (s *Server) callMethod(method string, params interface{}) (any, error) {
	switch method {
	case "prove":
		log.Println("prove requested")
		p := params.([]any)
		traceString, traceStringOk := p[0].(string)
		if !traceStringOk {
			return nil, errors.New("failed to read traceString parameter")
		}
		proofType, proofTypeError := strconv.Atoi(fmt.Sprintf("%v", p[1]))
		if proofTypeError != nil {
			return nil, proofTypeError
		}
		return s.service.Prove(traceString, Type(proofType))
	case "spec":
		log.Println("spec requested")
		return s.service.Spec()
	default:
		return nil, fmt.Errorf("unsupported method %s", method)
	}
}

func (s *Server) Close() {
	s.service.Close()
}
