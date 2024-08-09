package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Fatal("Erro ao criar request:", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Erro ao fazer request:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Erro do servidor: %s\n", resp.Status)
	}

	bid, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Erro ao ler resposta:", err)
	}

	f, err := os.Create("cotacao.txt")
	if err != nil {
		log.Fatal("Erro ao criar arquivo:", err)
	}
	defer f.Close()

	msg := append([]byte("Dólar: "), bid...)
	_, err = f.Write(msg)
	if err != nil {
		panic(err)
	}

}
