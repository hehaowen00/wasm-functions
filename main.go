package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"os"
)

var configInstance *Config

type Config struct {
	Addr    string `json:"addr"`
	Port    string `json:"port"`
	Db      string `json:"db_path"`
	Fs      string `json:"fs_dir"`
	Modules string `json:"modules_dir"`
}

func ConfigFromPathOrDefault(path string) (Config, error) {
	var config Config

	if path == "" {
		path = "./config.json"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		config = Config{
			Addr:    "127.0.0.1",
			Port:    "8080",
			Db:      "./wasm.db",
			Fs:      "./fs",
			Modules: "./modules",
		}

		bytes, err := json.MarshalIndent(&config, "", "  ")
		if err != nil {
			return config, err
		}

		f, err := os.Create(path)
		if err != nil {
			return config, err
		}
		defer f.Close()

		_, err = f.WriteString(string(bytes))

		return config, err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)

	return config, err
}

func (c *Config) MakeDirs() error {
	if _, err := os.Stat(c.Fs); os.IsNotExist(err) {
		err := os.Mkdir(c.Fs, 0777)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(c.Modules); os.IsNotExist(err) {
		err := os.Mkdir(c.Modules, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) FsDir() string {
	return c.Fs
}

func (c *Config) ModulesDir() string {
	return c.Modules
}

type FunctionConfig struct {
	Name    string            `json:"name"`
	Timeout int64             `json:"timeout"`
	Methods []string          `json:"methods"`
	Folders []string          `json:"folders"`
	Vars    map[string]string `json:"vars"`
}

func (config *FunctionConfig) parse(data string) error {
	err := json.Unmarshal([]byte(data), config)
	if err != nil {
		return err
	}
	return nil
}

func (config *FunctionConfig) blob() ConfigBlob {
	return ConfigBlob{
		Folders: config.Folders,
		Vars:    config.Vars,
	}
}

type ConfigBlob struct {
	timeout int64
	Folders []string          `json:"folders"`
	Vars    map[string]string `json:"vars"`
}

type Key struct {
	key string
}

var ContextKey = &Key{"connection"}

func SaveConnection(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ContextKey, c)
}
func GetConn(r *http.Request) net.Conn {
	return r.Context().Value(ContextKey).(net.Conn)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config, err := ConfigFromPathOrDefault("config.json")
	if err != nil {
		log.Fatal(err)
	}

	if err := config.MakeDirs(); err != nil {
		log.Fatal(err)
	}

	configInstance = &config

	moduleCache = make(map[string][]byte)

	addr := config.Addr + ":" + config.Port
	db := SetupDatabase(configInstance.Db)

	conn := NewConn(db)
	server := NewServer(conn)

	r := mux.NewRouter()

	// send function JSON configuration using id from upload
	r.HandleFunc("/upload", server.upload).Methods("POST")

	// list functions
	r.HandleFunc("/functions", server.listFunctions).Methods("GET")

	// call function
	r.HandleFunc("/wasm/{id}", server.handler)

	httpServer := http.Server{
		Addr:        addr,
		ConnContext: SaveConnection,
		Handler:     r,
	}

	log.Println("Started Server on", addr)
	err = httpServer.ListenAndServe()
	log.Fatal(err)
}
