package verifier

import (
	"net"

	"scroll-tech/common/message"
)

// Verifier represents a socket connection to a halo2 verifier.
type Verifier struct {
	conn net.Conn
}

// NewVerifier Sets up a connection with the Unix socket at `path`.
func NewVerifier(path string) (*Verifier, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		conn: conn,
	}, nil
}

// Stop stops the verifier and close the socket connection
func (v *Verifier) Stop() error {
	return v.conn.Close()
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	buf, err := proof.Marshal()
	if err != nil {
		return false, err
	}

	if _, err := v.conn.Write(buf); err != nil {
		return false, err
	}

	// Read verified byte
	bs := make([]byte, 1)
	if _, err := v.conn.Read(bs); err != nil {
		return false, err
	}

	return bs[0] != 0, nil
}
