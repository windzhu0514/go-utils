package utils

import (
	"encoding/json"
)

func JsonMarshalByte(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	return data
}

func JsonMarshalString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return string(data)
}
