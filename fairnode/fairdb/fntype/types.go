// this type for mongodb stored type
package fntype

type Header struct {
	ParentHash   string `json:"parentHash" bson:"parentHash"`
	Coinbase     string `json:"miner" bson:"miner"`
	Root         string `json:"stateRoot" bson:"stateRoot"`
	TxHash       string `json:"transactionsRoot" bson:"transactionsRoot"`
	VoteHash     string `json:"voteRoot" bson:"voteRoot"`
	ReceiptHash  string `json:"receiptsRoot" bson:"receiptsRoot"`
	Bloom        []byte `json:"logsBloom" bson:"logsBloom"`
	Difficulty   string `json:"difficulty" bson:"difficulty"`
	Number       uint64 `json:"number" bson:"number"`
	GasLimit     uint64 `json:"gasLimit" bson:"gasLimit"`
	GasUsed      uint64 `json:"gasUsed" bson:"gasUsed"`
	Time         string `json:"timestamp" bson:"timestamp"`
	Extra        []byte `json:"extraData" bson:"extraData"`
	Nonce        uint64 `json:"nonce" bson:"nonce"`
	Otprn        []byte `json:"otprn" bson:"otprn"`
	FairnodeSign []byte `json:"fairnodeSign" bson:"fairnodeSign"`
}

type Body struct {
	Transactions []Transaction `json:"transactions" bson:"transactions"`
	Voters       []Voter       `json:"voters" bson:"voters"`
}

type TxData struct {
	Type         uint64 `json:"type"  bson:"type"`
	AccountNonce uint64 `json:"nonce" bson:"nonce"`
	Price        string `json:"gasPrice" bson:"gasPrice"`
	GasLimit     uint64 `json:"gas"    bson:"gas"`
	Recipient    string `json:"to"  bson:"to"`
	Amount       string `json:"value" bson:"value"`
	Payload      []byte `json:"input" bson:"input"`
}

type Transaction struct {
	Hash      string `json:"txHash" xml:"txHash"`
	BlockHash string `json:"blockHash" xml:"blockHash"`
	Data      TxData
}

// for saving transation -> transaction collection
type STransaction struct {
	Hash      string `json:"txHash" xml:"txHash" bson:"_id,omitempty"`
	BlockHash string `json:"blockHash" xml:"blockHash" bson:"blockHash"`
	From      string `json:"From" xml:"From" bson:"From"`
	Data      TxData
}

type Voter struct {
	Header   []byte `json:"blockHeader" bson:"blockHeader"`
	Voter    string `json:"voter" bson:"_id,omitempty"`
	VoteSign []byte `json:"voterSign" bson:"voterSign"`
}

type BlockHeader struct {
	Hash   string `json:"blockHash" bson:"_id,omitempty"`
	Header Header `json:"header" bson:"header"`
}

type Block struct {
	Hash   string `json:"blockHash" bson:"_id,omitempty"`
	Header Header `json:"header" bson:"header"`
	Body   Body   `json:"body" bson:"body"`
}

type RawBlock struct {
	Hash string `json:"blockHash" bson:"_id,omitempty"`
	Raw  string `json:"rowBlock" bson:"rowBlock"`
}

type VoteAggregation struct {
	VoteKey string  `json:"voteKey" bson:"_id,omitempty"`
	Voters  []Voter `json:"voters" bson:"voters"`
}

type ChainConfig struct {
	MinMiner    uint64 `json:"minMiner" bson:"minMiner"` // minimum node count in league
	BlockNumber uint64 `json:"blockNumber" bson:"blockNumber"`
	JoinTxPrice string `json:"joinTransactionPrice" bson:"joinTransactionPrice"`
	FnFee       string `json:"fairnodeFee" bson:"fairnodeFee"`
	Mminer      uint64 `json:"targetMiner" bson:"targetMiner"`
	Epoch       uint64 `json:"epoch" bson:"epoch"`
	NodeVersion string `json:"nodeVersion" bson:"nodeVersion"`
	Sign        []byte `json:"sign" bson:"sign"`
}

type Otprn struct {
	Hash      string      `json:"otprnHash" bson:"_id,omitempty"`
	Rand      []byte      `json:"rand" bson:"rand"`
	FnAddr    string      `json:"fairnodeAddress" bson:"fairnodeAddress"`
	Cminer    uint64      `json:"currentMiner" bson:"currentMiner"`
	Data      ChainConfig `json:"data" bson:"data"`
	Sign      []byte      `json:"sign" bson:"sign"`
	Timestamp int64       `json:"timestamp" bson:"timestamp"`
	Raw       []byte      `json:"rawData" bson:"rawData"`
}

type HeartBeat struct {
	Enode        string `json:"enode" bson:"_id,omitempty"`
	MinerAddress string `json:"minerAddress" bson:"minerAddress"`
	ChainID      string `json:"chainID" bson:"chainID"`
	NodeVersion  string `json:"nodeVersion" bson:"nodeVersion"`
	Host         string `json:"host" bson:"host"`
	Port         int64  `json:"port" bson:"port"`
	Time         int64  `json:"timestamp" bson:"timestamp"`
	Head         string `json:"head" bson:"head"`
	Sign         []byte `json:"sign" bson:"sign"`
}

type League struct {
	OtprnHash string      `json:"otprnHash" bson:"_id,omitempty"`
	Nodes     []HeartBeat `json:"nodes" bson:"nodes"`
}

type Config struct {
	Config    ChainConfig `json:"config" bson:"config"`
	Timestamp int64       `json:"timestamp" bson:"timestamp"`
}
