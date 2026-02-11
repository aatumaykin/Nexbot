package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWritePIDFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := WritePID(tmpDir, os.Getpid())
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Проверить, что файл существует
	pidPath := GetPIDPath(tmpDir)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Проверить содержимое
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Errorf("PID file contains invalid number: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("PID mismatch: got %d, want %d", pid, os.Getpid())
	}
}

func TestRemovePIDFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Создать PID файл
	err := WritePID(tmpDir, os.Getpid())
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Удалить через Cleanup
	err = Cleanup(tmpDir)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Проверить, что файл удален
	pidPath := GetPIDPath(tmpDir)
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file was not removed")
	}
}

func TestIsRunning(t *testing.T) {
	tmpDir := t.TempDir()

	// Создать PID файл с текущим процессом
	err := WritePID(tmpDir, os.Getpid())
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Читать PID
	pid, err := ReadPID(tmpDir)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}

	// Проверить, что процесс запущен
	running := IsRunning(pid)
	if !running {
		t.Error("Current process should be running")
	}
}

func TestPIDConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Создать PID файл с фиктивным PID (который не запущен)
	fakePID := 99999
	err := WritePID(tmpDir, fakePID)
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Попытаться создать снова с другим PID через перезапись
	err = WritePID(tmpDir, os.Getpid())
	if err != nil {
		t.Errorf("Overwrite PID file should succeed: %v", err)
	}

	// Проверить, что PID обновился
	pid, err := ReadPID(tmpDir)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("PID mismatch after overwrite: got %d, want %d", pid, os.Getpid())
	}
}

func TestReadPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := GetPIDPath(tmpDir)
	expectedPID := 12345

	// Создать файл вручную
	err := os.WriteFile(pidPath, fmt.Appendf(nil, "%d\n", expectedPID), 0600)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	// Читать PID
	pid, err := ReadPID(tmpDir)
	if err != nil {
		t.Errorf("ReadPID failed: %v", err)
	}

	if pid != expectedPID {
		t.Errorf("PID mismatch: got %d, want %d", pid, expectedPID)
	}
}

func TestReadNonExistentPIDFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Читать несуществующий файл
	_, err := ReadPID(tmpDir)
	if err == nil {
		t.Error("Expected error when reading non-existent PID file")
	}
}

func TestIsRunningZeroPID(t *testing.T) {
	// Проверить, что PID 0 не считается запущенным
	running := IsRunning(0)
	if running {
		t.Error("PID 0 should not be considered running")
	}
}

func TestIsRunningNonExistentPID(t *testing.T) {
	// Проверить, что несуществующий процесс не считается запущенным
	running := IsRunning(999999)
	if running {
		t.Error("Non-existent process should not be considered running")
	}
}

func TestCleanupRemovesBoth(t *testing.T) {
	tmpDir := t.TempDir()

	// Создать PID файл
	err := WritePID(tmpDir, os.Getpid())
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Создать socket файл
	socketPath := GetSocketPath(tmpDir)
	err = os.WriteFile(socketPath, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create socket file: %v", err)
	}

	// Удалить оба файла
	err = Cleanup(tmpDir)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Проверить, что оба файла удалены
	if _, err := os.Stat(GetPIDPath(tmpDir)); !os.IsNotExist(err) {
		t.Error("PID file was not removed")
	}

	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file was not removed")
	}
}

func TestCleanupNonExistentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Cleanup должен работать даже если файлы не существуют
	err := Cleanup(tmpDir)
	if err != nil {
		t.Errorf("Cleanup should not error on non-existent files: %v", err)
	}
}

func TestGetPIDPath(t *testing.T) {
	tmpDir := t.TempDir()

	expectedPath := filepath.Join(tmpDir, PIDFileName)
	actualPath := GetPIDPath(tmpDir)

	if actualPath != expectedPath {
		t.Errorf("PID path mismatch: got %s, want %s", actualPath, expectedPath)
	}
}

func TestGetSocketPath(t *testing.T) {
	tmpDir := t.TempDir()

	expectedPath := filepath.Join(tmpDir, SocketFileName)
	actualPath := GetSocketPath(tmpDir)

	if actualPath != expectedPath {
		t.Errorf("Socket path mismatch: got %s, want %s", actualPath, expectedPath)
	}
}

// Test 13: WritePID с ошибкой при создании директории (для покрытия)
func TestWritePIDInvalidPath(t *testing.T) {
	// Попытка записать PID в недопустимый путь
	// Например, путь который заканчивается на имя файла, а не директорию
	pid := os.Getpid()
	err := WritePID("/dev/null/test.pid", pid)
	if err == nil {
		t.Error("Expected error when writing PID to invalid path")
	}
}

// Test 14: WritePID перезаписывает существующий файл
func TestWritePIDOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Создать PID файл первым PID
	firstPID := 12345
	err := WritePID(tmpDir, firstPID)
	if err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	// Переписать другим PID
	secondPID := 67890
	err = WritePID(tmpDir, secondPID)
	if err != nil {
		t.Errorf("WritePID overwrite failed: %v", err)
	}

	// Проверить, что PID обновился
	pid, err := ReadPID(tmpDir)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}

	if pid != secondPID {
		t.Errorf("PID not overwritten: got %d, want %d", pid, secondPID)
	}
}

// Test 15: ReadPID с некорректным содержимым
func TestReadPIDInvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := GetPIDPath(tmpDir)

	// Создать файл с некорректным содержимым
	err := os.WriteFile(pidPath, []byte("not a number\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	// Читать PID
	_, err = ReadPID(tmpDir)
	if err == nil {
		t.Error("Expected error when reading invalid PID content")
	}
}

// Test 18: ReadPID с пустым файлом
func TestReadPIDEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := GetPIDPath(tmpDir)

	// Создать пустой файл
	err := os.WriteFile(pidPath, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	// Читать PID
	_, err = ReadPID(tmpDir)
	if err == nil {
		t.Error("Expected error when reading empty PID file")
	}
}
