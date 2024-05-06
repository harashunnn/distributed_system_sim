package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// 加算であればadd、乗算であればmultiplyがType、Valueに値を入れる命令の構造体
type Command struct {
	Type  string
	Value int
}

// ノードを終了する
func (cmd Command) isTerminate() bool {
	return cmd.Type == "terminate"
}

// コマンドとクロックを同時に保管するときの構造体
type ClockCommand struct {
	Command Command
	Clock   int
}

// ノードからノードに送るメッセージの構造体
type Message struct {
	Clock   int
	Command Command
}

func server(id string, command <-chan Command, msgIn <-chan Message, msgOut chan<- Message, state int, queueOut chan<- []ClockCommand) {
	var queue []ClockCommand
	clock := 0
	mutex := &sync.Mutex{}
	//終了させるトリガ
	terminate := false

	for !terminate {
		select {
		//ユーザから命令を受信したら
		case cmd := <-command:
			mutex.Lock()
			if cmd.isTerminate() {
				terminate = true
				break
			}
			processLocalCommand(id, cmd, &clock, &queue, msgOut)
			mutex.Unlock()
		//ノードからメッセージを受信したら
		case msg := <-msgIn:
			mutex.Lock()
			processReceivedMessage(id, msg, &clock, &queue)
			mutex.Unlock()
		}
	}
	//キューの情報を元に、命令を実行する
	finalizeQueue(id, queue, state, queueOut)
}

func processLocalCommand(id string, cmd Command, clock *int, queue *[]ClockCommand, msgOut chan<- Message) {
	//自分のクロックと命令をキューに入れ、クロックをインクリメント
	recordCommand(id, cmd, clock, queue)
	//自分のクロックを付与してメッセージを送信し、クロックをインクリメント
	sendMessage(id, cmd, clock, msgOut)
}

func processReceivedMessage(id string, msg Message, clock *int, queue *[]ClockCommand) {
	//元々自分が保有していたクロックか、メッセージのクロックの大きい方に1を足したものを自分のクロックにする
	updateClock(msg.Clock, clock)
	//自分のクロックと命令をキューに入れ、クロックをインクリメント
	recordCommand(id, msg.Command, clock, queue)
}

func recordCommand(id string, cmd Command, clock *int, queue *[]ClockCommand) {
	*queue = append(*queue, ClockCommand{Command: cmd, Clock: *clock})
	fmt.Printf("%s received command %s %d, at clock %d\n", id, cmd.Type, cmd.Value, *clock)
	*clock++
}

func sendMessage(id string, cmd Command, clock *int, msgOut chan<- Message) {
	msgOut <- Message{Clock: *clock, Command: cmd}
	fmt.Printf("%s sent message %s %d at clock %d\n", id, cmd.Type, cmd.Value, *clock)
	*clock++
}

func updateClock(receivedClock int, clock *int) {
	if receivedClock > *clock {
		*clock = receivedClock + 1
	} else {
		*clock++
	}
}

func finalizeQueue(id string, queue []ClockCommand, state int, queueOut chan<- []ClockCommand) {
	// queueを、クロックについて昇順でソートする
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].Clock < queue[j].Clock
	})
	processQueueCommands(&state, queue)
	fmt.Printf("%s final state %d\n", id, state)
	queueOut <- queue
}

func processQueueCommands(state *int, queue []ClockCommand) {
	// 状態とコマンドの出現回数を記録するマップ
	commandCount := make(map[Command]int)

	for _, cc := range queue {
		// コマンドの出現回数を更新
		commandCount[cc.Command]++
		// このコマンドが2回目に出現した時
		// Command.Typeに応じて処理を実行する
		if commandCount[cc.Command] == 2 {
			if cc.Command.Type == "multiply" {
				*state *= cc.Command.Value
			} else if cc.Command.Type == "add" {
				*state += cc.Command.Value
			}
		}
	}
}

func main() {
	nodeA := make(chan Command, 10)
	nodeB := make(chan Command, 10)
	msgAtoB := make(chan Message, 10)
	msgBtoA := make(chan Message, 10)
	queueOutA := make(chan []ClockCommand, 1)
	queueOutB := make(chan []ClockCommand, 1)

	//サーバのゴルーチンを起動
	go server("Node A", nodeA, msgBtoA, msgAtoB, 10, queueOutA)
	go server("Node B", nodeB, msgAtoB, msgBtoA, 10, queueOutB)

	//NodeAには加算命令を、NodeBには乗算命令を先に送り、順番の異なるメッセージが先に到着した事象をシミュレーションしている
	nodeA <- Command{"add", 5}
	time.Sleep(100 * time.Millisecond)
	nodeA <- Command{"multiply", 3}
	time.Sleep(100 * time.Millisecond)
	nodeB <- Command{"multiply", 3}
	time.Sleep(100 * time.Millisecond)
	nodeB <- Command{"add", 5}
	time.Sleep(100 * time.Millisecond)

	//ゴルーチンを終了
	nodeA <- Command{"terminate", 0}
	nodeB <- Command{"terminate", 0}

	//キューの中身を出力
	queueA := <-queueOutA
	queueB := <-queueOutB
	fmt.Println("Final queue state in Node A:", queueA)
	fmt.Println("Final queue state in Node B:", queueB)
}
