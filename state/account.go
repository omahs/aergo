package state

import (
	"fmt"
	"math/big"

	key "github.com/aergoio/aergo/v2/account/key/crypto"
	"github.com/aergoio/aergo/v2/state/statedb"
	"github.com/aergoio/aergo/v2/types"
	"github.com/ethereum/go-ethereum/common"
	ethstate "github.com/ethereum/go-ethereum/core/state"
)

type AccountState struct {
	luaStates *statedb.StateDB
	ethStates *ethstate.StateDB

	id    []byte
	aid   types.AccountID
	ethId common.Address

	oldState *types.State
	newState *types.State
	newOne   bool
	deploy   int8
}

const (
	deployFlag = 0x01 << iota
	redeployFlag
)

func (as *AccountState) ID() []byte {
	if len(as.id) < types.AddressLength {
		as.id = types.AddressPadding(as.id)
	}
	return as.id
}

func (as *AccountState) AccountID() types.AccountID {
	return as.aid
}

func (as *AccountState) EthID() common.Address {
	return as.ethId
}

func (as *AccountState) State() *types.State {
	return as.newState
}

func (as *AccountState) SetNonce(nonce uint64) {
	as.newState.Nonce = nonce
}

func (as *AccountState) GetNonce() uint64 {
	return as.newState.Nonce
}

func (as *AccountState) Balance() *big.Int {
	return new(big.Int).SetBytes(as.newState.Balance)
}

func (as *AccountState) AddBalance(amount *big.Int) {
	balance := new(big.Int).SetBytes(as.newState.Balance)
	as.newState.Balance = new(big.Int).Add(balance, amount).Bytes()
}

func (as *AccountState) SubBalance(amount *big.Int) {
	balance := new(big.Int).SetBytes(as.newState.Balance)
	as.newState.Balance = new(big.Int).Sub(balance, amount).Bytes()
}

func (as *AccountState) RP() uint64 {
	return as.newState.SqlRecoveryPoint
}

func (as *AccountState) IsNew() bool {
	return as.newOne
}

func (as *AccountState) IsContract() bool {
	return len(as.State().CodeHash) > 0
}

func (as *AccountState) IsDeploy() bool {
	return as.deploy&deployFlag != 0
}

func (as *AccountState) SetRedeploy() {
	as.deploy = deployFlag | redeployFlag
}

func (as *AccountState) IsRedeploy() bool {
	return as.deploy&redeployFlag != 0
}

func (as *AccountState) Reset() {
	as.newState = as.oldState.Clone()
}

func (as *AccountState) PutState() error {
	if err := as.luaStates.PutState(as.aid, as.newState); err != nil {
		return err
	}
	if as.ethStates != nil {
		as.ethStates.SetBalance(as.ethId, new(big.Int).SetBytes(as.newState.Balance))
		as.ethStates.SetNonce(as.ethId, as.newState.Nonce)
	}
	return nil
}

func (as *AccountState) ClearAid() {
	as.aid = statedb.EmptyAccountID
}

//----------------------------------------------------------------------------------------------//
// global functions

func CreateAccountState(id []byte, states *statedb.StateDB, ethStates *ethstate.StateDB) (*AccountState, error) {
	v, err := GetAccountState(id, states, ethStates)
	if err != nil {
		return nil, err
	}
	if !v.newOne {
		return nil, fmt.Errorf("account(%s) aleardy exists", types.EncodeAddress(v.ID()))
	}
	v.newState.SqlRecoveryPoint = 1
	v.deploy = deployFlag
	return v, nil
}

func GetAccountState(id []byte, states *statedb.StateDB, ethStates *ethstate.StateDB) (*AccountState, error) {
	aid := types.ToAccountID(id)
	st, err := states.GetState(aid)
	if err != nil {
		return nil, err
	}
	ethAccount := key.NewAddressEth(id)

	if st == nil {
		if states.Testmode {
			amount := new(big.Int).Add(types.StakingMinimum, types.StakingMinimum)
			return &AccountState{
				luaStates: states,
				ethStates: ethStates,
				id:        id,
				aid:       aid,
				ethId:     ethAccount,
				oldState:  &types.State{Balance: amount.Bytes()},
				newState:  &types.State{Balance: amount.Bytes()},
				newOne:    true,
			}, nil
		}
		return &AccountState{
			luaStates: states,
			ethStates: ethStates,
			id:        id,
			aid:       aid,
			ethId:     ethAccount,
			oldState:  &types.State{},
			newState:  &types.State{},
			newOne:    true,
		}, nil
	}
	return &AccountState{
		luaStates: states,
		ethStates: ethStates,
		id:        id,
		aid:       aid,
		ethId:     ethAccount,
		oldState:  st,
		newState:  st.Clone(),
	}, nil
}

func InitAccountState(id []byte, sdb *statedb.StateDB, ethsdb *ethstate.StateDB, old *types.State, new *types.State) *AccountState {
	return &AccountState{
		luaStates: sdb,
		ethStates: ethsdb,
		id:        id,
		aid:       types.ToAccountID(id),
		ethId:     key.NewAddressEth(id),
		oldState:  old,
		newState:  new,
	}
}