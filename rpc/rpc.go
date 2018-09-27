package rpc

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	rpcID  = 99
	rpcURI = "/cgi-bin/luci/rpc/"
)

var (
	ErrHttpUnauthorized = errors.New("http: Unauthorized")
	ErrHttpForbidden    = errors.New("http: Forbidden")

	defaultTimeout = 15
)

type Auth struct {
	Username string
	Password string
	Timeout  int
}

type RpcServer struct {
	Hostname string
	Port     int
	SSL      bool
}

type Client struct {
	rpcServer *RpcServer
	auth      *Auth

	token string
	id    int

	httpClient *http.Client
}

type Payload struct {
	ID     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  *string     `json:"error"`
}

func New(rpcServer *RpcServer, auth *Auth) (*Client, error) {

	if auth.Timeout == 0 {
		auth.Timeout = defaultTimeout
	}
	client := &Client{
		rpcServer: rpcServer,
		auth:      auth,
		id:        rpcID,
	}

	client.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Dial: (&net.Dialer{
				Timeout:   time.Duration(auth.Timeout) * time.Second,
				KeepAlive: time.Duration(auth.Timeout) * time.Second,
			}).Dial,
		},
	}

	err := client.login()
	if err != nil {
		return nil, err
	}

	return client, err
}

func (c *Client) httpError(code int) error {
	if code == 401 {
		return ErrHttpUnauthorized
	} else if code == 403 {
		return ErrHttpForbidden
	} else {
		return errors.New(fmt.Sprintf("HTTP status code: %d", code))
	}
}

func (c *Client) call(url string, postBody []byte) ([]byte, error) {
	var respBody []byte

	body := bytes.NewReader(postBody)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return respBody, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode > 226 {
		return respBody, c.httpError(resp.StatusCode)
	}

	return respBody, err
}

func (c *Client) url(uri string) string {
	proto := "http://"
	port := ""
	if c.rpcServer.SSL == true {
		proto = "https://"
	}

	if c.rpcServer.Port != 0 && c.rpcServer.Port != 443 {
		port = fmt.Sprintf(":%d", c.rpcServer.Port)
	}

	url := proto + c.rpcServer.Hostname + port + rpcURI + uri
	if c.token != "" {
		url = url + "?auth=" + c.token
	}

	return url
}

func (c *Client) rpc(library, method string, params []string) (string, error) {
	var response Response
	var result string

	if library != "auth" && c.token == "" {
		return "", errors.New("RPC client is not authenticated")
	}

	payload := Payload{
		ID:     c.id,
		Method: method,
		Params: params,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := c.url(library)
	respBody, err := c.call(url, data)
	// Session timeout or OpenWrt reboot
	if err == ErrHttpUnauthorized || err == ErrHttpForbidden {
		log.Warn("Login again")
		url := c.url(library)
		c.login()
		respBody, err = c.call(url, data)
	}

	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", errors.New(fmt.Sprintf("%s", *response.Error))
	}

	if _, ok := response.Result.(string); ok == true {
		result = fmt.Sprintf("%v", response.Result)
		return result, nil
	}
	jsonBytes, err := json.Marshal(response.Result)
	if err == nil {
		result = string(jsonBytes)
	}

	return result, err
}

func (c *Client) login() error {
	var token string
	token, err := c.rpc("auth", "login", []string{c.auth.Username, c.auth.Password})
	if err != nil {
		log.Error(err)
		return err
	}
	c.token = token
	return err
}

func (c *Client) SysRPC(method string, params []string) (string, error) {
	return c.rpc("sys", method, params)
}

func (c *Client) UciRPC(method string, params []string) (string, error) {
	return c.rpc("uci", method, params)
}

func (c *Client) FsRPC(method string, params []string) (string, error) {
	return c.rpc("fs", method, params)
}
