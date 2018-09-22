package rpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

var (
	mux    *http.ServeMux
	ts     *httptest.Server
	client *Client

	authToken = "2149750d18d5da63b123b3e68309956e"
)

func setup() func() {
	mux = http.NewServeMux()
	ts = httptest.NewServer(mux)

	u, _ := url.Parse(ts.URL)
	auth := &Auth{
		Username: "root",
		Password: "openwrt",
	}

	port, _ := strconv.Atoi(u.Port())
	server := &RpcServer{
		Hostname: u.Hostname(),
		Port:     port,
	}

	mux.HandleFunc(rpcURI+"auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":"%s","error":null}`, rpcID, authToken)
	})

	client, _ = New(server, auth)
	return func() {
		ts.Close()
	}
}

func TestHttpsAuth(t *testing.T) {
	mux = http.NewServeMux()
	ts = httptest.NewTLSServer(mux)
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	auth := &Auth{
		Username: "root",
		Password: "openwrt",
	}

	port, _ := strconv.Atoi(u.Port())
	server := &RpcServer{
		Hostname: u.Hostname(),
		Port:     port,
		SSL:      true,
	}

	mux.HandleFunc(rpcURI+"auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":"%s","error":null}`, rpcID, authToken)
	})

	client, err = New(server, auth)
	if err != nil {
		t.Error(err)
	}

	if client.token != authToken {
		t.Error("Failed to autheciate the json RPC server over HTTPS")
	}
}

func TestResponseString(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `OpenWrt`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":"%s","error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("hostname", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponseBool(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `true`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":%s,"error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("testBool", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponseNumber(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `48650`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":%s,"error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("uptime", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponseStrinArray(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `["lo","eth0","eth1","br-lan"]`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":%s,"error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("net.devices", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponseObjectArray(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `[{"PID":"1","PPID":0},{"PID":"100","PPID":1}]`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":%s,"error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("net.devices", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponseObject(t *testing.T) {
	teardown := setup()
	defer teardown()

	var result string = `{"b":true,"m":[1,2,3],"n":99,"s":"String"}`

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":%s,"error":null}`, rpcID, result)
	})

	r, err := client.SysRPC("getObject", nil)
	if err != nil {
		t.Error(err)
	}
	if r != result {
		t.Error("Failed to get response")
	}
}

func TestResponse403(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
	})

	_, err := client.SysRPC("forbidden", nil)
	if err != ErrHttpForbidden {
		t.Error(err)
	}
}

func TestRPCError(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(rpcURI+"sys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":%d,"result":null,"error":"FakeError"}`, rpcID)
	})

	r, err := client.SysRPC("test", nil)
	if err.Error() != "FakeError" {
		t.Error("RPC should throw error message")
	}
	if r != "" {
		t.Error("RPC result is not empty")
	}
}
