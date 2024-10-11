package proof

type (
	ZKEVMProofResponse struct {
		FinalPair []byte `json:"final_pair,omitempty"`
		Proof     []byte `json:"proof,omitempty"`
	}

	ZKVMProofResponse struct {
		ParentOutputRoot string `json:"parent_output_root"`
		OutputRoot       string `json:"output_root"`
		L1HeadHash       string `json:"l1_head_hash"`
		VKeyHash         string `json:"v_key_hash"`
		Proof            string `json:"proof"`
	}

	RequestStatusResponse struct {
		TaskStatusCode uint8 `json:"task_status_code"`
	}

	ProverSpecResponse struct {
		Degree      uint32 `json:"degree,omitempty"`
		AggDegree   uint32 `json:"agg_degree,omitempty"`
		ChainId     uint32 `json:"chain_id,omitempty"`
		MaxTxs      uint32 `json:"max_txs,omitempty"`
		MaxCallData uint32 `json:"max_call_data,omitempty"`
	}
)
