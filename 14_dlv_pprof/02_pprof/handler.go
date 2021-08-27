package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func MiningHandler(difficulty int, timeout time.Duration) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		block := Block{}
		err = json.Unmarshal(body, &block)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		md := make(chan BlockMetadata, 1)
		done := make(chan struct{})
		timeoutTimer := time.NewTimer(timeout)
		go mineBlock(block, difficulty, md, done)

		select {
		case v := <-md:
			block.Metadata.Nonce, block.Metadata.Hash = v.Nonce, v.Hash

			res, err := json.Marshal(block)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				timeoutTimer.Stop()
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(res)
			timeoutTimer.Stop()
			return

		case <-timeoutTimer.C:
			close(done)
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
	})
}

func mineBlock(block Block, difficulty int, md chan<- BlockMetadata, done <-chan struct{}) {
	prefix := strings.Repeat("0", difficulty)
	for i := int64(0); ; i++ {
		block.Metadata.Nonce = i
		hash := block.Hash()
		if strings.HasPrefix(hash, prefix) {
			block.Metadata.Hash = hash
			md <- block.Metadata
			break
		}
		select {
		case <-done:
			return
		default:
		}
	}
}
