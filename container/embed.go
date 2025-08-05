package container

import (
	"bytes"
	"fmt"
	"log"
	"myclient/utils"
	"os"
	"os/exec"
	"path/filepath"
)

func RunEmbeddingContainer(inputFile string) ([]byte, error) {
	absPath, err := filepath.EvalSymlinks(inputFile)
	if err != nil {
		return nil, fmt.Errorf("invalid input file path: %w", err)
	}

	dir := filepath.Dir(absPath)
	fileName := utils.SanitizeFileName(filepath.Base(absPath))

	containerDir := "/data"
	containerInputPath := filepath.Join(containerDir, fileName)

	if _, err := os.Stat(absPath); err != nil {
		log.Printf("file does not exist at: %s", absPath)
		return nil, fmt.Errorf("file not found: %s", absPath)
	}

	cmd := exec.Command(
		"docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:%s:ro", dir, containerDir),
		"protectednative/embed:latest",
		containerInputPath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("Host path:", absPath)
	files, _ := os.ReadDir(filepath.Dir(absPath))
	for _, f := range files {
		fmt.Println("Host sees:", f.Name())
	}

	fmt.Println("running embedding container...")
	if err := cmd.Run(); err != nil {
		log.Print(stderr.String())
		return nil, fmt.Errorf("container failed: %v\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
