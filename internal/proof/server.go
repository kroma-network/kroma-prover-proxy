package proof

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sync"

	"google.golang.org/grpc"

	"github.com/kroma-network/kroma-prover-grpc-proto/prover"
	"github.com/kroma-network/kroma-prover-proxy/internal/ec2"
)

type Server struct {
	prover.UnimplementedProverServer
	disk            *DiskRepository
	ec2             *ec2.Controller
	mu              sync.Mutex
	inProgressProof map[string]*sync.WaitGroup
}

func NewServer(disk *DiskRepository, ec2 *ec2.Controller) *Server {
	return &Server{
		disk:            disk,
		ec2:             ec2,
		inProgressProof: make(map[string]*sync.WaitGroup),
	}
}

func (s *Server) Prove(_ context.Context, request *prover.ProveRequest) (*prover.ProveResponse, error) {
	id := computeId(request)
	if proof := s.disk.Find(id); proof != nil {
		return newProofResponseFromFileProof(proof), nil
	}
	s.mu.Lock()
	wg := s.inProgressProof[id]
	if wg == nil {
		wg = &sync.WaitGroup{}
		wg.Add(1)
		s.inProgressProof[id] = wg
		go func() {
			withClient(s, func(c prover.ProverClient) (*prover.ProveResponse, error) {
				res, err := c.Prove(context.Background(), request)
				if err == nil {
					s.disk.Save(id, &FileProof{FinalPair: res.FinalPair, Proof: res.Proof})
				}
				delete(s.inProgressProof, id)
				return res, err
			})
			wg.Done()
		}()
	}
	s.mu.Unlock()
	wg.Wait()
	return newProofResponseFromFileProof(s.disk.Find(id)), nil
}

func (s *Server) Spec(ctx context.Context, request *prover.ProverSpecRequest) (*prover.ProverSpecResponse, error) {
	return withClient(s, func(c prover.ProverClient) (*prover.ProverSpecResponse, error) {
		return c.Spec(ctx, request)
	})
}

func (s *Server) Close() {
	s.disk.Close()
}

func withClient[R interface{}](s *Server, callback func(c prover.ProverClient) (*R, error)) (*R, error) {
	defer func() {
		if len(s.inProgressProof) == 0 {
			s.ec2.StopIfRunning()
		}
	}()
	if err := s.ec2.StartIfNotRunning(); err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(s.ec2.IpAddress())
	if err != nil {
		return nil, err
	}
	return callback(prover.NewProverClient(conn))
}

func computeId(request *prover.ProveRequest) string {
	hash := md5.Sum([]byte(request.TraceString))
	return hex.EncodeToString(hash[:])
}

func newProofResponseFromFileProof(proof *FileProof) *prover.ProveResponse {
	return &prover.ProveResponse{
		FinalPair: proof.FinalPair,
		Proof:     proof.Proof,
	}
}
