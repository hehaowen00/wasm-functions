package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

const S1 string = "INSERT INTO functions (name) VALUES (?)"
const S2 string = "SELECT id, name FROM functions"
const S3 string = `SELECT timeout FROM configs
WHERE id = ?`
const S4 string = `SELECT http.method FROM http
JOIN methods ON http.id = methods.method
WHERE methods.id = ?`
const S5 string = "INSERT INTO folders (label) VALUES (?)"
const S6 string = "SELECT id FROM folders WHERE label = ?"
const S7 string = "INSERT INTO configs (id, timeout, config) VALUES (?, ?, ?)"
const S8 string = `INSERT INTO methods (id, method)
VALUES (?, (SELECT  id FROM http WHERE method = ?))`
const S9 string = `SELECT id FROM methods
WHERE id = ? AND method = (SELECT id FROM http WHERE method = ?)`
const S10 string = `INSERT INTO metrics (function_id, called_at, duration)
VALUES (?, ?, ?)`
const S11 string = "SELECT config FROM configs WHERE id = ?"
const S12 string = "SELECT id FROM folders WHERE label = ?"

type Transaction struct {
	tx *sql.Tx
}

func (conn *Transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *Transaction) Rollback() error {
	return conn.tx.Rollback()
}

type Entry struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Timeout int64    `json:"timeout"`
	Methods []string `json:"methods"`
}

func (conn *Transaction) AddFunction(name string) (int64, error) {
	stmt, err := conn.tx.Prepare(S1)
	if err != nil {
		return 0, err
	}

	res, err := stmt.Exec(name)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (conn *Transaction) GetFunctions() ([]Entry, error) {
	collection := make([]Entry, 0)

	stmt, err := conn.tx.Prepare(S2)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry Entry

		if err := rows.Scan(&entry.ID, &entry.Name); err != nil {
			log.Fatal(err)
			return nil, err
		}

		stmt, err = conn.tx.Prepare(S3)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		rows1 := stmt.QueryRow(entry.ID)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		err = rows1.Scan(&entry.Timeout)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		stmt, err := conn.tx.Prepare(S4)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		rows2, err := stmt.Query(entry.ID)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		defer rows2.Close()

		for rows2.Next() {
			var method string
			if err := rows2.Scan(&method); err != nil {
				return nil, err
			}
			entry.Methods = append(entry.Methods, method)
		}

		collection = append(collection, entry)
	}

	return collection, nil
}

func (conn *Transaction) AddFolder(name string) error {
	_, err := conn.GetFolder(name)
	if err == nil {
		return nil
	}
	stmt, err := conn.tx.Prepare(S5)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(name)
	return err
}

func (conn *Transaction) GetFolder(name string) (string, error) {
	stmt, err := conn.tx.Prepare(S6)
	if err != nil {
		return "", err
	}

	row := stmt.QueryRow(name)

	var id string
	err = row.Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

func (conn *Transaction) AddConfigBlob(id int64, timeout int64, blob *ConfigBlob) error {
	bytes, err := json.Marshal(blob)
	if err != nil {
		return err
	}

	if timeout <= 0 {
		timeout = 1000
	}

	stmt, err := conn.tx.Prepare(S7)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id, timeout, bytes)
	return err
}

func (conn *Transaction) AddMethod(id int64, method string) error {
	stmt, err := conn.tx.Prepare(S8)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, method)
	return err
}

func (conn *Transaction) CheckMethod(id string, method string) (bool, error) {
	stmt, err := conn.tx.Prepare(S9)
	if err != nil {
		return false, nil
	}

	var id_ int64

	row := stmt.QueryRow(id, method)
	err = row.Scan(&id_)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (conn *Transaction) AddMetric(id string, timestamp time.Time, duration int64) error {
	stmt, err := conn.tx.Prepare(S10)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id, timestamp.Format(time.RFC3339), duration)
	if err != nil {
		return err
	}

	return nil
}

func (conn *Transaction) GetConfig(id string) (ConfigBlob, error) {
	stmt, err := conn.tx.Prepare(S11)
	row := stmt.QueryRow(id)

	var blob ConfigBlob

	if err != nil {
		return blob, err
	}

	var bytes []byte

	err = row.Scan(&bytes)
	if err != nil {
		return blob, err
	}

	err = json.Unmarshal(bytes, &blob)

	return blob, err
}

func (conn *Transaction) GetFolders(list []string) (map[string]string, error) {
	folders := make(map[string]string)

	for _, name := range list {
		stmt, err := conn.tx.Prepare(S12)
		if err != nil {
			return nil, err
		}

		var id string

		row := stmt.QueryRow(name)
		if err := row.Scan(&id); err != nil {
			return nil, err
		}

		folders[name] = id
	}

	log.Println(folders)

	return folders, nil
}
