package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type TableField struct {
	Field string
	Type  string
	Null  bool
}

type TablesListHandler struct {
	DB     *sql.DB
	Tables map[string][]TableField
}

func (h *TablesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	var tables []string
	for table := range h.Tables {
		tables = append(tables, table)
	}

	response := map[string]interface{}{"response": map[string]interface{}{"tables": tables}}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Fatalf("Error querying tables: %v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {

	rows, err := db.Query("SHOW tables")
	if err != nil {
		log.Fatalf("Error querying tables: %v", err.Error())
		panic(err)
	}

	tables := make(map[string][]TableField)
	var table string
	for rows.Next() {
		rows.Scan(&table)

		tables[table] = nil
	}
	rows.Close()

	for table := range tables {
		rows, err = db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%v`", table))
		if err != nil {
			log.Fatalf("Error querying table %v columns: %v", table, err.Error())
			panic(err)
		}

		var fields []TableField
		var field TableField
		for rows.Next() {
			rows.Scan(&field)
			fields = append(fields, field)
		}
		rows.Close()

		tables[table] = fields
	}
	rows.Close()

	siteMux := http.NewServeMux()
	handler := &TablesListHandler{db, tables}
	siteMux.Handle("/", handler)

	return siteMux, nil
}
