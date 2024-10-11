package proof

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

func (s *Service) Prove(traceString string) (*ZKEVMProofResponse, error) {
	proofId, blockNumber := computeProofId(traceString), readBlockNumber(traceString)
	if proof := s.disk.Find(proofId); proof != nil {
		log.Printf("proof for block number %s already generated, load from disk", blockNumber)
		return newProofResponseFromFileProof(proof)
	}

	s.mu.Lock()
	wg := s.inProgressProof[proofId]
	if wg == nil {
		var err error
		wg, err = withClient(s, func(c ProverClient) (*sync.WaitGroup, error) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			s.inProgressProof[proofId] = wg
			go func(proofId, blockNumber string) {
				defer wg.Done()
				defer func() {
					s.mu.Lock()
					delete(s.inProgressProof, proofId)
					s.mu.Unlock()
					if len(s.inProgressProof) == 0 {
						log.Println("there is no proof in progress, shut down prover instance if running")
						s.ec2.StopIfRunning()
					}
				}()
				log.Printf("send request for proof generation to prover (blockNumber: %s, proofId: %s)", blockNumber, proofId)
				res, err := c.Prove(traceString)
				if err != nil {
					log.Println(fmt.Errorf("error occured while proof generation (blockNumber: %s, proofId: %s): %w", blockNumber, proofId, err))
				} else {
					log.Printf("proof generation completed (blockNumber: %s, proofId: %s)", blockNumber, proofId)
				}
				proof := &FileProof{}
				if res != nil {
					proof.FinalPair = res.FinalPair
					proof.Proof = res.Proof
				}
				if err != nil {
					proof.Error = err.Error()
					proof.RpcError = NewJsonRpcErrorFromErrorOrNil(err)
				}
				s.disk.Save(proofId, proof)
			}(proofId, blockNumber)
			return wg, nil
		})
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	s.mu.Unlock()
	log.Printf("waiting for proof generation (blockNumber: %s, proofId: %s)", blockNumber, proofId)
	wg.Wait()
	return newProofResponseFromFileProof(s.disk.Find(proofId))
}

func (s *Service) Spec() (*ProverSpecResponse, error) {
	log.Println("send request of prover spec")
	return withClient(s, func(c ProverClient) (*ProverSpecResponse, error) { return c.Spec() })
}

func (s *Service) Close() {
	s.disk.Close()
}

func withClient[R interface{}](s *Service, callback func(c ProverClient) (*R, error)) (*R, error) {
	defer func() {
		if len(s.inProgressProof) == 0 {
			log.Println("there is no proof in progress, shut down prover instance if running")
			s.ec2.StopIfRunning()
		}
	}()
	if err := s.ec2.StartIfNotRunning(); err != nil {
		return nil, err
	}
	client := NewProverClient(s.ec2.IpAddress())
	for { // Wait for the prover server to run.
		_, err := client.Spec()
		if err == nil {
			break
		}
		var urlError *url.Error
		if errors.As(err, &urlError) {
			log.Println("instance started. but server not ready. waiting...", "err", err)
			time.Sleep(1 * time.Second)
		} else {
			// unexpected error
			return nil, err
		}
	}
	return callback(client)
}

func computeProofId(traceString string) string {
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
			log.Println("readBlockNumber: blockNumber is not string")
			return ""
		}
		log.Println("readBlockNumber: header field is not object")
		return ""
	}
	log.Println("readBlockNumber: header does not exist")
	return ""
}

func newProofResponseFromFileProof(proof *FileProof) (*ZKEVMProofResponse, error) {
	if proof == nil {
		return nil, errors.New("unexpected error")
	}
	if len(proof.Error) != 0 {
		if proof.RpcError != nil {
			return nil, proof.RpcError
		}
		return nil, NewJsonRpcErrorFromString(proof.Error)
	}
	return &ZKEVMProofResponse{
		FinalPair: proof.FinalPair,
		Proof:     proof.Proof,
	}, nil
}
