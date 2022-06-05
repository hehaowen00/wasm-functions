package main

import (
  "fmt"
  wasmer "github.com/wasmerio/wasmer-go/wasmer"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path"
  "time"
)

type WasmInstance struct {
  module   *wasmer.Module
  env      *wasmer.WasiEnvironment
  instance *wasmer.Instance
  Elapsed  time.Duration
}

func NewWasmInstance(name string, config *ConfigBlob, folders map[string]string, req *http.Request) (*WasmInstance, error) {
  store := wasmer.NewStore(wasmer.NewEngine())

  module, err := loadWasmModule(store, name)
  if err != nil {
    return nil, err
  }

  body, err := io.ReadAll(req.Body)
  if err != nil {
    return nil, err
  }

  vars := make(map[string]string)
  for k, v := range config.Vars {
    vars[k] = v
  }

  vars["REQUEST"] = string(body)

  env, err := createEngineAndEnv("program", vars, folders)
  if err != nil {
    return nil, err
  }

  obj, err := env.GenerateImportObject(store, module)
  if err != nil {
    return nil, err
  }

  instance, err := wasmer.NewInstance(module, obj)
  if err != nil {
    return nil, err
  }

  var Elapsed time.Duration

  container := &WasmInstance{
    module,
    env,
    instance,
    Elapsed,
  }

  return container, nil
}

func (c *WasmInstance) runModule(timeout int64) (string, error) {
  s0 := time.Now()

  start, err := c.instance.Exports.GetWasiStartFunction()

  if err != nil {
    return "", err
  }

  c1 := make(chan error, 1)
  go func() {
    _, err = start()
    c1 <- err
  }()

  select {
  case err = <- c1:
    if err != nil {
      return "", err
    }
  case <- time.After(time.Duration(timeout) * time.Millisecond):
    return "", nil
  }

  s1 := time.Since(s0)
  c.Elapsed = s1

  data := string(c.env.ReadStdout())

  return data, nil
}

type Env = map[string]string

func newEnv() Env {
  return make(map[string]string)
}

func createEngineAndEnv(name string, env Env, fs map[string]string) (*wasmer.WasiEnvironment, error) {
  builder := wasmer.NewWasiStateBuilder(name)
  builder.CaptureStdout()
  builder.InheritStdin()

  for k, v := range env {
    builder.Environment(k, v)
  }

  for k, v := range fs {
    dir := path.Join(configInstance.FsDir(), v)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
      err := os.Mkdir(dir, 0777)
      if err != nil {
        return nil, err
      }
    }

    builder.MapDirectory(k, dir)
  }

  return builder.Finalize()
}

func loadWasmModule(store *wasmer.Store, name string) (*wasmer.Module, error) {
  filename := fmt.Sprintf("%s.wasm", name)
  modulePath := path.Join(configInstance.ModulesDir(), filename)

  bytes, err := ioutil.ReadFile(modulePath)
  if err != nil {
    return nil, err
  }

  return wasmer.NewModule(store, bytes)
}
