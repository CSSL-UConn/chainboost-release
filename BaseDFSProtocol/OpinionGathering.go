package BaseDFSProtocol

import (
	"math/rand"
	"sync"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"golang.org/x/xerrors"

	//Raha
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/util/key"
)

/*
New Count Protocol: a simple coin tossing protocol:

The root sends a message (like Hello) signaling the beginning of the protocol,
Once this message reaches leave nodes, they:
- selects a random string
- sign it
- forwards that to their parent
The parent does the same and sends all what he received from the children to the upper level until all reach the root.

The root:
- verifies all signatures
- pick some random string
- xor all random strings received from everyone and his

So we have 3 kinds of nodes:
1) root
2) leaf
3) non-root,non-leaf

Each node should generate
- a key pair (except root)
- a random string

and we have 2 types of msgs:
1) Hello -> fixed-length
2) Opinion -> []? of a structure which includes
	- nodeID
	- Node's public key -> fixed-length string
	- signed random String -> fixed-length string
	- Random String -> fixed-length string
-------------------------------------------------------------
Previous Count Protocol:
The count-protocol returns the number of nodes reachable in a given
timeout. To correctly wait for the whole tree, every node that receives
a message sends a message to the root before contacting its children.
As long as the root receives those messages, he knows the counting
still goes on.
-------------------------------------------------------------
Note: I have kept all previous count protocol codes commented, for future compares.
*/

func init() {
	//network.RegisterMessage(PrepareCount{})
	//network.RegisterMessage(Count{})
	//network.RegisterMessage(NodeIsUp{})
	//onet.GlobalProtocolRegister("Count", NewCount)

	network.RegisterMessage(Opinion{})
	network.RegisterMessage(Hello{})
	onet.GlobalProtocolRegister("OpinionGathering", NewOpinionGathering)
}

// ProtocolCount holds all channels. If a timeout occurs or the counting
// is done, the Count-channel receives the number of nodes reachable in
// the tree.
/*
type ProtocolCount struct {
	*onet.TreeNodeInstance
	Replies          int
	Count            chan int
	Quit             chan bool
	timeout          time.Duration
	timeoutMu        sync.Mutex
	PrepareCountChan chan struct {
		*onet.TreeNode
		PrepareCount
	}
	CountChan    chan []CountMsg
	NodeIsUpChan chan struct {
		*onet.TreeNode
		NodeIsUp
	}
}*/

type ProtocolOpinionGathering struct {
	*onet.TreeNodeInstance

	FinalXor  chan string
	Quit      chan bool
	timeout   time.Duration
	timeoutMu sync.Mutex
	HelloChan chan struct {
		*onet.TreeNode
		Hello
	}
	OpinionChan chan []OpinionMsg
}

//PrepareCount is sent so that every node can contact the root to say
// the counting is still going on.
// type PrepareCount struct { Timeout time.Duration }
// NodeIsUp - if it is received by the root it will reset the counter.
//type NodeIsUp struct{}
// Count sends the number of children to the parent node.
//type Count struct { Children int32 }
// CountMsg is wrapper around the Count-structure
//type CountMsg struct {
//	*onet.TreeNode
//	Count
//}

// Hello is sent down the tree from the root node, once it get to leaves,
// they will respond with their signed string
type Hello struct {
	Timeout time.Duration
}

//Opinion is a verificable signed string from each node
type Opinion struct {
	RandomString []byte
	PublicKey    []kyber.Point
	SignedString []byte
}

//OpinionMsg is wrapper around the Opinion-structure
type OpinionMsg struct {
	*onet.TreeNode
	Opinion
}

// NewCount returns a new protocolInstance
// func NewCount(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
/*
func NewOpinionGathering(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	p := &ProtocolCount{
		TreeNodeInstance: n,
		Quit:             make(chan bool),
		timeout:          1 * time.Second,
	}
	p.Count = make(chan int, 1)
	t := n.Tree()
	if t == nil {
		return nil, xerrors.New("cannot find tree")
	}
	if err := p.RegisterChannelsLength(len(t.List()),
		&p.CountChan, &p.PrepareCountChan, &p.NodeIsUpChan); err != nil {
		log.Error("Couldn't reister channel:", err)
	}
	return p, nil
}
*/

// NewOpinionGathering returns a new protocolInstance
func NewOpinionGathering(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	p := &ProtocolOpinionGathering{
		TreeNodeInstance: n,
		Quit:             make(chan bool),
		FinalXor:         make(chan string),
		timeout:          1 * time.Second,
	}
	t := n.Tree()
	if t == nil {
		return nil, xerrors.New("cannot find tree")
	}
	if err := p.RegisterChannelsLength(len(t.List()),
		&p.HelloChan, &p.OpinionChan); err != nil {
		log.Error("Couldn't reister channel:", err)
	}
	return p, nil
}

// Start the protocol
/*
func (p *ProtocolCount) Start() error {
	// Send an empty message
	log.Lvl3("Starting to count")
	p.FuncPC()
	return nil
}
*/
func (p *ProtocolOpinionGathering) Start() error {
	// Send an empty message
	log.Lvl1("Starting to gather opinion")
	p.FuncHello()
	return nil
}

// Dispatch listens for all channels and waits for a timeout in case nothing
// happens for a certain duration
/*
func (p *ProtocolCount) Dispatch() error {
	running := true
	for running {
		log.Lvl3(p.Info(), "waiting for message for", p.Timeout())
		select {
		case pc := <-p.PrepareCountChan:
			log.Lvl3(p.Info(), "received from", pc.TreeNode.ServerIdentity.Address,
				"timeout", pc.Timeout)
			p.SetTimeout(pc.Timeout)
			p.FuncPC()
		case c := <-p.CountChan:
			p.FuncC(c)
			running = false
		case _ = <-p.NodeIsUpChan:
			if p.Parent() != nil {
				err := p.SendTo(p.Parent(), &NodeIsUp{})
				if err != nil {
					log.Lvl2(p.Info(), "couldn't send to parent",
						p.Parent().Name(), err)
				}
			} else {
				p.Replies++
			}
		case <-time.After(time.Duration(p.Timeout())):
			log.Lvl3(p.Info(), "timed out while waiting for", p.Timeout())
			if p.IsRoot() {
				log.Lvl2("Didn't get all children in time:", p.Replies)
				p.Count <- p.Replies
				running = false
			}
		}
	}
	p.Done()
	return nil
}
*/
func (p *ProtocolOpinionGathering) Dispatch() error {
	running := true
	for running {
		log.Lvl3(p.Info(), "waiting for message for", p.Timeout())
		select {
		case pc := <-p.HelloChan:
			log.Lvl1(p.Info(), "received Hello from", pc.TreeNode.ServerIdentity.Address,
				"timeout", pc.Timeout)
			p.SetTimeout(pc.Timeout)
			p.FuncHello()
		case c := <-p.OpinionChan:
			p.FuncOpinion(c)
			running = false
		case <-time.After(time.Duration(p.Timeout())):
			log.Lvl3(p.Info(), "timed out while waiting for", p.Timeout())
			if p.IsRoot() {
				log.Lvl1("Didn't get all Hellos in time")
				//p.Count <- p.Replies
				running = false
			}
		}
	}
	p.Done()
	return nil
}

// FuncPC handles PrepareCount messages. These messages go down the tree and
// every node that receives one will reply with a 'NodeIsUp'-message
/*
func (p *ProtocolCount) FuncPC() {
	if !p.IsRoot() {
		err := p.SendTo(p.Parent(), &NodeIsUp{})
		if err != nil {
			log.Lvl2(p.Info(), "couldn't send to parent",
				p.Parent().Name(), err)
		}
	}
	if !p.IsLeaf() {
		for _, child := range p.Children() {
			go func(c *onet.TreeNode) {
				log.Lvl3(p.Info(), "sending to", c.ServerIdentity.Address, c.ID, "timeout", p.timeout)
				err := p.SendTo(c, &PrepareCount{Timeout: p.timeout})
				if err != nil {
					log.Lvl2(p.Info(), "couldn't send to child",
						c.Name())
				}
			}(child)
		}
	} else {
		p.CountChan <- nil
	}
}
*/

// FuncHello handles Hello messages. These messages go down the tree and
// every node that receives one will send it down to its children, once Hello msgs get to leaves they call FuncOpinion function
func (p *ProtocolOpinionGathering) FuncHello() {
	if !p.IsLeaf() {
		for _, child := range p.Children() {
			go func(c *onet.TreeNode) {
				log.Lvl3(p.Info(), "sending to", c.ServerIdentity.Address, c.ID, "timeout", p.timeout)
				err := p.SendTo(c, &Hello{Timeout: p.timeout})
				if err != nil {
					log.Lvl2(p.Info(), "couldn't send to child",
						c.Name())
				}
			}(child)
		}
	} else {
		p.OpinionChan <- nil
	}
}

// FuncC creates a Count-message that will be received by all parents and
// count the total number of children
/*
func (p *ProtocolCount) FuncC(cc []CountMsg) {
	count := 1
	for _, c := range cc {
		count += int(c.Count.Children)
	}
	if !p.IsRoot() {
		log.Lvl3(p.Info(), "Sends to", p.Parent().ID, p.Parent().ServerIdentity.Address)
		if err := p.SendTo(p.Parent(), &Count{int32(count)}); err != nil {
			log.Lvl2(p.Name(), "coouldn't send to parent",
				p.Parent().Name())
		}
	} else {
		p.Count <- count
	}
	log.Lvl3(p.ServerIdentity().Address, "Done")
}
*/

// FuncOpinion creates an Opinion mesaages (verificable signed random strings)
// that will go up the tree,

func (p *ProtocolOpinionGathering) FuncOpinion(cc []OpinionMsg) {
	var ChildrenRandomStrings []byte
	var ChildrenPublicKeys []kyber.Point
	var ChildrenSignedStrings []byte

	for _, c := range cc {
		ChildrenRandomStrings = append(ChildrenRandomStrings, c.RandomString...)
		ChildrenPublicKeys = append(ChildrenPublicKeys, c.PublicKey...)
		ChildrenSignedStrings = append(ChildrenSignedStrings, c.SignedString...)
		// to get length of signed strings and random strings
		log.Lvl2("Raha: c.SignedString length:", len(c.SignedString),
			"\n c.RandomString length:", len(c.RandomString))
	}
	if !p.IsRoot() {
		// generate random string
		//msg := []byte("Hello Schnorr")
		rand.Seed(time.Now().UnixNano())
		msg := []byte(randSeq(13))
		// -------- signing ----------------------------
		kp := key.NewKeyPair(p.Suite())
		p.ServerIdentity().SetPrivate(kp.Private)
		s, err := schnorr.Sign(p.Suite() /* kp.Private */, p.ServerIdentity().GetPrivate(), msg)
		if err != nil {
			log.Lvl1(p.Name(), "Couldn't sign msg: Error:", err)
		}
		// ----------------------------------------------
		log.Lvl2("Raha: \n public key method 1:", p.ServerIdentity().Public.String(),
			"\n public key method 2:", kp.Public, "!!!")
		//ToDo: raha: check why these two are different

		ChildrenRandomStrings = append(ChildrenRandomStrings, msg...)
		ChildrenPublicKeys = append(ChildrenPublicKeys, kp.Public /* , p.ServerIdentity().Public */)
		ChildrenSignedStrings = append(ChildrenSignedStrings, s...)

		// Raha: Here, each node is sending an opinion message to its parent,
		// Since opinion is a component of OpinionMsg struct
		// and the type of OpinionChan is OpinionMsg (I am not sure why slice!
		// bcz they get simentanously and we want to save them to then use them?)
		// So when an opinion msg gets catched by Dispatch function,
		// it is recieved via OpinionChan Channel

		log.Lvl1(p.Info(), "Sends his Opinion + his descendant's opinions to",
			p.Parent().ID, p.Parent().ServerIdentity.Address)
		if err := p.SendTo(p.Parent(),
			&Opinion{ChildrenRandomStrings, ChildrenPublicKeys, ChildrenSignedStrings}); err != nil {
			log.Lvl2(p.Name(), "couldn't send to parent",
				p.Parent().Name(), err)
		}
		log.Lvl3(p.ServerIdentity().Address, "Done")
	} else {
		log.Lvl1("Root recieved opinions!")
		var invalidSigniture = false
		// ToDo: raha: check if all of them are recieved
		// verify signed strings
		index := 0
		for _, c := range cc {
			err := schnorr.Verify(p.Suite(),
				c.PublicKey[index], c.RandomString[index*13:index*13+13],
				c.SignedString[index*64:index*64+64])
			if err != nil {
				log.Lvl1(p.Name(), "Couldn't verify signature. Error:", err)
				invalidSigniture = true
			}
			index++
		}
		if invalidSigniture == false {
			// generate a random string
			//msg := []byte("Hello Schnorr")
			rand.Seed(time.Now().UnixNano())
			msg := []byte(randSeq(13))
			// append his own string to all childrens' signed strings
			ChildrenRandomStrings = append(ChildrenRandomStrings, msg...)
			ChildrenPublicKeys = append(ChildrenPublicKeys, nil)
			ChildrenSignedStrings = append(ChildrenSignedStrings, 0)
			// xor all the messages
			p.FinalXor <- xor(ChildrenRandomStrings, true)
		} else {
			p.FinalXor <- xor(ChildrenRandomStrings, false)
		}

	}
}

// SetTimeout sets the new timeout
func (p /* *ProtocolCount */ *ProtocolOpinionGathering) SetTimeout(t time.Duration) {
	p.timeoutMu.Lock()
	p.timeout = t
	p.timeoutMu.Unlock()
}

// Timeout returns the current timeout
func (p /* *ProtocolCount */ *ProtocolOpinionGathering) Timeout() time.Duration {
	p.timeoutMu.Lock()
	defer p.timeoutMu.Unlock()
	return p.timeout
}

//Xor
func xor(x []byte, state bool) string {
	signedStringLength := 13
	y := []byte{}
	s := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if state {
		for index := 0; (index*13 + 13) < len(x); index++ {
			y = x[index*13 : index*13+13]
			log.Lvl1("\n node", index, "signed string:", y)
			s = safeXORBytes(s, y, signedStringLength)
		}
		log.Lvl1("\n The final xored string is:", s)
		return "done"
	} else {
		return "invalid signiture"
	}

}
func safeXORBytes(s, b []byte, n int) (result []byte) {
	for i := 0; i < n; i++ {
		s[i] = s[i] ^ b[i]
	}
	return s
}
func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
