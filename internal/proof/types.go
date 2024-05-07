package proof

type (
	Type int32

	ProveResponse struct {
		FinalPair []byte `json:"final_pair,omitempty"`
		Proof     []byte `json:"proof,omitempty"`
	}

	ProverSpecResponse struct {
		Degree      uint32 `json:"degree,omitempty"`
		AggDegree   uint32 `json:"agg_degree,omitempty"`
		ChainId     uint32 `json:"chain_id,omitempty"`
		MaxTxs      uint32 `json:"max_txs,omitempty"`
		MaxCallData uint32 `json:"max_call_data,omitempty"`
	}
)

const (
	ProofType_NONE  Type = 0
	ProofType_EVM   Type = 1
	ProofType_STATE Type = 2
	ProofType_SUPER Type = 3
	ProofType_AGG   Type = 4
)
