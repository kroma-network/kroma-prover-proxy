package proof

import (
	"bytes"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestDiskSaveAndFind(t *testing.T) {
	disk := newTestDiskRepository(t)
	input, _ := disk.saveTestProof(1)
	result := disk.Find("0")
	if result == nil {
		t.Errorf("proof not exist")
	}
	if !bytes.Equal(result.FinalPair, input[0].FinalPair) {
		t.Errorf("final pair mismatch")
	}
	if !bytes.Equal(result.Proof, input[0].Proof) {
		t.Errorf("proof mismatch")
	}
}

func TestDeleteOldProof(t *testing.T) {
	disk := newTestDiskRepository(t)
	now := time.Now()
	proofs, sleepPerProof := disk.saveTestProof(10)
	disk.deleteOldProof(now.Add(time.Duration(len(proofs)/2) * sleepPerProof))
	files, _ := os.ReadDir(disk.baseDir)
	if len(files) != len(proofs)/2 {
		t.Errorf("saved proof count mismatch. expected %v, but got %v", len(proofs)/2, len(files))
	}
}

func newTestDiskRepository(t *testing.T) *DiskRepository {
	disk := NewDiskRepository("./" + t.Name())
	t.Cleanup(func() { os.RemoveAll(disk.baseDir) })
	return disk
}

func (r *DiskRepository) saveTestProof(count int) (result []*FileProof, sleepPerProof time.Duration) {
	sleepPerProof = 1 * time.Second
	for i := 0; i < count; i++ {
		result = append(result, &FileProof{
			FinalPair: []byte("test-" + strconv.Itoa(i)),
			Proof:     []byte("test-" + strconv.Itoa(i)),
		})
		r.Save(strconv.Itoa(i), result[len(result)-1])
		time.Sleep(sleepPerProof)
	}
	return
}
