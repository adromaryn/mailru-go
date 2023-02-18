package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type TableField struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
}

type Table struct {
	Table string
	Fields []TableField
}

type Handler struct {
	DB     *sql.DB
	Tables *[]Table
	TablesHash *map[string]int
	RecordItemPathRegexp *regexp.Regexp
	RecordsListPathRegexp *regexp.Regexp
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	parsedPath := h.RecordItemPathRegexp.FindSubmatch([]byte(path))
	if len(parsedPath) == 3 {
		panic("AAA")
	}

	parsedPath = h.RecordsListPathRegexp.FindSubmatch([]byte(path))
	if len(parsedPath) == 2 {
		tableName := string(parsedPath[1])
		if r.Method == http.MethodGet {
			h.getTableRecordsList(w, r, tableName)
			return
		} else {
			http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
			return
		}
	}

	if r.Method == http.MethodGet {
		h.getTablesList(w, r)
		return
	} else {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	
}

func (h *Handler) getTablesList(w http.ResponseWriter, r *http.Request) {
	var tables []string
	for _, table := range *h.Tables {
		tables = append(tables, table.Table)
	}

	response := map[string]interface{}{"response": map[string]interface{}{"tables": tables}}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) getTableRecordsList(w http.ResponseWriter, r *http.Request, tableName string) {
	const LIMIT = 5
	const OFFSET = 0

	tableIndex, ok := (*h.TablesHash)[tableName]
	if !ok {
		result, err := json.Marshal(map[string]string{"error": "unknown table"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		http.Error(w, string(result), http.StatusNotFound)
		return
	}

	table := (*h.Tables)[tableIndex]

	var limit int
	var err error
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = LIMIT
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid parameter `limit`", http.StatusBadRequest)
		}
	}

	var offset int
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = OFFSET
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid parameter `offset`", http.StatusBadRequest)
		}
	}

	var fieldNames []string
	for _, field  := range table.Fields {
		fieldNames = append(fieldNames, field.Field)
	}

	query := fmt.Sprintf("SELECT %v FROM %v LIMIT ? OFFSET ?", strings.Join(fieldNames, ", "), tableName)
	rows, err :=  h.DB.Query(query, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	result := make([]map[string]interface{}, 0)
	defer rows.Close()
	for rows.Next() {
		record := h.prepareValuesSlice(tableName)
		rows.Scan(record...)

		recordMap := make(map[string]interface{})
		for index, fieldName := range fieldNames {
			reflectValue := h.getValueFromInterface(tableName, record[index], index)
			recordMap[fieldName] = reflectValue
		}

		result = append(result, recordMap)
	}

	response := map[string]interface{}{"response": map[string]interface{}{"records": result}}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) prepareValuesSlice(tableName string) []interface{} {
	tableIndex, ok := (*h.TablesHash)[tableName]
	if !ok {
		panic("Call unexisted table")
	}
	table := (*h.Tables)[tableIndex]

	values := make([]interface{},len(table.Fields))

	for idx, field := range table.Fields {
		if field.Type == "int" {
			values[idx] = reflect.New(reflect.PtrTo(reflect.TypeOf(int64(0)))).Interface()
		} else if field.Type == "string" {
			values[idx] = reflect.New(reflect.PtrTo(reflect.TypeOf(""))).Interface()
		} else if field.Type == "boolean" {
			values[idx] = reflect.New(reflect.PtrTo(reflect.TypeOf(false))).Interface()
		} else if field.Type == "float" {
			values[idx] = reflect.New(reflect.PtrTo(reflect.TypeOf(0.0))).Interface()
		} else {
			panic("Invalid type")
		}
	}

	return values
}

func (h *Handler) getValueFromInterface(tableName string, reflected interface{}, index int) interface{} {
	tableIndex, ok := (*h.TablesHash)[tableName]
	if !ok {
		panic("Call unexisted table")
	}
	table := (*h.Tables)[tableIndex]

	pointer := reflect.Indirect(reflect.ValueOf(reflected)).Interface()
	field := table.Fields[index]

	if field.Type == "int" {
		if pointer.(*int64) == nil {
			return nil
		} else {
			return *(pointer.(*int64))
		}
	} else if field.Type == "string" {
		if pointer.(*string) == nil {
			return nil
		} else {
			return *(pointer.(*string))
		}
	} else if field.Type == "boolean" {
		if pointer.(*bool) == nil {
			return nil
		} else {
			return *(pointer.(*bool))
		}
	} else if field.Type == "float" {
		if pointer.(*float64) == nil {
			return nil
		} else {
			return *(pointer.(*float64))
		}
	} else {
		panic("Invalid type")
	}
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {

	rows, err := db.Query("SHOW tables")
	if err != nil {
		panic(err)
	}

	tables := []Table{}
	var table string
	for rows.Next() {
		rows.Scan(&table)

		tables = append(tables, Table{table, nil})
	}
	rows.Close()

	tablesHash := make(map[string]int, len(tables))
	for index, table := range tables {
		rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM `%v`", table.Table))
		if err != nil {
			panic(err)
		}


		table.Fields = fillFieldsDataSlice(rows, table.Table)
		tables[index] = table
		tablesHash[table.Table] = index
	}

	recordItemPathRegexp, _ := regexp.Compile(`^\/([A-Za-z1-9\_]+)\/(\d+)\/?$`)
	recordsListPathRegexp, _ := regexp.Compile(`^\/([A-Za-z1-9\_]+)\/?$`)

	siteMux := http.NewServeMux()

	handler := &Handler{db, &tables, &tablesHash, recordItemPathRegexp, recordsListPathRegexp}
	siteMux.Handle("/", handler)

	return siteMux, nil
}

func fillFieldsDataSlice (rows *sql.Rows, table string) []TableField {
	defer rows.Close()

	var fields []TableField
	var field TableField

	varcharRegexp, _ := regexp.Compile(`^varchar\(\d+\)$`)
	intRegexp, _ := regexp.Compile(`^int\(\d+\)$`)

	for rows.Next() {
		rows.Scan(&field.Field, &field.Type, &field.Null, &field.Key, &field.Default, &field.Extra)

		if field.Type == "boolean" || field.Type == "tinyint(1)" {
			field.Type = "boolean"
		} else if isVarchar := varcharRegexp.Match([]byte(field.Type)); field.Type == "text" || isVarchar {
			field.Type = "string"
		} else if isInt := intRegexp.Match([]byte(field.Type)); isInt {
			field.Type = "int"
		} else if field.Type == "float" || field.Type == "double" {
			field.Type = "float"
		} else {
			panic("Usuppoted field type")
		}

		fields = append(fields, field)
	}

	return fields
}
