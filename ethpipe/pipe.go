package ethpipe

import (
	"strings"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
)

var logger = ethlog.NewLogger("PIPE")

type Pipe struct {
	obj          ethchain.EthManager
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	world        *world
}

func New(obj ethchain.EthManager) *Pipe {
	pipe := &Pipe{
		obj:          obj,
		stateManager: obj.StateManager(),
		blockChain:   obj.BlockChain(),
	}
	pipe.world = NewWorld(pipe)

	return pipe
}

func (self *Pipe) Balance(addr []byte) *ethutil.Value {
	return ethutil.NewValue(self.World().safeGet(addr).Balance)
}

func (self *Pipe) Nonce(addr []byte) uint64 {
	return self.World().safeGet(addr).Nonce
}

func (self *Pipe) Execute(addr []byte, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	return self.ExecuteObject(self.World().safeGet(addr), data, value, gas, price)
}

func (self *Pipe) ExecuteObject(object *ethstate.StateObject, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	var (
		initiator = ethstate.NewStateObject([]byte{0})
		state     = self.World().State().Copy()
		block     = self.blockChain.CurrentBlock
	)

	vm := ethvm.New(NewEnv(state, block, value.BigInt(), initiator.Address()))

	closure := ethvm.NewClosure(initiator, object, object.Code, gas.BigInt(), price.BigInt())
	ret, _, err := closure.Call(vm, data)

	return ret, err
}

func (self *Pipe) Block(hash []byte) *ethchain.Block {
	return self.blockChain.GetBlock(hash)
}

func (self *Pipe) Storage(addr, storageAddr []byte) *ethutil.Value {
	return self.World().safeGet(addr).GetStorage(ethutil.BigD(storageAddr))
}

func (self *Pipe) ToAddress(priv []byte) []byte {
	pair, err := ethcrypto.NewKeyPairFromSec(priv)
	if err != nil {
		return nil
	}

	return pair.Address()
}

func (self *Pipe) TransactString(key *ethcrypto.KeyPair, rec string, value, gas, price *ethutil.Value, data []byte) error {
	// Check if an address is stored by this address
	var hash []byte
	addr := self.World().Config().Get("NameReg").StorageString(rec).Bytes()
	if len(addr) > 0 {
		hash = addr
	} else if ethutil.IsHex(rec) {
		hash = ethutil.Hex2Bytes(rec[2:])
	} else {
		hash = ethutil.Hex2Bytes(rec)
	}

	return self.Transact(key, hash, value, gas, price, data)
}

func (self *Pipe) Transact(key *ethcrypto.KeyPair, rec []byte, value, gas, price *ethutil.Value, data []byte) error {
	var hash []byte
	var contractCreation bool
	if rec == nil {
		contractCreation = true
	}

	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		script, err := ethutil.Compile(string(data), false)
		if err != nil {
			return err
		}

		tx = ethchain.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), script)
	} else {
		data := ethutil.StringToByteFunc(string(data), func(s string) (ret []byte) {
			slice := strings.Split(s, "\n")
			for _, dataItem := range slice {
				d := ethutil.FormatData(dataItem)
				ret = append(ret, d...)
			}
			return
		})

		tx = ethchain.NewTransactionMessage(hash, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	acc := self.stateManager.TransState().GetOrNewStateObject(key.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.stateManager.TransState().UpdateStateObject(acc)

	tx.Sign(key.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		logger.Infof("Contract addr %x", tx.CreationAddress())
	}

	return nil
}
