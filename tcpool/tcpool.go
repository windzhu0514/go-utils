// https://github.com/flyaways/pool/blob/master/pool_tcp.go
// https://github.com/fatih/pool/blob/master/channel.go
// https://github.com/redis/go-redis
package tcpool

import (
	"bufio"
	"container/list"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var (
	errConnBroken    = errors.New("tcpool: putIdleConn: connection is in bad state")
	errTooManyIdle   = errors.New("tcpool: putIdleConn: too many idle connections")
	errPoolClosed    = errors.New("tcpool: pool is closed")
	ErrBadConn       = errors.New("tcpool: bad connection")
	ErrGetConnFailed = errors.New("tcpool: get conn failed,max retry times")
)

type Option struct {
	MaxIdleConns int // default 0
	MaxConns     int // default 3
	MaxRetries   int // default 3
	Trace        *ConnTrace
}

type pool struct {
	hostPort string
	opt      Option

	mu           sync.Mutex // guards following fields
	idleConn     []*persistConn
	idleConnWait wantConnQueue // waiting getConns
	idleLRU      connLRU
	closech      chan struct{}
	closed       bool
	conns        int
}

func New(hostPort string, opt Option) io.WriteCloser {
	p := &pool{hostPort: hostPort, opt: opt}

	if p.opt.MaxConns == 0 {
		p.opt.MaxConns = 3
	}

	if p.opt.MaxRetries == 0 {
		p.opt.MaxRetries = 3
	}

	return p
}

func (p *pool) Write(bytes []byte) (int, error) {
	for i := 0; i < p.opt.MaxRetries; i++ {
		select {
		case <-p.closech:
			return 0, errPoolClosed
		default:
		}

		pc, err := p.getConn()
		if err != nil {
			return 0, err
		}

		n, err := pc.bw.Write(bytes)
		if err == nil {
			err = p.tryPutIdleConn(pc)
			if p.opt.Trace != nil && p.opt.Trace.PutIdleConn != nil {
				p.opt.Trace.PutIdleConn(err)
			}

			err = pc.bw.Flush()

			return n, err
		} else {
			log.Println("Write failed:" + err.Error())
			pc.close(err)
			p.removeIdleConn(pc)
			p.conns--

			if p.opt.Trace != nil && p.opt.Trace.PutIdleConn != nil {
				p.opt.Trace.PutIdleConn(err)
			}
		}

		if !pc.shouldRetry(err) {
			time.Sleep(time.Millisecond * 500)
			return 0, err
		}
	}

	return 0, ErrGetConnFailed
}

func (p *pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}

	close(p.closech)

	for _, pc := range p.idleConn {
		pc.close(nil)
	}

	return nil
}

func (p *pool) getConn() (pc *persistConn, err error) {
	if p.opt.Trace != nil && p.opt.Trace.GetConn != nil {
		p.opt.Trace.GetConn(p.hostPort)
	}

	w := &wantConn{
		ready: make(chan struct{}, 1),
	}
	defer func() {
		if err != nil {
			w.cancel(p, err)
		}
	}()

	if p.queueForIdleConn(w) {
		pc := w.pc
		if p.opt.Trace != nil && p.opt.Trace.GotConn != nil {
			p.opt.Trace.GotConn(pc.gotIdleConnTrace(pc.idleAt))
		}

		return pc, nil
	}

	if p.opt.MaxConns <= 0 || p.conns < p.opt.MaxConns {
		p.conns++
		go p.dialConnFor(w)
	}

	select {
	case <-p.closech:
		return nil, io.EOF
	case <-w.ready:
		if w.pc != nil && p.opt.Trace != nil && p.opt.Trace.GotConn != nil {
			p.opt.Trace.GotConn(GotConnInfo{Conn: w.pc.conn, Reused: w.pc.isReused()})
		}

		if w.err != nil {
			return nil, w.err
		}
		return w.pc, nil
	}
}

func (p *pool) queueForIdleConn(w *wantConn) (delivered bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.idleConn) > 0 {
		pc := p.idleConn[len(p.idleConn)-1]

		if pc.isBrokenOrClosed() {
			p.idleLRU.remove(pc)
			p.idleConn = p.idleConn[:len(p.idleConn)-1]
			continue
		}

		delivered = w.tryDeliver(pc, nil)
		if delivered {
			p.idleLRU.remove(pc)
			p.idleConn = p.idleConn[:len(p.idleConn)-1]

			return delivered
		}
	}

	p.idleConnWait.cleanFront()
	p.idleConnWait.pushBack(w)
	return false
}

func (p *pool) dialConnFor(w *wantConn) {
	pc, err := p.dialConn()
	delivered := w.tryDeliver(pc, err)
	if err == nil && !delivered {
		if err := p.tryPutIdleConn(pc); err != nil {
			pc.close(err)
		}
	}
	if err != nil {
		p.conns--
	}
}

func (p *pool) dialConn() (pc *persistConn, err error) {
	if p.opt.Trace != nil {
		if p.opt.Trace.ConnectStart != nil {
			p.opt.Trace.ConnectStart("tcp", p.hostPort)
		}
		if p.opt.Trace.ConnectDone != nil {
			defer func() { p.opt.Trace.ConnectDone("tcp", p.hostPort, err) }()
		}
	}

	raddr, err := net.ResolveTCPAddr("tcp", p.hostPort)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	bw := bufio.NewWriter(conn)

	return &persistConn{conn: conn, bw: bw}, nil
}

func (p *pool) tryPutIdleConn(pc *persistConn) error {
	if pc.isBrokenOrClosed() {
		return errConnBroken
	}

	pc.markReused()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		pc.close(nil)
		return nil
	}

	done := false
	for p.idleConnWait.len() > 0 {
		w := p.idleConnWait.popFront()
		if w.tryDeliver(pc, nil) {
			done = true
			break
		}

	}
	if done {
		return nil
	}

	if p.opt.MaxIdleConns > 0 && len(p.idleConn) >= p.opt.MaxIdleConns {
		return errTooManyIdle
	}

	for _, exist := range p.idleConn {
		if exist == pc {
			log.Fatalf("dup idle pconn %p in freelist", pc)
		}
	}

	p.idleConn = append(p.idleConn, pc)
	p.idleLRU.add(pc)
	if p.opt.MaxIdleConns != 0 && len(p.idleConn) > p.opt.MaxIdleConns {
		oldest := p.idleLRU.removeOldest()
		oldest.close(errTooManyIdle)
		p.removeIdleConnLocked(oldest)
	}

	pc.idleAt = time.Now()

	return nil
}

func (p *pool) removeIdleConn(pconn *persistConn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.removeIdleConnLocked(pconn)
}

func (p *pool) removeIdleConnLocked(pconn *persistConn) bool {
	p.idleLRU.remove(pconn)

	pconns := p.idleConn
	var removed bool
	switch len(pconns) {
	case 0:
		// Nothing
	case 1:
		if pconns[0] == pconn {
			p.idleConn = p.idleConn[0:0]
			removed = true
		}
	default:
		for i, v := range pconns {
			if v != pconn {
				continue
			}
			// Slide down, keeping most recently-used
			// conns at the end.
			copy(pconns[i:], pconns[i+1:])
			removed = true
			break
		}
	}
	return removed
}

type persistConn struct {
	conn *net.TCPConn
	bw   *bufio.Writer

	mu     sync.Mutex
	closed bool
	broken error // set non-nil when conn is closed, before closech is closed
	reused bool
	idleAt time.Time
}

func (pc *persistConn) Read(p []byte) (n int, err error) {
	return pc.conn.Read(p)
}

func (pc *persistConn) Write(p []byte) (n int, err error) {
	return pc.conn.Write(p)
}

func (pc *persistConn) close(err error) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.closed = true
	pc.broken = err

	return pc.conn.Close()
}

func (pc *persistConn) isBrokenOrClosed() bool {
	return pc.broken != nil || pc.closed
}

func (pc *persistConn) markReused() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.reused = true
}

func (pc *persistConn) isReused() bool {
	pc.mu.Lock()
	r := pc.reused
	pc.mu.Unlock()
	return r
}

func (pc *persistConn) shouldRetry(err error) bool {
	return true
}

func (pc *persistConn) gotIdleConnTrace(idleAt time.Time) (t GotConnInfo) {
	// pc.mu.Lock()
	// defer pc.mu.Unlock()
	t.Reused = pc.reused
	t.Conn = pc.conn
	t.WasIdle = false
	if !idleAt.IsZero() {
		t.WasIdle = true
		t.IdleTime = time.Since(idleAt)
	}
	return
}

type connLRU struct {
	ll *list.List // list.Element.Value type is of *persistConn
	m  map[*persistConn]*list.Element
}

// add adds pc to the head of the linked list.
func (cl *connLRU) add(pc *persistConn) {
	if cl.ll == nil {
		cl.ll = list.New()
		cl.m = make(map[*persistConn]*list.Element)
	}
	ele := cl.ll.PushFront(pc)
	if _, ok := cl.m[pc]; ok {
		panic("persistConn was already in LRU")
	}
	cl.m[pc] = ele
}

func (cl *connLRU) removeOldest() *persistConn {
	ele := cl.ll.Back()
	pc := ele.Value.(*persistConn)
	cl.ll.Remove(ele)
	delete(cl.m, pc)
	return pc
}

// remove removes pc from cl.
func (cl *connLRU) remove(pc *persistConn) {
	if ele, ok := cl.m[pc]; ok {
		cl.ll.Remove(ele)
		delete(cl.m, pc)
	}
}

// len returns the number of items in the cache.
func (cl *connLRU) len() int {
	return len(cl.m)
}

type wantConn struct {
	mu    sync.Mutex
	ready chan struct{}
	pc    *persistConn
	err   error
}

// waiting reports whether w is still waiting for an answer (connection or error).
func (w *wantConn) waiting() bool {
	select {
	case <-w.ready:
		return false
	default:
		return true
	}
}

// tryDeliver attempts to deliver pc, err to w and reports whether it succeeded.
func (w *wantConn) tryDeliver(pc *persistConn, err error) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pc != nil || w.err != nil {
		return false
	}

	w.pc = pc
	w.err = err
	if w.pc == nil && w.err == nil {
		panic("net/http: internal error: misuse of tryDeliver")
	}
	close(w.ready)
	return true
}

// cancel marks w as no longer wanting a result (for example, due to cancellation).
// If a connection has been delivered already, cancel returns it with t.putOrCloseIdleConn.
func (w *wantConn) cancel(t *pool, err error) {
	w.mu.Lock()
	if w.pc == nil && w.err == nil {
		close(w.ready) // catch misbehavior in future delivery
	}
	// pc := w.pc
	w.pc = nil
	w.err = err
	w.mu.Unlock()

	// if pc != nil {
	// 	t.putOrCloseIdleConn(pc)
	// }
}

// A wantConnQueue is a queue of wantConns.
type wantConnQueue struct {
	// This is a queue, not a deque.
	// It is split into two stages - head[headPos:] and tail.
	// popFront is trivial (headPos++) on the first stage, and
	// pushBack is trivial (append) on the second stage.
	// If the first stage is empty, popFront can swap the
	// first and second stages to remedy the situation.
	//
	// This two-stage split is analogous to the use of two lists
	// in Okasaki's purely functional queue but without the
	// overhead of reversing the list when swapping stages.
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

// len returns the number of items in the queue.
func (q *wantConnQueue) len() int {
	return len(q.head) - q.headPos + len(q.tail)
}

// pushBack adds w to the back of the queue.
func (q *wantConnQueue) pushBack(w *wantConn) {
	q.tail = append(q.tail, w)
}

// popFront removes and returns the wantConn at the front of the queue.
func (q *wantConnQueue) popFront() *wantConn {
	if q.headPos >= len(q.head) {
		if len(q.tail) == 0 {
			return nil
		}
		// Pick up tail as new head, clear tail.
		q.head, q.headPos, q.tail = q.tail, 0, q.head[:0]
	}
	w := q.head[q.headPos]
	q.head[q.headPos] = nil
	q.headPos++
	return w
}

// peekFront returns the wantConn at the front of the queue without removing it.
func (q *wantConnQueue) peekFront() *wantConn {
	if q.headPos < len(q.head) {
		return q.head[q.headPos]
	}
	if len(q.tail) > 0 {
		return q.tail[0]
	}
	return nil
}

// cleanFront pops any wantConns that are no longer waiting from the head of the
// queue, reporting whether any were popped.
func (q *wantConnQueue) cleanFront() (cleaned bool) {
	for {
		w := q.peekFront()
		if w == nil || w.waiting() {
			return cleaned
		}
		q.popFront()
		cleaned = true
	}
}
