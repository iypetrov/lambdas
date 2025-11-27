package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Base64Encode(data string) (string, error) {
	if data == "" {
		return "", fmt.Errorf("Cannot encode empty string")
	}

	return base64.URLEncoding.EncodeToString([]byte(data)), nil
}

func Base64Decode(data string) (string, error) {
	if data == "" {
		return "", fmt.Errorf("Cannot decode empty string")
	}

	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("Failed to decode base64 string: %v", err)
	}

	return string(decoded), nil
}

func JsonMarshal[T any](data T) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err 
	}

	return string(jsonBytes), nil
}
