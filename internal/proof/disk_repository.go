package proof

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type FileProof struct {
	FinalPair []byte `json:"final_pair,omitempty"`
	Proof     []byte `json:"proof,omitempty"`
	Error     error
}

type DiskRepository struct {
	baseDir      string
	deleteBefore time.Duration
	closeContext context.Context
	Close        context.CancelFunc
}

func NewDiskRepository(baseDir string) *DiskRepository {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		err = os.MkdirAll(baseDir, 0777)
		if err != nil {
			log.Panicln(fmt.Errorf("os.MkdirAll failed: %w", err))
		}
	}
	if !strings.HasSuffix(baseDir, "/") {
		baseDir += "/"
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	disk := &DiskRepository{
		baseDir:      baseDir,
		deleteBefore: 7 * 24 * time.Hour,
		closeContext: ctx,
		Close:        cancelFunc,
	}
	go disk.scheduleDeleteOldProof(10 * time.Minute)
	return disk
}

func (r *DiskRepository) Find(id string) (proof *FileProof) {
	file, err := os.ReadFile(r.baseDir + id)
	if err == nil {
		err = json.Unmarshal(file, &proof)
		if err != nil {
			log.Printf("json.Unmarshal failed. %v", err)
		}
	}
	return
}

func (r *DiskRepository) Save(id string, proof *FileProof) {
	jsonResult, _ := json.Marshal(proof)
	err := os.WriteFile(r.baseDir+id, jsonResult, 0644)
	if err != nil {
		log.Printf("os.WriteFile failed. %v", err)

	}
	return
}

func (r *DiskRepository) scheduleDeleteOldProof(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			deletedCount := r.deleteOldProof(time.Now().Add(-r.deleteBefore))
			log.Printf("deleted old proof count %d\n", deletedCount)
		case <-r.closeContext.Done():
			return
		}
	}
}

// deleteOldProof deletes proofs stored at a time earlier than time.
func (r *DiskRepository) deleteOldProof(time time.Time) (deletedCount int) {
	files, _ := os.ReadDir(r.baseDir)
	for _, file := range files {
		info, _ := file.Info()
		hasError := func() bool {
			proof := r.Find(file.Name())
			if proof == nil {
				return false
			}
			return proof.Error != nil
		}
		if info.ModTime().Before(time) || hasError() {
			if err := os.Remove(r.baseDir + file.Name()); err != nil {
				log.Println(fmt.Errorf("failed to delete old proof %s: %w", file.Name(), err))
			} else {
				deletedCount++
			}
		}
	}
	return
}
