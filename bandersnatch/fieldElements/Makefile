all: Optimizations.txt Compiles.txt Objdump.txt
Optimizations.txt: *.go
	go build -gcflags="-m=2" . 2> Optimizations.txt
a.out: *.go
	go build -o a.out .
Compiles.txt: *.go
	go build -gcflags="-S" . 2> Compiles.txt
Objdump.txt: *.go a.out
	go tool objdump a.out > Objdump.txt
