# Usage

1. Upload a multipart HTTP POST request to `/upload`
  - A JSON config is added as text to the `config` entry
  - Compiled wasm module is added to the `module` entry as a file

2. Call the function using its ID 
  - ID can be found by calling the `/functions` endpoint using a HTTP GET request
  - Function will only receive the request body

