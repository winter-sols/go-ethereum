package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

// States within the ethereum protocol are used to store anything
// within the merkle trie. States take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type State struct {
	// The trie for this structure
	trie *ethutil.Trie
	// Nested states
	states map[string]*State

	stateObjects map[string]*StateObject

	manifest *Manifest
}

// Create a new state from a given trie
func NewState(trie *ethutil.Trie) *State {
	return &State{trie: trie, states: make(map[string]*State), stateObjects: make(map[string]*StateObject), manifest: NewManifest()}
}

/*
// Resets the trie and all siblings
func (s *State) Reset() {
	s.trie.Undo()

	// Reset all nested states
	for _, state := range s.states {
		state.Reset()
	}
}

// Syncs the trie and all siblings
func (s *State) Sync() {
	// Sync all nested states
	for _, state := range s.states {
		state.Sync()
	}

	s.trie.Sync()
}
*/

// Resets the trie and all siblings
func (s *State) Reset() {
	s.trie.Undo()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.state == nil {
			continue
		}

		stateObject.state.Reset()
	}
}

// Syncs the trie and all siblings
func (s *State) Sync() {
	// Sync all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.state == nil {
			continue
		}

		stateObject.state.Sync()
	}

	s.trie.Sync()
}

// Purges the current trie.
func (s *State) Purge() int {
	return s.trie.NewIterator().Purge()
}

func (s *State) EachStorage(cb ethutil.EachCallback) {
	it := s.trie.NewIterator()
	it.Each(cb)
}

/*
func (s *State) GetStateObject(addr []byte) *StateObject {
	data := s.trie.Get(string(addr))
	if data == "" {
		return nil
	}

	stateObject := NewStateObjectFromBytes(addr, []byte(data))

	// Check if there's a cached state for this contract
	cachedStateObject := s.states[string(addr)]
	if cachedStateObject != nil {
		//fmt.Printf("get cached #%d %x addr: %x\n", cachedStateObject.trie.Cache().Len(), cachedStateObject.Root(), addr[0:4])
		stateObject.state = cachedStateObject
	}

	return stateObject
}

// Updates any given state object
func (s *State) UpdateStateObject(object *StateObject) {
	addr := object.Address()

	if object.state != nil && s.states[string(addr)] == nil {
		s.states[string(addr)] = object.state
	}

	ethutil.Config.Db.Put(ethutil.Sha3Bin(object.Script()), object.Script())

	s.trie.Update(string(addr), string(object.RlpEncode()))

	s.manifest.AddObjectChange(object)
}

func (s *State) GetAccount(addr []byte) (account *StateObject) {
	data := s.trie.Get(string(addr))
	if data == "" {
		account = NewAccount(addr, big.NewInt(0))
	} else {
		account = NewStateObjectFromBytes(addr, []byte(data))
	}

	// Check if there's a cached state for this contract
	cachedStateObject := s.states[string(addr)]
	if cachedStateObject != nil {
		account.state = cachedStateObject
	}

	return
}
*/

func (self *State) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()

	if self.stateObjects[string(addr)] == nil {
		self.stateObjects[string(addr)] = stateObject
	}

	ethutil.Config.Db.Put(ethutil.Sha3Bin(stateObject.Script()), stateObject.Script())

	self.trie.Update(string(addr), string(stateObject.RlpEncode()))

	self.manifest.AddObjectChange(stateObject)
}

func (self *State) GetStateObject(addr []byte) *StateObject {
	stateObject := self.stateObjects[string(addr)]
	if stateObject != nil {
		return stateObject
	}

	data := self.trie.Get(string(addr))
	if len(data) == 0 {
		return nil
	}

	stateObject = NewStateObjectFromBytes(addr, []byte(data))
	self.stateObjects[string(addr)] = stateObject

	return stateObject
}

func (self *State) GetOrNewStateObject(addr []byte) *StateObject {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		stateObject = NewStateObject(addr)
		self.stateObjects[string(addr)] = stateObject
	}

	return stateObject
}

func (self *State) GetAccount(addr []byte) *StateObject {
	return self.GetOrNewStateObject(addr)
}

func (s *State) Cmp(other *State) bool {
	return s.trie.Cmp(other.trie)
}

/*
func (s *State) Copy() *State {
	state := NewState(s.trie.Copy())
	for k, subState := range s.states {
		state.states[k] = subState.Copy()
	}

	return state
}
*/
func (self *State) Copy() *State {
	state := NewState(self.trie.Copy())
	for k, stateObject := range self.stateObjects {
		state.stateObjects[k] = stateObject.Copy()
	}

	return state
}

func (s *State) Snapshot() *State {
	return s.Copy()
}

func (s *State) Revert(snapshot *State) {
	s.trie = snapshot.trie
	s.states = snapshot.states
}

func (s *State) Put(key, object []byte) {
	s.trie.Update(string(key), string(object))
}

func (s *State) Root() interface{} {
	return s.trie.Root
}

// Object manifest
//
// The object manifest is used to keep changes to the state so we can keep track of the changes
// that occurred during a state transitioning phase.
type Manifest struct {
	// XXX These will be handy in the future. Not important for now.
	objectAddresses  map[string]bool
	storageAddresses map[string]map[string]bool

	objectChanges  map[string]*StateObject
	storageChanges map[string]map[string]*big.Int
}

func NewManifest() *Manifest {
	m := &Manifest{objectAddresses: make(map[string]bool), storageAddresses: make(map[string]map[string]bool)}
	m.Reset()

	return m
}

func (m *Manifest) Reset() {
	m.objectChanges = make(map[string]*StateObject)
	m.storageChanges = make(map[string]map[string]*big.Int)
}

func (m *Manifest) AddObjectChange(stateObject *StateObject) {
	m.objectChanges[string(stateObject.Address())] = stateObject
}

func (m *Manifest) AddStorageChange(stateObject *StateObject, storageAddr []byte, storage *big.Int) {
	if m.storageChanges[string(stateObject.Address())] == nil {
		m.storageChanges[string(stateObject.Address())] = make(map[string]*big.Int)
	}

	m.storageChanges[string(stateObject.Address())][string(storageAddr)] = storage
}
