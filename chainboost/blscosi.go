package chainboost

// import (
// 	"bytes"
// 	"crypto/sha256"
// 	"encoding/binary"
// 	"encoding/hex"
// 	"fmt"

// 	"math/rand"
// 	"strconv"
// 	"sync"
// 	"time"

// 	onet "github.com/basedfs"
// 	"github.com/basedfs/blockchain"
// 	"github.com/basedfs/log"
// 	"github.com/basedfs/network"
// 	"github.com/basedfs/por"
// 	"github.com/basedfs/vrf"
// 	"github.com/xuri/excelize/v2"
// )

//     err = cosiProtocol.Start()
// 	if err != nil {
// 		return err
// 	}

// 	select {
// 	case sig := <-cosiProtocol.FinalSignature:
// 		pubs := roster.ServicePublics(testServiceName)
// 		return BdnSignature(sig).Verify(testSuite, cosiProtocol.Msg, pubs)
// 	case <-time.After(2 * time.Second):
// 	}
