package saz

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

var reFileName *regexp.Regexp

func init() {
	reFileName, _ = regexp.Compile("(\\d+)_(\\w)")
}

type Session struct {
	XMLName xml.Name      `xml:"Session"`
	Timers  SessionTimers `xml:"SessionTimers"`
	Flags   SessionFlags  `xml:"SessionFlags"`
}

type SessionTimers struct {
	XMLName             xml.Name `xml:"SessionTimers"`
	ClientConnected     string   `xml:"ClientConnected,attr"`
	ClientBeginRequest  string   `xml:"ClientBeginRequest,attr"`
	GotRequestHeaders   string   `xml:"GotRequestHeaders,attr"`
	ClientDoneRequest   string   `xml:"ClientDoneRequest,attr"`
	GatewayTime         string   `xml:"GatewayTime,attr"`
	DNSTime             string   `xml:"DNSTime,attr"`
	TCPConnectTime      string   `xml:"TCPConnectTime,attr"`
	HTTPSHandshakeTime  string   `xml:"HTTPSHandshakeTime,attr"`
	ServerConnected     string   `xml:"ServerConnected,attr"`
	FiddlerBeginRequest string   `xml:"FiddlerBeginRequest,attr"`
	ServerGotRequest    string   `xml:"ServerGotRequest,attr"`
	ServerBeginResponse string   `xml:"ServerBeginResponse,attr"`
	GotResponseHeaders  string   `xml:"GotResponseHeaders,attr"`
	ServerDoneResponse  string   `xml:"ServerDoneResponse,attr"`
	ClientBeginResponse string   `xml:"ClientBeginResponse,attr"`
	ClientDoneResponse  string   `xml:"ClientDoneResponse,attr"`
}

type SessionFlags struct {
	XMLName xml.Name      `xml:"SessionFlags"`
	Flags   []SessionFlag `xml:"SessionFlag"`
}

type SessionFlag struct {
	XMLName xml.Name `xml:"SessionFlag"`
	Name    string   `xml:"N,attr"`
	Value   string   `xml:"V,attr"`
}

type ParseResult struct {
	Requests []*RequestParseResult
}

func ParseFile(filepath string, urlMatchList []string) (*ParseResult, error) {
	//flag.Parse()
	//r, err := zip.OpenReader(flag.Arg(0))
	r, err := zip.OpenReader(filepath)
	if err != nil {
		fmt.Printf("%s %v\n", filepath, err)
		os.Exit(-1)
		return nil, err
	}
	defer r.Close()
	var request *http.Request
	var response *http.Response
	var session Session

	var res = ParseResult{}

	for _, f := range r.File {
		match, num, t := parseFileName(f.Name)
		if match == false {
			continue
		}

		if t == "c" {
			read, err := f.Open()
			if nil != err {
				//fmt.Printf("%v\n", err)
				//os.Exit(-1)
				return nil, err
			}
			defer read.Close()

			reqReader := bufio.NewReader(read)
			request, _ = http.ReadRequest(reqReader)
		}

		// 检查URL
		var urlOk bool
		for _, urlMatch := range urlMatchList {
			reg := regexp.MustCompile(urlMatch)
			urlOk = reg.MatchString(request.URL.String())
			if urlOk {
				break
			}
		}
		//fmt.Println("@@@@@@@", urlOk, request.URL.String())
		if !urlOk {
			continue
		}

		if t == "m" {
			read, err := f.Open()
			if nil != err {
				//fmt.Printf("%v\n", err)
				//os.Exit(-1)
				return nil, err
			}
			defer read.Close()

			bytes, _ := ioutil.ReadAll(read)
			xml.Unmarshal(bytes, &session)
		}

		if t == "s" {
			read, err := f.Open()
			if nil != err {
				fmt.Printf("%v\n", err)
				//os.Exit(-1)
				return nil, err
			}
			defer read.Close()

			respReader := bufio.NewReader(read)
			response, _ = http.ReadResponse(respReader, request)

			//printResult(num, request, response, session)

			result, err := parseRequest(num, request, response, session)
			//if err != nil{
			//}
			//fmt.Println(result.String(), err)

			if err != nil {
				return nil, err
			}
			res.Requests = append(res.Requests, result)
		}
	}

	return &res, err
}

func parseFileName(name string) (bool, string, string) {
	match := reFileName.FindAllStringSubmatch(name, -1)
	if len(match) == 0 {
		return false, "", ""
	}
	return true, match[0][1], match[0][2]
}

//func printResult(num string, request *http.Request, response *http.Response, session Session) {
//	clientBeginRequest, err := time.Parse(time.RFC3339, session.Timers.ClientBeginRequest)
//	if nil != err {
//		fmt.Println("Error while parsing clientConnected:", err)
//		os.Exit(-1)
//	}
//	clientDoneResponse, err := time.Parse(time.RFC3339, session.Timers.ClientDoneResponse)
//	if nil != err {
//		fmt.Println("Error while parsing clientDoneResponse:", err)
//		os.Exit(-1)
//	}
//	var process string
//	for _, flag := range session.Flags.Flags {
//		if flag.Name == "x-processinfo" {
//			process = flag.Value
//			break
//		}
//	}
//	fmt.Printf("%v\t%v\t%v\t%v\t%v\t%v\t%v\n", num, request.Method, response.StatusCode, request.URL.String(),
//		clientBeginRequest.Format("15:04:05.000"), clientDoneResponse.Format("15:04:05.000"), process)
//
//	defer response.Body.Close()
//	bytes, _ := ioutil.ReadAll(response.Body)
//
//	fmt.Println("@@@", string(bytes))
//}

func parseRequest(num string, request *http.Request, response *http.Response, session Session) (*RequestParseResult, error) {
	clientBeginRequest, err := time.Parse(time.RFC3339, session.Timers.ClientBeginRequest)
	if nil != err {
		fmt.Println("Error while parsing clientConnected:", err)
		return nil, err
	}
	clientDoneResponse, err := time.Parse(time.RFC3339, session.Timers.ClientDoneResponse)
	if nil != err {
		fmt.Println("Error while parsing clientDoneResponse:", err)
		return nil, err
	}
	var process string
	for _, flag := range session.Flags.Flags {
		if flag.Name == "x-processinfo" {
			process = flag.Value
			break
		}
	}
	//fmt.Printf("%v\t%v\t%v\t%v\t%v\t%v\t%v\n", num, request.Method, response.StatusCode, request.URL.String(),
	//	clientBeginRequest.Format("15:04:05.000"), clientDoneResponse.Format("15:04:05.000"), process)

	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	res := &RequestParseResult{
		No:            num,
		RequestMethod: request.Method,
		StatusCode:    response.StatusCode,
		RequestURL:    request.URL.String(),
		RequestBegin:  clientBeginRequest.Format("15:04:05.000"),
		RequestDone:   clientDoneResponse.Format("15:04:05.000"),
		Process:       process,
		ResponseBody:  bytes,
	}
	return res, nil
}

type RequestParseResult struct {
	No            string `json:"no"`
	RequestMethod string `json:"request_method"`
	StatusCode    int    `json:"status_code"`
	RequestURL    string `json:"request_url"`
	RequestBegin  string `json:"request_begin"`
	RequestDone   string `json:"request_done"`
	Process       string `json:"process"`
	ResponseBody  []byte `json:"response_body"`
}

func (r *RequestParseResult) ResponseBodyString() string {
	return string(r.ResponseBody)
}

func (r *RequestParseResult) Summary() string {

	m := map[string]interface{}{
		"no":                 r.No,
		"request_method":     r.RequestMethod,
		"status_code":        r.StatusCode,
		"request_url":        r.RequestURL,
		"request_begin":      r.RequestBegin,
		"request_done":       r.RequestDone,
		"process":            r.Process,
		"response_body_size": len(r.ResponseBody),
	}
	bytes, _ := json.Marshal(&m)
	return string(bytes)
}
