package cryptos

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
)

func TestHashServiceIsSafeForConcurrentReuse(t *testing.T) {
	service := NewHashService(sha256.New)
	payload := []byte("coffeehc")
	wantHash := sha256.Sum256(payload)
	wantHex := hex.EncodeToString(wantHash[:])
	var taskGroup sync.WaitGroup
	for range 100 {
		taskGroup.Go(func() {
			if got := service.HashToHexString(payload); got != wantHex {
				t.Errorf("unexpected SHA-256 digest: got=%s want=%s", got, wantHex)
			}
		})
	}
	taskGroup.Wait()
}
