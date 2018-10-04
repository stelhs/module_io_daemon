package main 

import (
	"fmt"
	"mod_io"
    "conf"
    "os"
    "os/exec"
    "strings"
    "net"
)

type module_io_daemon struct {
	cfg *conf.Module_io_cfg
	mio *mod_io.Mod_io
	last_client_id int
}


func main() {
	var err error
	var md module_io_daemon

	md.last_client_id = 1

	md.cfg, err = conf.Conf_parse()
    if err != nil {
        panic(fmt.Sprintf("main: can't get configuration: %v", err))
    }

	md.mio, err = mod_io.New(md.cfg)
	if err != nil {
		panic(fmt.Sprintf("main: can't create mod_io: %v", err))
	}

	err = os.Chdir(md.cfg.Exec_path);
	if err != nil {
		panic(fmt.Sprintf("main: can't change current dir: %v", err))
	}

	go md.do_listen_for_connections()

	// waiting actions
	for {
		msg := md.mio.Recv(0, []string{"AIP", "ASP"}, 0)
		fmt.Println("recv msg = ", msg)
		if msg == nil {
            continue
        }

        if msg.Si == "AIP" {
            run_action_script(md.cfg.Exec_script, "io_input", msg.Args[1], msg.Args[2])
		}

        if msg.Si == "ASP" {
            run_action_script(md.cfg.Exec_script, "restart", 0, 0)
        }
	}
}


func run_action_script(script string, action string, port int, state int) {
    p := exec.Command(script, fmt.Sprintf("%s", action),
						       fmt.Sprintf("%d", port),
							   fmt.Sprintf("%d", state))

    stdin, err := p.StdinPipe()
    if err != nil {
    	panic(fmt.Sprintf("main: can't run script: %s: %v", script, err))
    }
    defer stdin.Close()

    p.Stdout = os.Stdout
    p.Stderr = os.Stderr

    if err = p.Start(); err != nil { 
    	panic(fmt.Sprintf("main: can't run script: %s: %v", script, err))
    }

    p.Wait()
}


func (md *module_io_daemon) do_listen_for_connections() {
	os.Remove(md.cfg.Control_socket)
    l, err := net.Listen("unix", md.cfg.Control_socket)
    if err != nil {
    	panic(fmt.Sprintf("main: can't listen socket: %s: %v", 
			    			md.cfg.Control_socket, err))
    }

    for {
        fd, err := l.Accept()
        if err != nil {
	    	panic(fmt.Sprintf("main: can't accept new connection: %v", err))
        }

		md.last_client_id++
		if md.last_client_id > 255 {
			md.last_client_id = 1
		}
        go md.do_process_cmd(fd, md.last_client_id)
    }
}

func (md *module_io_daemon) do_process_cmd(fd net.Conn, client_id int) {
	defer fd.Close()
	
	ret := ""
    buf := make([]byte, 512)
    nr, err := fd.Read(buf)
    if err != nil {
        return
    }

	// read incomming data
    input_data := buf[0:nr]
	// split by rows
	queries := strings.Split(string(input_data), "\n")

	// analysis rows as queries
	for _, query := range queries {
        query = strings.Trim(query, " ")
        if query == "" {
        	continue
        }
        
        // split query by args
        println("query = ", query)
        cmd, args := parse_query(query)
        switch cmd {
        case "relay_set":
	        var port, new_state int
	        fmt.Sscanf(args[0], "%d", &port)
	        fmt.Sscanf(args[1], "%d", &new_state)
	        err := md.mio.Relay_set_state(client_id, port, new_state)
	        if err == nil {
		        ret = "ok"
	        } else {
	        	ret = fmt.Sprintf("%v", err)
	        }
	        break;	

        case "relay_get":
	        var port int
	        fmt.Sscanf(args[0], "%d", &port)
	        state, err := md.mio.Get_output_port_state(client_id, port)
	        if err == nil {
		        ret = fmt.Sprintf("%d", state)
	        } else {
	        	ret = fmt.Sprintf("%v", err)
	        }
	        break;	

        case "input_get":
	        var port int
	        fmt.Sscanf(args[0], "%d", &port)
	        state, err := md.mio.Get_input_port_state(client_id, port)
	        if err == nil {
                ret = fmt.Sprintf("%d", state)
	        } else {
                ret = fmt.Sprintf("%v", err)
	        }
	        break;	

        case "wdt_reset":
	        println("wdt_reset")
	        md.mio.Wdt_reset()
	        break;	

        case "wdt_off":
	        println("wdt_off")
	        err := md.mio.Wdt_set_state(client_id, 0)
	        if err == nil {
		        ret = "ok"
	        } else {
	        	ret = fmt.Sprintf("%v", err)
	        }
	        break;	

        case "wdt_on":
	        println("wdt_on")
	        err := md.mio.Wdt_set_state(client_id, 1)
	        if err == nil {
		        ret = "ok"
	        } else {
	        	ret = fmt.Sprintf("%v", err)
	        }
	        break;	
        }
        fd.Write([]byte(ret))
	}
}

func parse_query(query string) (string, []string) {
	var cmd string
	var args []string

    parts := strings.Split(query, " ")
	first := true
	for _, arg := range parts {
	    arg = strings.Trim(arg, " ")
	    if arg == "" {
	    	continue
	    }
    
	    if first {
	    	cmd = arg
	    	first = false
	    	continue
	    }
	    
	    args = append(args, arg)
	}
	
	return cmd, args
}

