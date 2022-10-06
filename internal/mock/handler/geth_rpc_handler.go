package handler

import (
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

// GethRPCHandler mock geth rpc handler for mock geth server
type GethRPCHandler struct {
	upgrader             websocket.Upgrader
	chainID              *big.Int
	nonces               map[common.Address]uint64
	Txns                 map[common.Hash]*types.Transaction
	blockNumber          uint64
	maxPriorityFeePerGas uint64
	maxFeePerGas         uint64
	gasPrice             uint64
	gasLimit             uint64
}

type request struct {
	ID      uint64          `json:"id"          gencodec:"required"`
	Jsonrpc string          `json:"jsonrpc"     gencodec:"required"`
	Method  string          `json:"method"      gencodec:"required"`
	Params  json.RawMessage `json:"params"      gencodec:"required"`
}

type getTransactionCountRequest struct {
	Address     common.Address
	BlockNumber string
}

type sendRawTransactionRequest struct {
	raw string
}

type response struct {
	ID      uint64          `json:"id"          gencodec:"required"`
	Jsonrpc string          `json:"jsonrpc"     gencodec:"required"`
	Result  json.RawMessage `json:"result"      gencodec:"required"`
}

// NewGethRPCHanlder return a new instance of GethRPCHandler
func NewGethRPCHanlder(chainID *big.Int) *GethRPCHandler {
	return &GethRPCHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		chainID: chainID,
		nonces:  make(map[common.Address]uint64),
		Txns:    make(map[common.Hash]*types.Transaction),
	}
}

// UpdateNonce update the nonce of some account
func (mock *GethRPCHandler) UpdateNonce(account common.Address, nonce uint64) {
	mock.nonces[account] = nonce
}

// UpdateBlockNumber update the block number
func (mock *GethRPCHandler) UpdateBlockNumber(number uint64) {
	mock.blockNumber = number
}

// UpdateMaxPriorityFeePerGas update the max priority fee per gas
func (mock *GethRPCHandler) UpdateMaxPriorityFeePerGas(maxPriorityFeePerGas uint64) {
	mock.maxPriorityFeePerGas = maxPriorityFeePerGas
}

// UpdateMaxFeePerGas update the max fee per gas
func (mock *GethRPCHandler) UpdateMaxFeePerGas(maxFeePerGas uint64) {
	mock.maxFeePerGas = maxFeePerGas
}

// UpdateGasPrice update the gas price
func (mock *GethRPCHandler) UpdateGasPrice(gasPrice uint64) {
	mock.gasPrice = gasPrice
}

// UpdateGasLimit update the gas limit
func (mock *GethRPCHandler) UpdateGasLimit(gasLimit uint64) {
	mock.gasLimit = gasLimit
}

// WsHandle websocket handler
func (mock *GethRPCHandler) WsHandle(w http.ResponseWriter, r *http.Request) {
	mock.upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := mock.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("error upgrade to websocket", err)
	}

	mock.reader(ws)
}

func (mock *GethRPCHandler) reader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// client error, should disconnect by returning
			log.Error("ReadMessage error", err)
			return
		}
		var req request
		err = json.Unmarshal(p, &req)
		if err != nil {
			log.Error("Unmarshal error", err)
			continue
		}
		res, err := mock.handleMsg(req)
		if err == nil {
			var out []byte
			out, err = json.Marshal(res)
			if err != nil {
				log.Error("Marshal error", err)
				continue
			}
			if err = conn.WriteMessage(messageType, out); err != nil {
				// client error, should disconnect by returning
				log.Error("WriteMessage error", err)
				return
			}
		} else {
			log.Error("handleMsg error", err)
		}

	}
}

// Handle http handler
func (mock *GethRPCHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var q request

	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	res, err := mock.handleMsg(q)
	if err != nil {
		log.Error("handleMsg error", err)
		return
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Error("Encode error", err)
	}
}

func (mock *GethRPCHandler) getTransactionCount(params json.RawMessage) (json.RawMessage, error) {
	var p getTransactionCountRequest
	err := json.Unmarshal(params, &[]interface{}{&p.Address, &p.BlockNumber})

	if err != nil {
		return nil, err
	}

	nonce, ok := mock.nonces[p.Address]

	if ok {
		return json.Marshal(hexutil.EncodeBig(new(big.Int).SetUint64(nonce)))
	}
	return json.Marshal(hexutil.EncodeBig(new(big.Int).SetUint64(0)))
}

func (mock *GethRPCHandler) sendRawTransaction(params json.RawMessage) error {
	var p sendRawTransactionRequest
	err := json.Unmarshal(params, &[]interface{}{&p.raw})
	if err != nil {
		return err
	}

	hex, err := hexutil.Decode(p.raw)
	if err != nil {
		return err
	}

	tx := new(types.Transaction)
	err = tx.UnmarshalBinary(hex)
	if err != nil {
		return err
	}

	mock.Txns[tx.Hash()] = tx
	return nil
}

func (mock *GethRPCHandler) handleMsg(req request) (*response, error) {
	var result json.RawMessage
	var err error
	switch req.Method {
	case "eth_estimateGas":
		result, err = json.Marshal(hexutil.EncodeBig(new(big.Int).SetUint64(mock.gasLimit)))
	case "eth_sendRawTransaction":
		err = mock.sendRawTransaction(req.Params)
	case "eth_getTransactionCount":
		result, err = mock.getTransactionCount(req.Params)
	case "eth_chainId":
		result, err = json.Marshal(hexutil.EncodeBig(mock.chainID))
	case "eth_getBlockByNumber":
		block :=
			&types.Header{
				ParentHash:  common.Hash{},
				UncleHash:   types.EmptyUncleHash,
				Coinbase:    common.Address{},
				Root:        common.Hash{},
				TxHash:      types.EmptyRootHash,
				ReceiptHash: common.Hash{},
				Bloom:       types.Bloom{},
				Difficulty:  new(big.Int),
				Number:      big.NewInt(int64(mock.blockNumber)),
				GasLimit:    0,
				GasUsed:     0,
				Time:        0,
				Extra:       []byte{},
				MixDigest:   common.Hash{},
				Nonce:       types.BlockNonce{},
				BaseFee:     new(big.Int).SetUint64(mock.maxFeePerGas),
			}
		result, err = json.Marshal(block)
	case "eth_gasPrice":
		result, err = json.Marshal(hexutil.EncodeBig(new(big.Int).SetUint64(mock.gasPrice)))
	case "eth_maxPriorityFeePerGas":
		result, err = json.Marshal(hexutil.EncodeBig(new(big.Int).SetUint64(mock.maxPriorityFeePerGas)))
	case "eth_subscribe":
		result, err = json.Marshal("0xcd0c3e8af590364c09d0fa6a1210faf5")
	case "eth_unsubscribe":
		result, err = json.Marshal(true)
	}

	if err != nil {
		return nil, err
	}

	res := response{
		ID:      req.ID,
		Jsonrpc: req.Jsonrpc,
		Result:  result,
	}

	return &res, nil
}
