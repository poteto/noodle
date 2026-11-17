package filex

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteFileAtomicConcurrentWindowsReaders(t *testing.T) {
	for _, count := range []int{2, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "mise.json")
			first, second := bytes.Repeat([]byte("x"), 32768), bytes.Repeat([]byte("y"), 16384)
			if err := WriteFileAtomic(target, first); err != nil {
				t.Fatal(err)
			}
			stop := make(chan struct{})
			failures := make(chan error, count)
			var readers sync.WaitGroup
			for i := 0; i < count; i++ {
				readers.Add(1)
				go func() {
					defer readers.Done()
					for {
						select {
						case <-stop:
							return
						default:
						}
						data, err := ReadFile(target)
						if err != nil {
							failures <- err
							return
						}
						if !bytes.Equal(data, first) && !bytes.Equal(data, second) {
							failures <- fmt.Errorf("partial or invalid file: %d bytes", len(data))
							return
						}
					}
				}()
			}
			defer func() {
				close(stop)
				readers.Wait()
				close(failures)
				for err := range failures {
					t.Error(err)
				}
			}()
			for i := 0; i < 100; i++ {
				data := first
				if i%2 == 0 {
					data = second
				}
				if err := WriteFileAtomic(target, data); err != nil {
					t.Fatalf("update %d: %v", i, err)
				}
			}
		})
	}
}

func TestWriteFileAtomicRetainsOpenReaderVersion(t *testing.T) {
	target := filepath.Join(t.TempDir(), "state.snapshot.json")
	if err := WriteFileAtomic(target, []byte("old")); err != nil {
		t.Fatal(err)
	}
	reader, err := openShared(target)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := WriteFileAtomic(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	old, err := io.ReadAll(reader)
	if err != nil || string(old) != "old" {
		t.Fatalf("old handle: %q, %v", old, err)
	}
	current, err := ReadFile(target)
	if err != nil || string(current) != "new" {
		t.Fatalf("current path: %q, %v", current, err)
	}
}
