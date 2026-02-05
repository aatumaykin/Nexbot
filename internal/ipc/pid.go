package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	PIDFileName    = ".nexbot.pid"
	SocketFileName = ".nexbot.sock"
)

// WritePID записывает PID в файл
func WritePID(workspacePath string, pid int) error {
	pidPath := GetPIDPath(workspacePath)
	pidData := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(pidPath, []byte(pidData), 0600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// ReadPID читает PID из файла
func ReadPID(workspacePath string) (int, error) {
	pidPath := GetPIDPath(workspacePath)

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, err
	}

	return pid, nil
}

// IsRunning проверяет что процесс запущен
func IsRunning(pid int) bool {
	if pid == 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 - проверяет существование процесса
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}

// GetSocketPath возвращает путь к сокету
func GetSocketPath(workspacePath string) string {
	return filepath.Join(workspacePath, SocketFileName)
}

// GetPIDPath возвращает путь к PID файлу
func GetPIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, PIDFileName)
}

// Cleanup удаляет PID файл и сокет
func Cleanup(workspacePath string) error {
	pidPath := GetPIDPath(workspacePath)
	socketPath := GetSocketPath(workspacePath)

	// Remove PID file
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove socket file
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
