package zencodeio

import (
	"encoding/json"
)

func NewJSONEncoder(v interface{}) Encoder {
	return NewMarshallerEncoder(v, json.Marshal)
}

func NewJSONDecoder(v interface{}) Decoder {
	return NewMarshallerDecoder(v, "application/json", json.Unmarshal)
}
