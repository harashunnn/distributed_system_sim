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

	//終了させるトリガ変数
	terminate := false

	for !terminate {
		select {
		case cmd := <-command:
			mutex.Lock()
			if cmd.Type == "terminate" { //終了させるコマンド
				terminate = true
				mutex.Unlock()
				break
			}

			//自分のクロックと命令をキューに入れ、クロックをインクリメント
			ClockCmd := ClockCommand{Command: cmd, Clock: clock}
			queue = append(queue, ClockCmd)
			fmt.Printf("%s received command %s %d, own clock to %d\n", id, cmd.Type, cmd.Value, clock)
			clock++

			//自分のクロックを付与してメッセージを送信し、クロックをインクリメント
			msgOut <- Message{Clock: clock, Command: cmd}
			fmt.Printf("%s send message %s %d, own clock to %d\n", id, cmd.Type, cmd.Value, clock)
			clock++

			mutex.Unlock()
		case msg := <-msgIn:
			mutex.Lock()
			//元々自分が保有していたクロックか、メッセージのクロックの大きい方に1を足したものを自分のクロックにする
			if msg.Clock > clock {
				clock = msg.Clock + 1
			} else {
				clock++
			}
			ClockCmd := ClockCommand{Command: msg.Command, Clock: clock}
			queue = append(queue, ClockCmd)

			fmt.Printf("%s received message from other node with clock %d, own clock to %d\n", id, msg.Clock, clock)
			clock++
			mutex.Unlock()
		}
	}

	// queueを、クロックについて昇順でソートする
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].Clock < queue[j].Clock
	})

	// 状態とコマンドの出現回数を記録するマップ
	commandCount := make(map[Command]int)

	for _, queue_c := range queue {
		// コマンドの出現回数を更新
		commandCount[queue_c.Command]++
		// このコマンドが2回目に出現した場合
		if commandCount[queue_c.Command] == 2 {
			// コマンドタイプに応じて処理を実行する
			if queue_c.Command.Type == "multiply" {
				state *= queue_c.Command.Value
			} else if queue_c.Command.Type == "add" {
				state += queue_c.Command.Value
			}
		}
	}

	fmt.Printf("%s state to %d\n", id, state)
	queueOut <- queue //キューの内容を出力する
}

func main() {
	nodeA := make(chan Command, 5)
	nodeB := make(chan Command, 5)
	msgAtoB := make(chan Message, 5)
	msgBtoA := make(chan Message, 5)
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
