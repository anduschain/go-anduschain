package fntype

type Fairnode struct {
	ID      string `json:"id" bson:"_id,omitempty"`
	Address string `json:"address" bson:"address"`
	Status  string `json:"status" bson:"status"`
	Timestamp int64 `json:"timestamp" bson:"timestamp"`
}
