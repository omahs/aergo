package contract

import (
	"sync"
	"net"
	"fmt"
	"math"
	"crypto/rand"
	"os"
	"os/exec"
	"time"
)

var maxInstances int
var getCh chan *VmInstance
var freeCh chan *VmInstance
var once sync.Once

func StartVMPool(numInstances int) {
	once.Do(func() {
		maxInstances = numInstances
		// create channels for getting and freeing vm instances
		getCh = make(chan *VmInstance, numInstances)
		freeCh = make(chan *VmInstance, numInstances)
		// start a goroutine to manage the vm instances
		go vmPoolRoutine()
	})
}

func vmPoolRoutine() {

	// create vm instances
	spawnVmInstances(maxInstances)

	// wait for instances to be released
	for {
		select {
		case vmInstance := <-freeCh:
			// close the vm instance
			vmInstance.close()
			// replenish the pool
			repopulatePool()
		}
	}

}

//--------------------------------------------------------------------//
// exported functions

func GetVmInstance() *VmInstance {
	vmInstance := <-getCh
	ctrLgr.Trace().Msg("VmInstance acquired")
	return vmInstance
}

func FreeVmInstance(vmInstance *VmInstance) {
	if vmInstance != nil {
		freeCh <- vmInstance
		ctrLgr.Trace().Msg("VmInstance released")
	}
}

// flush and renew all vm instances
func FlushVmInstances() {
	// first retrieve all vm instances, so when releasing the first one
	// the pool is empty and then it will spawn many at once
	list := []*VmInstance{}
	num := len(getCh)
	for i := 0; i < num; i++ {
		vmInstance := GetVmInstance()
		list = append(list, vmInstance)
	}
	for _, vmInstance := range list {
		FreeVmInstance(vmInstance)
	}
}

//--------------------------------------------------------------------//
// VmInstance type

type VmInstance struct {
	id         uint64
	socketName string
	secretKey  [32]byte
	listener   *net.UnixListener
	conn       *net.UnixConn
	pid        int
	used       bool
}

// pool of vm instances
var pool []*VmInstance

// repopulate the pool with new vm instances
func repopulatePool() {

	for {
		// check how many instances are available on the getCh
		numAvailable := len(getCh)
		// if there are less than maxInstances, create new ones
		if numAvailable < maxInstances {
			spawnVmInstances(maxInstances - numAvailable)
		} else {
			break
		}
	}

}

// spawn a number of vm instances
func spawnVmInstances(num int) {

	for i := 0; i < num; i++ {
		// get a random id
		var id uint64
		for {
			id = rand.Uint64()
			// check if it is already used
			for _, vmInstance := range pool {
				if vmInstance.id == id {
					continue
				}
			}
			break
		}

		// get a random secret key
		secretKey := [32]byte{}
		rand.Read(secretKey[:])

		// get a random name for the abstract unix domain socket
		socketName := fmt.Sprintf("aergo-vm-%x", id)

		// create an abstract unix domain socket listener
		listener, err := net.Listen("unix", "\x00"+socketName)
		if err != nil {
			ctrLgr.Error().Msg("Failed to create unix domain socket listener")
			// try again
			i--
			continue
		}

		// spawn the exernal VM executable process
		cmd := exec.Command("vm/vm-lua", currentForkVersion, PubNet, socketName, secretKey)
		err = cmd.Start()
		if err != nil {
			ctrLgr.Error().Msg("Failed to spawn external VM process")
			listener.Close()
			// try again
			i--
			continue
		}
		// get the process id
		pid := cmd.Process.Pid
		ctrLgr.Trace().Msgf("Spawned external VM process with pid: %d", pid)

		// create a vm instance object
		vmInstance := &VmInstance{
			id:         id,
			socketName: socketName,
			secretKey:  secretKey,
			listener:   listener,
			conn:       nil,
			pid:        pid,
			used:       false,
		}

		// add the vm instance to the pool
		pool = append(pool, vmInstance)

	}

	// keep track of the instances that should be removed
	instancesToRemove := []*VmInstance{}

	// keep track of the new instances that are connected
	instancesToRead := []*VmInstance{}

	// the timeout is 100 milliseconds for each vm instance
	timeout := time.Millisecond * 100 * num
	if timeout < time.Second {
		timeout = time.Second
	}
	// set a common deadline for the accept calls
	deadline := time.Now().Add(timeout)

	// iterate over all instances
	for _, vmInstance := range pool {
		// if this VM instance is not yet connected
		if vmInstance.conn == nil {
			// set a deadline for the accept call
			vmInstance.listener.SetDeadline(deadline)
			// wait for the incoming connection
			vmInstance.conn, err = vmInstance.listener.Accept()
			if err == nil {
				// connection accepted
				instancesToRead = append(instancesToRead, vmInstance)
			} else {
				ctrLgr.Error().Msgf("Failed to accept incoming connection: %v", err)
				instancesToRemove = append(instancesToRemove, vmInstance)
			}
		}
	}

	// remove the instances that are not connected
	for _, vmInstance := range instancesToRemove {
		vmInstance.close()
	}

	// iterate over the instances that are connected
	for _, vmInstance := range instancesToRead {
		// wait for a message from the vm instance
		msg, err := waitForMessage(vmInstance.conn, deadline)
		if err != nil {
			ctrLgr.Error().Msgf("Failed to read incoming message: %v", err)
			vmInstance.close()
			continue
		}
		// check if the data is valid
		if !isValidMessage(vmInstance, msg) {
			ctrLgr.Error().Msg("Invalid message received")
			vmInstance.close()
			continue
		}
		// send the instance to the getCh
		getCh <- vmInstance
	}

}

func isValidMessage(vmInstance *VmInstance, msg string) bool {


}

// this should ONLY be called by the vmPoolRoutine. use FreeVmInstance() to release a vm instance
func (vmInstance *VmInstance) close() {
	if vmInstance != nil {
		// close the connections
		vmInstance.listener.Close()
		vmInstance.conn.Close()
		// terminate the process
		process, err := os.FindProcess(vmInstance.pid)
		if err == nil {
			process.Kill()
		}
		// remove the vm instance from the pool
		for i, v := range pool {
			if v == vmInstance {
				pool = append(pool[:i], pool[i+1:]...)
				break
			}
		}
	}
}