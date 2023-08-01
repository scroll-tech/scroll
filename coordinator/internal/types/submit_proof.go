package types

// SubmitProofParameter the SubmitProof api request parameter
type SubmitProofParameter struct {
	TaskID    string `form:"task_id" json:"task_id" binding:"required"`
	ProofType int    `form:"proof_type" json:"proof_type" binding:"required"`
	Status    int    `form:"status" json:"status"`
	Signature string `form:"signature" json:"signature" binding:"required"`
	Proof     string `form:"proof" json:"proof"`
	Error     string `form:"error" json:"error"`
}
