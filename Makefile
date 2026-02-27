WASM_EXEC := $(shell go env GOROOT)/lib/wasm/wasm_exec.js

.PHONY: web serve clean

web: web/blizzardsim.wasm web/wasm_exec.js

web/blizzardsim.wasm: $(wildcard *.go) go.mod
	GOOS=js GOARCH=wasm go build -o $@ .

web/wasm_exec.js: $(WASM_EXEC)
	cp $< $@

serve: web
	cd web && python3 -m http.server 8080

clean:
	rm -f web/blizzardsim.wasm web/wasm_exec.js
