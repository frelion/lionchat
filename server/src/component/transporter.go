package component

import (
	"context"
	"fmt"

	treemap "github.com/liyue201/gostl/ds/map"
)

type Transporter struct {
	mainPipeline               chan *transMsg     // 主管道
	pipelineMap                *treemap.Map       // 映射管道
	ctxFromPullFromMainPipline context.Context    // 取消上下文 传给所有的读协程
	cancelPullFromMainPipline  context.CancelFunc // 取消函数 用于通知pullFromMainPipline
	state                      bool               // 表示是否在运行
}

type transMsg struct {
	fromId string      // 唯一标识的id 如果fromid为-1则代表这个链接已断开 如果fromid为0则表示本消息传送所有的id
	toId   string      // 唯一表示的id 如果toId为-1则代表广播
	data   interface{} // 数据
}

// 用于分配消息
func (trans *Transporter) pullFromMainPipline(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done(): // 表示本transMsg被关闭了
			cancel() // 关闭掉所有的读协程
			return
		case msg := <-trans.mainPipeline:
			fmt.Println(msg)
			fromId := msg.fromId // 查看发出id
			toId := msg.toId     // 查看目标id
			if fromId == "-1" {  // 表示本chan应当关闭
				if iter := trans.pipelineMap.Find(toId); iter.IsValid() {
					pipeline := iter.Value().(chan transMsg) // 找到chan
					trans.pipelineMap.Erase(toId)            // 从集合中删除
					close(pipeline)                          // 关闭该chan
				}
			} else { // 正常传输信息
				if iter := trans.pipelineMap.Find(toId); iter.IsValid() {
					pipeline := iter.Value().(chan *transMsg) // 找到chan
					pipeline <- msg                           // 传送进去
				}
			}
		}
	}
}

// 用于读取消息,写入bigchan
func (trans *Transporter) ReadConn(Sw *SafeWebsocket, ctx context.Context, cancelWrite context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			cancelWrite()
			Sw.conn.Close()
			return
		default:
			msg := transMsg{}
			if err := Sw.ReadJson(&msg); err == nil {
				trans.mainPipeline <- &msg
			} else {
				cancelWrite()
				Sw.conn.Close()
				return
			}
		}
	}
}

// 读取smallchan，用于写入消息
func (trans *Transporter) WriteConn(Sw *SafeWebsocket, pipeline chan *transMsg, myId string, ctxFromRead context.Context) {
	for {
		select {
		case msg := <-pipeline:
			Sw.WriteJson(msg)
		case <-ctxFromRead.Done():
			// 通知关闭本pipeline
			trans.mainPipeline <- &transMsg{
				fromId: "-1",
				toId:   myId,
				data:   nil,
			}
			return
		}
	}
}

// 增加一个conn
func (trans *Transporter) AddConn(Sw *SafeWebsocket) {
	if trans.state == false {
		return
	}
	uuid := Uuidmaker()
	trans.pipelineMap.Insert(uuid, make(chan *transMsg, 50*50))
	ctxFromRead, cancelWrite := context.WithCancel(context.TODO())
	go trans.ReadConn(Sw, trans.ctxFromPullFromMainPipline, cancelWrite)
	go trans.WriteConn(Sw, trans.pipelineMap.Find(uuid).Value().(chan *transMsg), uuid, ctxFromRead)
	trans.mainPipeline <- &transMsg{ // 传送自己的id
		fromId: uuid,
		toId:   "-1",
		data:   nil,
	}
	Ids := make([]string, 0, trans.pipelineMap.Size())
	for iter := trans.pipelineMap.First(); iter.IsValid(); iter.Next() {
		id := iter.Key().(string)
		if id != uuid {
			Ids = append(Ids, id)
		}
	}
	trans.mainPipeline <- &transMsg{ // 传送所有的id
		fromId: "0",
		toId:   uuid,
		data:   Ids,
	}
}

// 开始run
func (trans *Transporter) Run(cap int) {
	trans.mainPipeline = make(chan *transMsg, cap)
	trans.pipelineMap = treemap.New(treemap.WithGoroutineSafe())
	var ctxFromRun context.Context
	ctxFromRun, trans.cancelPullFromMainPipline = context.WithCancel(context.TODO())
	var cancelRead context.CancelFunc
	trans.ctxFromPullFromMainPipline, cancelRead = context.WithCancel(context.TODO())
	go trans.pullFromMainPipline(ctxFromRun, cancelRead)
	trans.state = true
}

// 关闭
func (trans *Transporter) Exit() {
	trans.cancelPullFromMainPipline()
	trans.state = false
}
