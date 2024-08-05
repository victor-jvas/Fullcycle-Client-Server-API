package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type Bid struct {
	Usdbrl struct {
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
	} `json:"USDBRL"`
}

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", bidHandler)
	http.ListenAndServe(":8080", mux)

	clientRequest()

}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	bid, err := getBid()
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

func getBid() (string, error) {
	resp, err := http.Get("https://economia.awesomeapi.com.br/json/last/USD-BRL")
	if err != nil {
		return "", err
	}
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
	return bid.Usdbrl.Bid, nil
}
