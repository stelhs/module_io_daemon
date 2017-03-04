package nmea0183

import (
	"fmt"
	"strings"
)

type Nmea0183 struct {
	buf []byte
	rx_carry bool
	start bool
}

type Nmea_msg struct {
	Ti string
	Si string
	Args []int
}

// Constructor of Nmea0183 transiver
func New() *Nmea0183 {
	t := new(Nmea0183)
	t.buf = make([]byte, 0, 256)
	return t
}

func (t *Nmea0183) calc_checksum(buf string) byte {
    var sum byte = 0
    for _, v := range buf {
        sum += byte(v);
    }
    return sum;
}

func (t *Nmea0183) parse() *Nmea_msg {
    buf := string(t.buf)

    parts := strings.Split(buf, "*")
    if len(parts) > 1 {
	    var check_sum byte
		fmt.Sscanf(parts[1], "%x", &check_sum)
		buf = parts[0]
		if t.calc_checksum(buf) != check_sum {
			return nil
		}
    }
    
    var msg Nmea_msg
    parts = strings.Split(buf, ",")
    first := true
    for _, v := range parts {
    	if first {
    		first = false
    		if len(v) != 5 {
    			return nil
    		}
    		msg.Ti = string([]byte(v)[:2])
    		msg.Si = string([]byte(v)[2:5])
    		continue
    	}
    	
    	var arg int
    	fmt.Sscanf(v, "%d", &arg)
    	msg.Args = append(msg.Args, arg)
    }
    
	return &msg
}

// Push byte data into Nmea0183 parser
func (t *Nmea0183) Push_rxb(rxb byte) *Nmea_msg {
	switch rxb {
	case '$':
        t.rx_carry = false
        t.buf = t.buf[0:0]
        t.start = true
	
	case '\r', '\n':
        if t.rx_carry || !t.start {
	        return nil
        }
		
        t.rx_carry = true
        t.start = false
        return t.parse()

    default:
        t.rx_carry = false
        if !t.start {
	        t.buf = t.buf[0:0]
            return nil;
        }

        if (len(t.buf) == cap(t.buf)) {
            t.start = false
            return nil;
        }

        t.buf = append(t.buf, rxb)
        return nil;
	}
	
	return nil
}


// Create nmea0183's text message from ti,si,args components 
func (t *Nmea0183) Create_msg(ti string, si string, args []int) string {
	msg := make([]byte, 0, 64)
	msg = []byte(fmt.Sprintf("$%s%s", ti, si))
	for _, arg := range args {
		msg = []byte(fmt.Sprintf("%s,%d", msg, arg))
	}
	msg = []byte(fmt.Sprintf("$%s\n", msg))
	return string(msg)
}



