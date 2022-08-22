package unboundedChan

import "container/list"

type chanBuffer[T any] struct{ *list.List }

type ChanItem[T any] struct {
	Value    T
	Priority int
}

type UnboundedChan[T any] struct {
	In     chan<- ChanItem[T]
	Out    <-chan T
	buffer chanBuffer[T]
}

func NewUnboundedChan[T any]() *UnboundedChan[T] {

	in := make(chan ChanItem[T])
	out := make(chan T)

	ch := &UnboundedChan[T]{
		In:  in,
		Out: out,
		buffer: chanBuffer[T]{
			list.New(),
		},
	}

	go func() {
		defer close(out)

		for {
			val, ok := <-in
			if !ok {
				break
			}
			// 尝试直接将 in 里的数据写入 out
			select {
			case out <- val.Value:
				continue
			default:
				ch.buffer.PushPriorityItem(val)
			}
			for ch.buffer.Len() > 0 {
				select {
				case out <- ch.buffer.Front().Value.(ChanItem[T]).Value:
					ch.buffer.Remove(ch.buffer.Front())
				case val, ok := <-in:
					if !ok {
						continue
					}
					ch.buffer.PushPriorityItem(val)
				}
			}
		}
	}()
	return ch
}

func (p *UnboundedChan[T]) Len() int {
	return p.buffer.Len()
}

func (ls chanBuffer[T]) PushPriorityItem(v ChanItem[T]) {
	if ls.Len() == 0 {
		ls.PushBack(v)
	} else if v.Priority > ls.Front().Value.(ChanItem[T]).Priority {
		ls.PushFront(v)
	} else if v.Priority <= ls.Back().Value.(ChanItem[T]).Priority {
		ls.PushBack(v)
	} else {
		for e := ls.Front().Next(); e != nil; e = e.Next() {
			if v.Priority <= e.Value.(ChanItem[T]).Priority {
				continue
			}
			ls.InsertBefore(v, e)
			break
		}
	}
}
