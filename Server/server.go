package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"time"
)

type Usdbrl struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type Bid struct {
	Usdbrl Usdbrl `json:"USDBRL"`
}

var db *sql.DB

var apiURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

func initiDatabase() {
	var err error
	db, err = sql.Open("sqlite", "./data/my_database.db")
	if err != nil {
		log.Fatal(err)
		return
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS bids (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        code TEXT,
        codein TEXT,
        name TEXT,
        high TEXT,
        low TEXT,
        varBid TEXT,
        pctChange TEXT,
        bid TEXT,
        ask TEXT,
        timestamp TEXT,
        create_date TEXT
    )
    `)
	if err != nil {
		log.Fatal("Erro ao criar tabela: ", err)
		return
	}
}

func saveToDatabase(reqctx context.Context, data Bid) {
	ctx, cancel := context.WithTimeout(reqctx, 10*time.Millisecond)
	defer cancel()

	// Iniciar uma transação
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Erro ao iniciar transação:", err)
		return
	}

	usdbrl := data.Usdbrl

	// Tentar inserir os dados na tabela
	_, err = tx.ExecContext(
		ctx,
		`
        INSERT INTO bids (code, codein, name, high, low, varBid, pctChange, bid, ask, timestamp, create_date)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usdbrl.Code,
		usdbrl.Codein,
		usdbrl.Name,
		usdbrl.High,
		usdbrl.Low,
		usdbrl.VarBid,
		usdbrl.PctChange,
		usdbrl.Bid,
		usdbrl.Ask,
		usdbrl.Timestamp,
		usdbrl.CreateDate,
	)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Println("Erro: operação de salvamento no banco de dados excedeu o tempo limite. ", ctx.Err())
		} else {
			log.Println("Erro ao salvar no banco de dados:", err)
		}
		return
	}

	// Confirmar a transação se não houver erros
	if comErr := tx.Commit(); comErr != nil {
		log.Println("Erro ao confirmar transação:", comErr)
	} else {
		log.Println("Dados salvos no banco de dados com sucesso.")
	}
}

func main() {

	initiDatabase()
	http.HandleFunc("/cotacao", handler)
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		return
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	log.Println("Request iniciada")
	defer log.Println("Request finalizada")

	c := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Erro ao criar request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := c.Do(req)
	if err != nil {
		log.Println(err)
		http.Error(w, "Erro ao realizar request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Erro da API externa: "+resp.Status, http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, "Erro ao ler resposta: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var data Bid
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Erro ao parsear JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	saveToDatabase(ctx, data)

	_, err = w.Write([]byte(data.Usdbrl.Bid))
	if err != nil {
		log.Println(err)
	}
}
