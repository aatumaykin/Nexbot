package main

import (
	"fmt"

	"github.com/aatumaykin/nexbot/internal/cron"
)

func main() {
	// Проверка пакетной функции
	jobID1 := cron.GenerateJobID()
	fmt.Printf("Package function: %s\n", jobID1)

	jobID2 := cron.GenerateJobID()
	fmt.Printf("Package function: %s\n", jobID2)

	fmt.Printf("IDs are unique: %v\n", jobID1 != jobID2)
}
