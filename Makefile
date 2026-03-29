server-build:
	go build -o ./cmd/server ./cmd/server
agent-build:
	go build -o ./cmd/agent ./cmd/agent
app-build:
	make server-build
	make agent-build

server-run:
	go run ./cmd/server/main.go
agent-run:
	go run ./cmd/agent/main.go

server-test:
	go test -v ./internal/handler/server
agent-test:
	go test -v ./internal/agent

# тесты для тасок
test-iter1:
	make server-build /
	./metricstest -test.v -test.run=^TestIteration1$ \
            -binary-path=cmd/server/server

test-iter2: 
	make agent-build 
	./metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=./ -agent-binary-path=cmd/agent/agent

test-iter3:
	make app-build 
	./metricstest -test.v -test.run=^TestIteration3[AB]*$ \
        -source-path=. \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server