package proof

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/kroma-network/kroma-prover-proxy/internal/ec2"
)

type Service struct {
	disk            *DiskRepository
	ec2             *ec2.Controller
	mu              sync.Mutex
	inProgressProof map[string]*sync.WaitGroup
}

func NewService(disk *DiskRepository, ec2 *ec2.Controller) *Service {
	return &Service{
		disk:            disk,
		ec2:             ec2,
		inProgressProof: make(map[string]*sync.WaitGroup),
	}
}

func (s *Service) Prove(traceString string, proofType Type) (*ProveResponse, error) {
	id, blockNumber := computeId(traceString), readBlockNumber(traceString)
	log.Printf("request prove for block number %s to prover", blockNumber)
	if proof := s.disk.Find(id); proof != nil {
		return newProofResponseFromFileProof(proof)
	}
	s.mu.Lock()
	wg := s.inProgressProof[id]
	if wg == nil {
		var err error
		wg, err = withClient(s, func(c ProverClient) (*sync.WaitGroup, error) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			s.inProgressProof[id] = wg
			go func(id, blockNumber string) {
				defer wg.Done()
				defer func() {
					s.mu.Lock()
					delete(s.inProgressProof, id)
					s.mu.Unlock()
					if len(s.inProgressProof) == 0 {
						s.ec2.StopIfRunning()
					}
				}()
				log.Println("prove start.", "blockNumber:", blockNumber, "id:", id)
				res, err := c.Prove(traceString, proofType)
				log.Println("prove complete.", "blockNumber:", blockNumber, "id:", id, "err:", err)
				proof := &FileProof{}
				if res != nil {
					proof.FinalPair = res.FinalPair
					proof.Proof = res.Proof
				}
				if err != nil {
					proof.Error = err.Error()
					proof.RpcError = NewJsonRpcErrorFromErrorOrNil(err)
				}
				s.disk.Save(id, proof)
			}(id, blockNumber)
			return wg, nil
		})
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	s.mu.Unlock()
	log.Println("waiting proof generation.", "blockNumber:", blockNumber, "id:", id)
	wg.Wait()
	return newProofResponseFromFileProof(s.disk.Find(id))
}

func (s *Service) Spec() (*ProverSpecResponse, error) {
	log.Println("request spec to prover")
	return withClient(s, func(c ProverClient) (*ProverSpecResponse, error) { return c.Spec() })
}

func (s *Service) Close() {
	s.disk.Close()
}

func withClient[R interface{}](s *Service, callback func(c ProverClient) (*R, error)) (*R, error) {
	defer func() {
		if len(s.inProgressProof) == 0 {
			log.Println("there is no proof in progress. shut down if it is running.")
			s.ec2.StopIfRunning()
		}
	}()
	if err := s.ec2.StartIfNotRunning(); err != nil {
		return nil, err
	}
	client, err := NewProverClient(s.ec2.IpAddress())
	if err != nil {
		return nil, err
	}
	for { // Wait for the prover server to run.
		_, err := client.Spec()
		if err == nil {
			break
		}
		var urlError *url.Error
		if errors.As(err, &urlError) {
			log.Println("instance started. but server not ready. waiting...")
			time.Sleep(1 * time.Second)
		} else {
			// unexpected  error
			return nil, err
		}
	}
	return callback(client)
}

func computeId(traceString string) string {
	hash := md5.Sum([]byte(traceString))
	return hex.EncodeToString(hash[:])
}

func readBlockNumber(traceString string) string {
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(traceString), &result); err != nil {
		log.Println("readBlockNumber: failed to json.Unmarshal", err)
	}
	if header, ok := result["header"]; ok {
		if header, ok := header.(map[string]interface{}); ok {
			if number, ok := header["number"].(string); ok {
				return number
			}
			log.Println("readBlockNumber: blockNUmber does not string")
			return ""
		}
		log.Println("readBlockNumber: header field does not object")
		return ""
	}
	log.Println("readBlockNumber: header does not exist")
	return ""
}

func newProofResponseFromFileProof(proof *FileProof) (*ProveResponse, error) {
	if proof == nil {
		return nil, errors.New("unexpected error")
	}
	if len(proof.Error) != 0 {
		if proof.RpcError != nil {
			return nil, proof.RpcError
		}
		return nil, NewJsonRpcErrorFromString(proof.Error)
	}
	return &ProveResponse{
		FinalPair: proof.FinalPair,
		Proof:     proof.Proof,
	}, nil
}
