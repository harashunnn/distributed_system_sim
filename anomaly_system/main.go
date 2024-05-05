package main

import (
	"fmt"
	"math/rand"
	"time"
)

// 加算であればadd、乗算であればmultiplyがType、Valueに値を入れる命令の構造体
type Command struct {
	Type  string
	Value int
}

func server(id string, commands <-chan Command, state int) {
	for cmd := range commands {
		switch cmd.Type {
		case "add":
			state += cmd.Value
		case "multiply":
			state *= cmd.Value
		}
		fmt.Printf("%s updated state to %d\n", id, state)
	}
}

func client(id string, cmd Command, nodeA, nodeB chan<- Command) {
	// ネットワークのランダムな遅延をシミュレーションしている
	fmt.Printf("%s sent %s %d to both nodes\n", id, cmd.Type, cmd.Value)
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	nodeA <- cmd
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	nodeB <- cmd
}

func main() {
	nodeA := make(chan Command)
	nodeB := make(chan Command)

	go server("Node A", nodeA, 10)
	go server("Node B", nodeB, 10)

	go client("Command 1", Command{"add", 5}, nodeA, nodeB)
	go client("Command 2", Command{"multiply", 3}, nodeA, nodeB)

	time.Sleep(2 * time.Second)
}
