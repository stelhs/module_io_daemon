package mod_io

import (
	"nmea0183"
	"os"
	"os/exec"
	"container/list"
	"sync"
	"time"
	"conf"
	"fmt"
)

type Mod_io struct {
	sync.Mutex
	nmea *nmea0183.Nmea0183
	dev *os.File
	tx chan string
	rx_queue *list.List
	rx_recepient_channels *list.List
}


func New(iocfg *conf.Module_io_cfg) (*Mod_io, error) {
	var err error
	
	mio := new(Mod_io)
	mio.tx = make(chan string, 64)
	mio.rx_queue = list.New()
	mio.rx_recepient_channels = list.New()
	
	mio.dev, err = os.OpenFile(iocfg.Uart_dev, 
						os.O_RDWR | os.O_APPEND, 0660)
	if err != nil {
		return nil, fmt.Errorf("can't open file %s", iocfg.Uart_dev)
	}
	
	mio.nmea = nmea0183.New()
	
	err = exec.Command("bash", "-c", "stty -F" + iocfg.Uart_dev + 
						" " + iocfg.Uart_speed + " raw -echo").Run()
	if err != nil {
		return nil, fmt.Errorf("can't set tty params: %v", err)
	}
	
	go mio.Receiver_thread()
	go mio.Transmitter_thread()
	return mio, err
}


func (mio *Mod_io) Receiver_thread() {
	var buf [64]byte
	var err error
	var count int
	
	for {
		count, err = mio.dev.Read(buf[:])
		if err != nil {
			continue; // TODO:
		}
		
		if count <= 0 {
			continue; // TODO:
		}
		
		for _, byte := range buf[:count] {
			msg := mio.nmea.Push_rxb(byte)
			if msg == nil {
				continue	
			}

			mio.Lock()
			mio.rx_queue.PushBack(msg)

			for e := mio.rx_recepient_channels.Front(); e != nil; e = e.Next() {
				chain, _ := e.Value.(chan bool)
				chain <- true
			}
			mio.Unlock()


		}
	}
}


func (mio *Mod_io) Transmitter_thread() {
	var count int

	for {
		msg := <- mio.tx
		count = 0
		for count < len(msg) {
			var err error
			count, err = mio.dev.Write([]byte(msg))
			if err != nil {
				panic("Can't write to UART")
			}
		}
	}
}

// Send nmea0183 message to transmitter
func (mio *Mod_io) Send_cmd(request_id int, ti string, si string, args []int) {
	// Remove incomming packet with request_id from rx_queue
	mio.Lock()
	for e := mio.rx_queue.Front(); e != nil; e = e.Next() {
		msg, _ := e.Value.(*nmea0183.Nmea_msg)

		if msg.Request_id == request_id {
			println(fmt.Sprintf("find request_id: %d", msg.Request_id))
			mio.rx_queue.Remove(e)
		}
	}
	mio.Unlock()

	args = append([]int{request_id}, args...)
	msg := mio.nmea.Create_msg(ti, si, args)
	mio.tx <- msg
}

// Set outport new state 
func (mio *Mod_io) Relay_set_state(request_id int, port_num int, state int) error {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd(request_id, "PC", "RWS", []int{port_num, state})
		msg := mio.Recv(request_id, []string{"SOP"}, 500)
		if msg == nil {
			continue
		}
		
		if msg.Args[1] != port_num {
			continue
		}
		
		if msg.Args[2] != state {
			continue
		}
		
		return nil
	}
	return fmt.Errorf("mod_io: can't set relay state")	
}

// Get output port state
func (mio *Mod_io) Get_output_port_state(request_id int, port_num int) (int, error) {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd(request_id, "PC", "RRS", []int{port_num})
		msg := mio.Recv(request_id, []string{"SOP"}, 500)

		if msg == nil {
			continue
		}

		if msg.Args[1] != port_num {
			continue
		}

		return msg.Args[2], nil
	}
	return 0, fmt.Errorf("mod_io: can't get output state")
}


// Get input port state
func (mio *Mod_io) Get_input_port_state(request_id int, port_num int) (int, error) {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd(request_id, "PC", "RIP", []int{port_num})
		msg := mio.Recv(request_id, []string{"SIP"}, 500)
		if msg == nil {
			continue
		}

		if msg.Args[1] != port_num {
			continue
		}

		return msg.Args[2], nil
	}
	return 0, fmt.Errorf("mod_io: can't get input state")	
}


// Set WDT state
func (mio *Mod_io) Wdt_set_state(request_id int, state int) error {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd(request_id, "PC", "WDC", []int{state})
		msg := mio.Recv(request_id, []string{"WDS"}, 500)
		if msg == nil {
			continue
		}

		if (msg.Args[1] & 1) != state {
			continue
		}

		return nil
	}
	return fmt.Errorf("mod_io: can't set watchdog state %d", state)
}


// WDT reset
func (mio *Mod_io) Wdt_reset() {
	mio.Send_cmd(0, "PC", "WRS", []int{})
}

func (mio *Mod_io) recv_from_queue(request_id int, si string) *nmea0183.Nmea_msg {
	mio.Lock()
	for e := mio.rx_queue.Front(); e != nil; e = e.Next() {
		msg, _ := e.Value.(*nmea0183.Nmea_msg)

		if msg.Request_id != request_id {
			continue
		}

		if len(si) == 0 {
			mio.rx_queue.Remove(e)
			mio.Unlock()
			return msg
		}

		if msg.Si == si {
			mio.rx_queue.Remove(e)
			mio.Unlock()
			return msg
		}
	}
	mio.Unlock()
	return nil
}


// Receive nmea0183 message by mask
func (mio *Mod_io) Recv(request_id int, si_list []string, timeout uint) *nmea0183.Nmea_msg {
    var msg *nmea0183.Nmea_msg
    for _, si := range si_list {
        msg = mio.recv_from_queue(request_id, si)
        if msg == nil {
            continue
        }
        break
    }
	if msg != nil {
		return msg
	}

	rx_flag := make(chan bool, 1)
	mio.Lock()
	rx_flag_queue_item := mio.rx_recepient_channels.PushBack(rx_flag)
	mio.Unlock()

	if timeout == 0 {
		for {
			<- rx_flag
            for _, si := range si_list {
                msg = mio.recv_from_queue(request_id, si)
                if msg == nil {
                    continue
                }
                break
            }

            if msg == nil {
                continue
            }

			mio.Lock()
			mio.rx_recepient_channels.Remove(rx_flag_queue_item)
			mio.Unlock()
			return msg
		}
	}

	for {
		select {
		case <- rx_flag:
            for _, si := range si_list {
                msg = mio.recv_from_queue(request_id, si)
                if msg == nil {
                    continue
                }
                break
            }
			if msg == nil {
				continue
			}
			mio.Lock()
			mio.rx_recepient_channels.Remove(rx_flag_queue_item)
			mio.Unlock()
			return msg

		case <- time.After(time.Millisecond * 
							time.Duration(timeout)):
			mio.Lock()
			mio.rx_recepient_channels.Remove(rx_flag_queue_item)
			mio.Unlock()
			return nil
		}
	}

	return nil
}
