# ASCII Image Generator with Golang & WebAssembly

Convert images into ASCII art using a web interface. Built using Go (Golang) and WebAssembly (WASM) to execute in the client's browser, providing a fast and efficient way to generate ASCII art from an image.

live version at [Demo](ascii-image-generator-two.vercel.app/).

## Install
Clone the project

```bash
git clone git@github.com:BrunoPoiano/ascii-image-generator.git
cd ascii-image-generator
```
### To run locally

**Install dependencies**
```bash
npm i
```
**Start the Server on Port 3000**
```bash
node server.js
```

### Go Code Location

```bash
src/go/main.go
```

**To compile the Go code into WebAssembly, run:**
```bash
GOOS=js GOARCH=wasm go build -o main.wasm
```

## Ref

 - [Bild](https://github.com/anthonynsimon/bild)
