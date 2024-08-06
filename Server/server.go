package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
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

func createTable(db *sql.DB) error {
	_, err := db.Exec(`
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
		fmt.Println("Error creating table:", err)
		return err
	}
	return nil
}

func saveToDatabase(reqCtx context.Context, db *sql.DB, bid Bid) error {

	ctx, cancel := context.WithTimeout(reqCtx, 10*time.Millisecond)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
    INSERT INTO bids (code, codein, name, high, low, varBid, pctChange, bid, ask, timestamp, create_date)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bid.Usdbrl.Code, bid.Usdbrl.Codein, bid.Usdbrl.Name, bid.Usdbrl.High, bid.Usdbrl.Low,
		bid.Usdbrl.VarBid, bid.Usdbrl.PctChange, bid.Usdbrl.Bid, bid.Usdbrl.Ask,
		bid.Usdbrl.Timestamp, bid.Usdbrl.CreateDate)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println("Error while inserting register:", err)
		return err

	}
	err = tx.Commit()
	if err != nil {
		fmt.Println("Error while commiting:", err)
		return err
	}
	return nil
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "./data/my_database.db")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", bidHandler)
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		return
	}

}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Request initiated")
	defer log.Println("Request completed")

	bid, err := getBid(ctx)

	select {
	case <-ctx.Done():
		log.Println("Requested canceled by user. Timeout.")

	default:
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(bid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

}

func getBid(reqCtx context.Context) (string, error) {

	ctx, cancel := context.WithTimeout(reqCtx, 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)

	if resp != nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		var bid Bid
		err = json.Unmarshal(body, &bid)
		if err != nil {
			return "", err
		}

		//save in the sqlite database
		err = saveToDatabase(ctx, db, bid)
		if err != nil {
			return "", err
		}

		return bid.Usdbrl.Bid, nil
	}

	return "", errors.New("No response from Economia.awesomeapi.com.br")

}
