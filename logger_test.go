package eventgoround

import (
	"fmt"
	"os"
	"testing"
)

func TestRotatingFileWriter(t *testing.T) {
	logFile := "/tmp/test_rotation.log"
	defer os.Remove(logFile)
	defer os.Remove(logFile + ".1")

	// Create a writer with small max size for testing (1KB)
	writer, err := NewRotatingFileWriter(logFile, 1024)
	if err != nil {
		t.Fatalf("Failed to create rotating file writer: %v", err)
	}
	defer writer.Close()

	// Write data that will trigger rotation
	data := make([]byte, 500)
	for i := range data {
		data[i] = 'A'
	}

	// Write 3 times (1500 bytes total, should trigger rotation)
	for i := 0; i < 3; i++ {
		n, err := writer.Write(data)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}
	}

	// Check that rotation occurred by verifying backup file exists
	if _, err := os.Stat(logFile + ".1"); os.IsNotExist(err) {
		t.Error("Expected backup file to be created after rotation")
	}

	// Verify main file is smaller than 1KB (was rotated)
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	if info.Size() >= 1024 {
		t.Errorf("Expected log file to be rotated, size is %d bytes", info.Size())
	}

	t.Logf("Rotation test successful: main file=%d bytes, backup file exists", info.Size())
}

func TestRotatingFileWriterConcurrent(t *testing.T) {
	logFile := "/tmp/test_concurrent.log"
	defer os.Remove(logFile)
	defer os.Remove(logFile + ".1")

	writer, err := NewRotatingFileWriter(logFile, 10240)
	if err != nil {
		t.Fatalf("Failed to create rotating file writer: %v", err)
	}
	defer writer.Close()

	// Launch multiple goroutines writing concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				msg := fmt.Sprintf("Goroutine %d, message %d\n", id, j)
				writer.Write([]byte(msg))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify file was written
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Expected log file to have content")
	}

	t.Logf("Concurrent write test successful: %d bytes written", info.Size())
}
