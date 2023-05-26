module github.com/ubermint/cluster

replace github.com/ubermint/kv => ../kv/

go 1.20

require github.com/ubermint/kv v0.0.0-00010101000000-000000000000

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.47.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
)
