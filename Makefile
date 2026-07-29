server-build:
	go build -o ./cmd/server ./cmd/server
agent-build:
	go build -o ./cmd/agent ./cmd/agent
app-build:
	make server-build
	make agent-build

server-run:
	go run ./cmd/server
agent-run:
	go run ./cmd/agent

agent-run-race:
	go run -race ./cmd/agent

server-test:
	go test -v ./internal/handler/server
agent-test:
	go test -v ./internal/agent

linter:
	go run ./cmd/linter ./...
   
code-generation:
	go run ./cmd/reset

# тесты для тасок
test-iter1:
	make server-build 
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

test-iter4:
	make app-build 
	SERVER_PORT=8084 ADDRESS="localhost:8084" TEMP_FILE="hello.txt" ./metricstest -test.v -test.run=^TestIteration4$ \
		-agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=8084 \
        -source-path=.

test-iter5:
	make app-build 
	SERVER_PORT=8085 ADDRESS="localhost:8085" TEMP_FILE="hello.txt" ./metricstest -test.v -test.run=^TestIteration4$ \
		-agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
    	-server-port=8085 \
    	-source-path=.

test-iter6:
	make app-build
	SERVER_PORT=8086 ADDRESS="localhost:8086" TEMP_FILE="hello.txt" ./metricstest -test.v -test.run=^TestIteration6$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=8086 \
        -source-path=.

test-iter7:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="hello.txt" ./metricstest -test.v -test.run=^TestIteration7$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=8080 \
        -source-path=.

test-iter8:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="hello.txt" ./metricstest -test.v -test.run=^TestIteration8$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=8080 \
        -source-path=.

test-iter9:	
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration9$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -file-storage-path="storage.json" \
        -server-port=8080 \
        -source-path=.

test-iter10:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration10$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://postgres:root@localhost:5432/praktikum?sslmode=disable' \
            -server-port=8080 \
            -source-path=.

migrate-force-11:
	migrate -database "postgres://postgres:root@localhost:5432/praktikum?sslmode=disable" -path ./migrations force 1

migrate-up-11:
	migrate -database "postgres://postgres:root@localhost:5432/praktikum?sslmode=disable" -path ./migrations up
migrate-down-11:
	migrate -database "postgres://postgres:root@localhost:5432/praktikum?sslmode=disable" -path ./migrations down --force

test-iter11:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration11$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://postgres:root@localhost:5432/praktikum?sslmode=disable' \
            -server-port=8080 \
            -source-path=.

test-iter12:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration12$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://postgres:root@localhost:5432/praktikum?sslmode=disable' \
            -server-port=8080 \
            -source-path=.

test-iter13:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration13$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://postgres:root@localhost:5432/praktikum?sslmode=disable' \
            -server-port=8080 \
            -source-path=.

test-iter14:
	make app-build
	SERVER_PORT=8080 ADDRESS="localhost:8080" TEMP_FILE="storage.json" ./metricstest -test.v -test.run=^TestIteration14$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://postgres:root@localhost:5432/praktikum?sslmode=disable' \
            -server-port=8080 \
            -source-path=. \
            -key="hello" \