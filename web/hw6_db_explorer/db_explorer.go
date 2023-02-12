package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type TablesListHandler struct {
	DB *sql.DB
}

func (h *TablesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	rows, err := h.DB.Query("SHOW tables")
	if err != nil {
		log.Fatalf("Error querying tables: %v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var tables []string
	var table string
	for rows.Next() {
		rows.Scan(&table)
		tables = append(tables, table)
	}

	response := map[string]interface{}{"response": map[string]interface{}{"tables": tables}}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Fatalf("Error querying tables: %v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	siteMux := http.NewServeMux()
	handler := &TablesListHandler{db}
	siteMux.Handle("/", handler)

	return siteMux, nil
}
