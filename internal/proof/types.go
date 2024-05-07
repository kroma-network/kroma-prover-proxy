package proof

type (
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
