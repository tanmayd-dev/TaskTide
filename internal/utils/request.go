package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func ReadRequest[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var completion T
	err := json.NewDecoder(r.Body).Decode(&completion)
	return completion, err
}

func HasError(w http.ResponseWriter, r *http.Request, resp string, err error) bool {
	if err == nil {
		return false
	}
	log.Printf("%v\n", err.Error())
	http.Error(w, resp, http.StatusBadRequest)
	return true
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write JSON response: %v\n", err)
	}
}

