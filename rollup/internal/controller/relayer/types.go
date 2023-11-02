package relayer

import (
	"fmt"
	"sort"
	"sync"
	"unicode"
)

const (
	MonitoredProofHashTxCode = 0
	MonitoredProofTxCode     = 1
)

type finalProofMsg struct {
	proverName     string
	proverID       string
	recursiveProof *state.Proof
	finalProof     *pb.FinalProof
}

type proofHash struct {
	hash                   string
	batchNumber            uint64
	batchNumberFinal       uint64
	monitoredProofHashTxID string
}

type sendFailProofMsg struct {
	BatchNumber      uint64
	BatchNumberFinal uint64
}

type finalProofMsgList []finalProofMsg

func (h finalProofMsgList) Len() int { return len(h) }
func (h finalProofMsgList) Less(i, j int) bool {
	return h[i].recursiveProof.BatchNumberFinal < h[j].recursiveProof.BatchNumberFinal
}
func (h finalProofMsgList) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func FirstToUpper(s string) string {
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

type SequenceList []state.Sequence

func (s SequenceList) Len() int { return len(s) }
func (s SequenceList) Less(i, j int) bool {
	return s[i].FromBatchNumber < s[j].FromBatchNumber
}
func (s SequenceList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type monitoredTxInfo struct {
	typeCode         int
	batchNumber      uint64
	batchNumberFinal uint64
	monitoredTxID    string
}

func (fpm finalProofMsg) genKey() string {
	return fmt.Sprintf("final_proof_%v_%v", fpm.recursiveProof.BatchNumber, fpm.recursiveProof.BatchNumberFinal)
}

func (fpm finalProofMsg) getCompareValue() uint64 {

	return fpm.recursiveProof.BatchNumberFinal
}

func (mt monitoredTxInfo) genKey() string {
	return mt.monitoredTxID
}

func (mt monitoredTxInfo) getCompareValue() uint64 {
	return mt.batchNumberFinal
}

func (ph proofHash) genKey() string {
	return ph.monitoredProofHashTxID
}

func (ph proofHash) getCompareValue() uint64 {
	return ph.batchNumberFinal
}

type cacheItem interface {
	genKey() string
	getCompareValue() uint64
	//compare(other cacheItem) bool
}

type cacheItemList []cacheItem

func (cil cacheItemList) Len() int { return len(cil) }

func (cil cacheItemList) Less(i, j int) bool {
	return cil[i].getCompareValue() < cil[j].getCompareValue()
}

func (cil cacheItemList) Swap(i, j int) { cil[i], cil[j] = cil[j], cil[i] }

type senderCache struct {
	rwMutex  *sync.RWMutex
	itemList cacheItemList
	existMap map[string]struct{}
}

func newSenderCache() senderCache {
	return senderCache{
		rwMutex:  &sync.RWMutex{},
		itemList: make(cacheItemList, 0),
		existMap: make(map[string]struct{}, 0),
	}
}

func (sc *senderCache) getLen() int {
	sc.rwMutex.Lock()
	defer sc.rwMutex.Unlock()
	return sc.itemList.Len()

}

func (sc *senderCache) insertMsgIntoCache(msg cacheItem) {
	sc.rwMutex.Lock()
	defer sc.rwMutex.Unlock()
	key := msg.genKey()
	if _, ok := sc.existMap[key]; !ok {
		sc.itemList = append(sc.itemList, msg)
		sc.existMap[key] = struct{}{}
		sort.Sort(sc.itemList)
	}
}

func (sc *senderCache) insertBatchMsgIntoCache(msgs []cacheItem) {
	sc.rwMutex.Lock()
	defer sc.rwMutex.Unlock()
	for _, msg := range msgs {
		key := msg.genKey()
		if _, ok := sc.existMap[key]; !ok {
			sc.itemList = append(sc.itemList, msg)
			sc.existMap[key] = struct{}{}
		}
	}
	sort.Sort(sc.itemList)
}

func (sc *senderCache) upMsgFromCache() *cacheItem {
	sc.rwMutex.Lock()
	defer sc.rwMutex.Unlock()
	length := len(sc.itemList)
	if length > 0 {
		msg := sc.itemList[0]
		if length > 1 {
			sc.itemList = sc.itemList[1:]
		} else {
			sc.itemList = make(cacheItemList, 0)
		}
		key := msg.genKey()
		delete(sc.existMap, key)
		return &msg
	} else {
		return nil
	}
}
