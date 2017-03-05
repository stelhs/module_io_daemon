package huawei_e303

import (
	"fmt"
	"net/http"
	"encoding/xml"
	"strings"
	"io/ioutil"
	"conf"
)

type Modem struct {
	ip_addr string
}

// Huawei modem error response by POST/GET query
type huawei_e303_err_resp_xml struct {
	XMLName xml.Name `xml:"error"`
	Code int `xml:"code"`
	Message string `xml:"message"`
}

// Huawei modem OK response by POST query
type huawei_e303_ok_resp_xml struct {
	XMLName xml.Name `xml:"response"`
	Ok string 	     `xml:",chardata"`
}

type Modem_sms_message struct {
	Smstat     int    `xml:"Smstat"`
	Index      int    `xml:"Index"`
	Phone      string `xml:"Phone"`
	Content    string `xml:"Content"`
	Date       string `xml:"Date"`
	Sca        string `xml:"Sca"`
	SaveType   int    `xml:"SaveType"`
	Priority   int    `xml:"Priority"`
	SmsType    int    `xml:"SmsType"`
}

type Modem_global_status struct {
	XMLName               xml.Name `xml:"response"`
	ConnectionStatus      int      `xml:"ConnectionStatus"`
	SignalStrength        int      `xml:"SignalStrength"`
	SignalIcon            int	   `xml:"SignalIcon"`
	CurrentNetworkType    int      `xml:"CurrentNetworkType"`
	CurrentServiceDomain  int      `xml:"CurrentServiceDomain"`
	RoamingStatus         int      `xml:"RoamingStatus"`
	WanIPAddress          string   `xml:"WanIPAddress"`
	PrimaryDns            string   `xml:"PrimaryDns"`
	SecondaryDns          string   `xml:"SecondaryDns"`
}

type Modem_trafic_statistics struct {
	XMLName               xml.Name `xml:"response"`
	CurrentConnectTime    uint64   `xml:"CurrentConnectTime"`
	CurrentUpload         uint64   `xml:"CurrentUpload"`
	CurrentDownload       uint64   `xml:"CurrentDownload"`
	CurrentDownloadRate   uint64   `xml:"CurrentDownloadRate"`
	CurrentUploadRate     uint64   `xml:"CurrentUploadRate"`
	TotalUpload           uint64   `xml:"TotalUpload"`
	TotalDownload         uint64   `xml:"TotalDownload"`
	TotalConnectTime      uint64   `xml:"TotalConnectTime"`
}


type Modem_sent_sms_stat struct {
	XMLName     xml.Name `xml:"response"`
	Phone       string   `xml:"Phone"`
	SucPhone    string   `xml:"SucPhone"`
	FailPhone   string   `xml:"FailPhone"`
	TotalCount  int      `xml:"TotalCount"`
	CurIndex    int      `xml:"CurIndex"`
}


func New(mcfg *conf.Modem_cfg) *Modem {
	m := new(Modem)
	m.ip_addr = mcfg.Ip_addr
	return m
}


func (m *Modem) send_xml_post_query(query interface{}, url string) (string, error) {
	var err error
	
	// create XML text	
	query_xml, err := xml.MarshalIndent(query, "  ", "    ")
	if err != nil {
		return "", fmt.Errorf("xml create err: %v\n", err)
	}
	
	// make POST query	
	resp, err := http.Post("http://" + m.ip_addr + url, 
						   "application/x-www-form-urlencoded", 
						   strings.NewReader(string(query_xml)))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("modem error response: %d\n", resp.Status)
	}

	// read POST response 	
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("modem read response error: %v\n", err)
	}
	defer resp.Body.Close()
	
	// Attempt to Parse Error response 	
	resp_err := new(huawei_e303_err_resp_xml)
	err = xml.Unmarshal(resp_body, resp_err)
	if err == nil {
		return "", fmt.Errorf("modem return error code: %d", resp_err.Code)
	}

	return string(resp_body), nil
}


func (m *Modem) send_get_query(url string) (string, error) {
	var err error
	
	// make GET query	
	resp, err := http.Get("http://" + m.ip_addr + url)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("modem error response: %d\n", resp.Status)
	}

	// read GET response 	
	resp_body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("modem read response error: %v\n", err)
	}

	// Attempt to Parse Error response
	resp_err := new(huawei_e303_err_resp_xml)
	err = xml.Unmarshal(resp_body, resp_err)
	if err == nil {
		return "", fmt.Errorf("modem return error code: %d", resp_err.Code) 
	}
	
	return string(resp_body), nil
}

func (m *Modem) Send_sms(phone_num string, text string) error {
	type query struct {
		XMLName   xml.Name `xml:"request"`
		Index     int      `xml:"Index"`
		Phone     string   `xml:"Phones>Phone"`
		Content   string   `xml:"Content"`
		Lenght    int      `xml:"Length"`
		Reserved  int      `xml:"Reserved"`
		Date      string   `xml:"Date"`
	}
	
	q := &query{Index: -1, 
				Phone: phone_num,
				Content: text,
				Lenght: len(text),
				Date: "111",
				}
	
	resp_body, err := m.send_xml_post_query(q, "/api/sms/send-sms")
	if err != nil {
		return err
	}
	
	// Attempt to Parse OK response 	
	resp_ok := new(huawei_e303_ok_resp_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_ok)
	if err != nil {
		return fmt.Errorf("Send_sms responce can`t parse: %v\n", err)
	}

	return nil
}


func (m *Modem) Remove_sms(sms_index int) error {
	type query struct {
		XMLName   xml.Name `xml:"request"`
		Index     int      `xml:"Index"`
	}
	
	q := &query{Index: sms_index,}
	
	resp_body, err := m.send_xml_post_query(q, "/api/sms/delete-sms")
	if err != nil {
		return err
	}
	
	// Attempt to Parse OK response 	
	resp_ok := new(huawei_e303_ok_resp_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_ok)
	if err != nil {
		return fmt.Errorf("Remove_sms responce can`t parse: %v\n", err)
	}

	return nil
}


func (m *Modem) Check_for_new_sms() ([]Modem_sms_message, error) {
	type query struct {
		XMLName          xml.Name `xml:"request"`
		PageIndex        int      `xml:"PageIndex"`
		ReadCount        int      `xml:"ReadCount"`
		BoxType          int      `xml:"BoxType"`
		SortType         int      `xml:"SortType"`
		Ascending        int      `xml:"Ascending"`
		UnreadPreferred  int      `xml:"UnreadPreferred"`
	}
	
	q := &query{PageIndex: 1, 
				ReadCount: 20,
				BoxType: 1,
				SortType: 0,
				Ascending: 0,
				UnreadPreferred: 0,
				}
	
	resp_body, err := m.send_xml_post_query(q, "/api/sms/sms-list")
	if err != nil {
		return nil, err
	}
	
	type huawei_e303_sms_list_xml struct {
		XMLName xml.Name `xml:"response"`
		Count   string   `xml:"Count"`
		Messages []Modem_sms_message `xml:"Messages>Message"`
	}
	
	resp_sms_list := new(huawei_e303_sms_list_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_sms_list)
	if err != nil {
		return nil, fmt.Errorf("List incomming sms can`t parse: %v\n", err)
	}

	return resp_sms_list.Messages, nil
}


func (m *Modem) Send_ussd(text string) error {
	type query struct {
		XMLName   xml.Name `xml:"request"`
		Content   string   `xml:"content"`
		Code_type string   `xml:"codeType"`
	}
	
	q := &query{Content: text,
				Code_type: "CodeType",
				}
	
	resp_body, err := m.send_xml_post_query(q, "/api/ussd/send")
	if err != nil {
		return err
	}
	
	// Attempt to Parse OK response 	
	resp_ok := new(huawei_e303_ok_resp_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_ok)
	if err != nil {
		return fmt.Errorf("Remove_sms responce can`t parse: %v\n", err)
	}

	return nil
}


func (m *Modem) Check_for_new_ussd() (string, error) {
	resp_body, err := m.send_get_query("/api/ussd/get")
	if err != nil {
		return "", err
	}
	
	type responce_xml struct {
		XMLName xml.Name `xml:"response"`
		Content string   `xml:"content"`
	}
	
	resp_ok := new(responce_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_ok)
	if err != nil {
		return "", fmt.Errorf("Check_for_new_ussd responce can`t parse: %v\n", err)
	}
	
	return resp_ok.Content, nil
}


func (m *Modem) Get_global_status() (*Modem_global_status, error) {
	resp_body, err := m.send_get_query("/api/monitoring/status")
	if err != nil {
		return nil, err
	}

	fmt.Printf("resp: %v", resp_body)

	resp_status := new(Modem_global_status)
	err = xml.Unmarshal([]byte(resp_body), resp_status)
	if err != nil {
		return nil, fmt.Errorf("Get_global_status responce can`t parse: %v\n", err)
	}
	
	return resp_status, nil
}


func (m *Modem) Get_traffic_statistics() (*Modem_trafic_statistics, error) {
	resp_body, err := m.send_get_query("/api/monitoring/traffic-statistics")
	if err != nil {
		return nil, err
	}

	fmt.Printf("resp: %v", resp_body)

	resp_statistics := new(Modem_trafic_statistics)
	err = xml.Unmarshal([]byte(resp_body), resp_statistics)
	if err != nil {
		return nil, fmt.Errorf("Get_traffic_statistics responce can`t parse: %v\n", err)
	}
	
	return resp_statistics, nil
}


func (m *Modem) Reset_traffic_statistics() error {
	type query struct {
		XMLName        xml.Name `xml:"request"`
		ClearTraffic   int      `xml:"ClearTraffic"`
	}
	
	q := &query{ClearTraffic: 1}
	
	resp_body, err := m.send_xml_post_query(q, "/api/monitoring/clear-traffic")
	if err != nil {
		return err
	}
	
	// Attempt to Parse OK response 	
	resp_ok := new(huawei_e303_ok_resp_xml)
	err = xml.Unmarshal([]byte(resp_body), resp_ok)
	if err != nil {
		return fmt.Errorf("Reset_traffic_statistics responce can`t parse: %v\n", err)
	}

	return nil
}


func (m *Modem) Check_sended_sms_status() (*Modem_sent_sms_stat, error) {
	resp_body, err := m.send_get_query("/api/sms/send-status")
	if err != nil {
		return nil, err
	}

	fmt.Printf("resp: %v", resp_body)

	resp_stat := new(Modem_sent_sms_stat)
	err = xml.Unmarshal([]byte(resp_body), resp_stat)
	if err != nil {
		return nil, fmt.Errorf("Check_sended_sms_status responce can`t parse: %v\n", err)
	}
	
	return resp_stat, nil
}
