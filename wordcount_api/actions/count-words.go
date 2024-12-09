package actions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/gobuffalo/buffalo"
)

func CountWords(c buffalo.Context) error {
	// Define um timeout de 2 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Canal para capturar o resultado
	resultChan := make(chan map[string]string)
	errorChan := make(chan error)

	// Nome do arquivo vindo da URL
	filename := c.Param("filename")

	// Inicia o processamento em uma goroutine separada
	go func() {
		// Verifica se o arquivo existe
		if err := verifyFile(filename); err != nil {
			errorChan <- fmt.Errorf("file not found: %v", err)
			return
		}

		// Inicializa os workers
		var workerCmds []*exec.Cmd
		workers := []string{"50001", "50002", "50003"}

		for _, port := range workers {
			cmd, err := startWorker(port)
			if err != nil {
				errorChan <- fmt.Errorf("error starting worker: %v", err)
				return
			}
			workerCmds = append(workerCmds, cmd)
		}

		// Inicializa o master
		filename = "files/" + filename + ".txt"
		masterCmd, err := startMaster(filename)
		if err != nil {
			// Encerra os workers caso haja erro
			for _, cmd := range workerCmds {
				stopProcess(cmd)
			}
			errorChan <- fmt.Errorf("error starting master: %v", err)
			return
		}

		// Aguarda o término do master
		err = masterCmd.Wait()
		if err != nil {
			// Encerra os workers caso o master falhe
			for _, cmd := range workerCmds {
				stopProcess(cmd)
			}
			errorChan <- fmt.Errorf("master process failed: %v", err)
			return
		}

		// Encerra os workers após o término do master
		for _, cmd := range workerCmds {
			stopProcess(cmd)
		}

		// Lê o arquivo no path ../wordcount/result/result-final.txt
		wordCount, err := readResultFile("../wordcount/result/result-final.txt")
		if err != nil {
			errorChan <- fmt.Errorf("failed to read result file: %v", err)
			return
		}

		// Envia o resultado no canal
		resultChan <- wordCount
	}()

	// Seleciona entre o timeout e o processamento
	select {
	case <-ctx.Done():
		// Timeout expirou
		defaultResponse := map[string]string{
			"message": "Processing exceeded 2 seconds, returning default response.",
		}
		return c.Render(200, r.JSON(defaultResponse))
	case err := <-errorChan:
		// Erro durante o processamento
		return c.Render(500, r.JSON(map[string]string{"error": err.Error()}))
	case wordCount := <-resultChan:
		// Resultado recebido dentro do tempo limite
		return c.Render(200, r.JSON(wordCount))
	}
}

// Função para ler o arquivo result-final.txt e convertê-lo em um map[string]string
func readResultFile(filepath string) (map[string]string, error) {
	wordCount := make(map[string]string)

	// Abre o arquivo
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the result file: %v", err)
	}
	defer file.Close()

	fmt.Println("Arquivo aberto com sucesso:", filepath)

	// Lê o arquivo linha por linha
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		var result map[string]string
		err := json.Unmarshal([]byte(line), &result)
		if err != nil {
			fmt.Println("error while unmarshaling output line:", line)
			continue // Ignora linhas inválidas
		}

		word := result["Key"]
		count := result["Value"]
		wordCount[word] = count
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read result file: %v", err)
	}

	return wordCount, nil
}

func verifyFile(filename string) error {
	filepath := "../wordcount/files/" + filename + ".txt" // Pasta onde estão os arquivos

	// Verifica se o arquivo existe
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("file %s not found", filename)
	}
	defer file.Close()

	return nil
}

func startWorker(port string) (*exec.Cmd, error) {
	// Configura o comando do worker
	cmd := exec.Command("sh", "-c", "cd ../wordcount && ./wordcount -mode distributed -type worker -port "+port)

	err := cmd.Start() // Inicia o worker em background
	if err != nil {
		return nil, fmt.Errorf("error while starting worker on port %s: %v", port, err)
	}

	return cmd, nil
}

func startMaster(filename string) (*exec.Cmd, error) {
	// Configura o comando do master
	cmd := exec.Command("sh", "-c", "cd ../wordcount && ./wordcount -mode distributed -type master -file "+filename+" -chunksize 10240 -reducejobs 5")

	// Executa o comando e aguarda a conclusão
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error while starting master: %v", err)
	}

	// Retorna o comando para gerenciamento posterior
	log.Printf("master inicializado\n")
	return cmd, nil
}

func stopProcess(cmd *exec.Cmd) error {
	// Envia sinal de interrupção para o processo
	if cmd.Process != nil {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	return fmt.Errorf("unable to find the process to be terminated")
}
