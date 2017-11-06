package conf

import (
	"fmt"
	"io/ioutil"
    "github.com/BurntSushi/toml"
)

const CONFIG_FILE = "/etc/sr90_automation/usio.conf"

type Module_io_cfg struct {
	Uart_dev string
	Uart_speed string
	Responce_timeout int
	Repeate_count int
	Exec_path string
	Exec_script string
	Control_socket string
}

func Conf_parse() (*Module_io_cfg, error) {
	var err error
	var conf Module_io_cfg
	
	config_text, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		return nil, fmt.Errorf("Can't open config file %s: %v", CONFIG_FILE, err)
	}

	_, err = toml.Decode(string(config_text), &conf)
	if err != nil {
		return nil, fmt.Errorf("Can't decode config file %s: %v", CONFIG_FILE, err)
	}
	
	return &conf, nil
}

