package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

var instances map[string]*WasmInstance

type Server struct {
	conn Conn
}

func NewServer(conn Conn) Server {
	instances = make(map[string]*WasmInstance)
	return Server{
		conn,
	}
}

func (s *Server) upload(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(32 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	tx, err := s.conn.Start()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var config FunctionConfig
	data := req.MultipartForm.Value["config"]
	err = config.parse(data[0])

	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	id, err := tx.AddFunction(config.Name)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	for _, name := range config.Folders {
		if err := tx.AddFolder(name); err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	blob := config.blob()
	err = tx.AddConfigBlob(id, config.Timeout, &blob)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	for _, method := range config.Methods {
		err = tx.AddMethod(id, method)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	file, _, err := req.FormFile("module")
	if err != nil {
		tx.Rollback()
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	defer file.Close()

	destPath := path.Join(configInstance.ModulesDir(), fmt.Sprintf("%d.wasm", id))

	dst, err := os.Create(destPath)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(500)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		tx.Rollback()
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	tx.Commit()
}

func (s *Server) listFunctions(w http.ResponseWriter, req *http.Request) {
	tx, err := s.conn.Start()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	collection, err := tx.GetFunctions()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(collection)
}

func (s *Server) handler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	fmt.Println("handler called:", id)

	tx, err := s.conn.Start()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	valid, err := tx.CheckMethod(id, req.Method)
	if err != nil || !valid {
		tx.Rollback()
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Function ID"))
		return
	}

	config, err := tx.GetConfig(id)
	if err != nil {
		tx.Rollback()
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	folders, err := tx.GetFolders(config.Folders)
	if err != nil {
		tx.Rollback()
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if instances[id] == nil {
		container, err := NewWasmInstance(id, &config, folders, req)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(err.Error())
			return
		}

		instances[id] = container
	}

	// container, err := NewWasmInstance(id, &config, folders, req)
	// if err != nil {
	// 	log.Println(err)
	// 	w.WriteHeader(500)
	// 	json.NewEncoder(w).Encode(err.Error())
	// 	return
	// }

	container := instances[id]

	// timestamp := time.Now().UTC()
	timeout := minInt64(config.timeout, 15*60*1000)

	output, err := container.runModule(timeout)
	if output == "" && err == nil {
		log.Println(err)
		w.WriteHeader(http.StatusRequestTimeout)
		w.Write([]byte(err.Error()))
		return
	}

	// elapsed := maxInt64(1, container.Elapsed.Milliseconds())

	log.Println("execution time (micro):", container.Elapsed.Microseconds())
	err = tx.Commit()

	/*
	  err = tx.AddMetric(id, timestamp, elapsed)
	  if err != nil {
	    log.Println(err)
	    w.WriteHeader(http.StatusInternalServerError)
	    w.Write([]byte(err.Error()))
	    return
	  }
	*/

	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	conn := GetConn(req)
	conn.Write([]byte(output))
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
