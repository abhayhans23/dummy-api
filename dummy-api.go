package main

import (
	"log"
	"io"
	"net/url"
	"net/http"
        "fmt"
        "flag"
        "encoding/json"
        "strconv"
        "time"
        "strings"
        "math/rand"
)

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
    var default_host = "127.0.0.1"
    var default_port = 1337
    var default_readtimeout = 10
    var default_writetimeout = 10
    var default_maxheaderbytes = 1 << 20 // 1 MB
    
    var host string
    var port int
    var readtimeout int
    var writetimeout int
    var maxheaderbytes int

    flag.StringVar(&host, "host", default_host, "Listen host")
    flag.IntVar(&port, "port", default_port, "Listen port")
    flag.IntVar(&readtimeout, "readtimeout", default_readtimeout,
        "Read timeout in seconds")
    flag.IntVar(&writetimeout, "writetimeout", default_writetimeout,
         "Write timeout in seconds")
    flag.IntVar(&maxheaderbytes, "maxheaderbytes", default_maxheaderbytes,
         "Max header bytes.")
    
    flag.Parse()
    
    if port < 1 || port > 65535 {
        port = default_port
        fmt.Println("Invalid port number, using default.")
    }
    
    if readtimeout < 1 || readtimeout > 300 {
        readtimeout = default_readtimeout
        fmt.Println("Invalid read timeout, using default.")
    }
    
    if writetimeout < 1 || writetimeout > 300 {
        writetimeout = default_writetimeout
        fmt.Println("Invalid write timeout, using default.")
    }

    if maxheaderbytes < 1 || maxheaderbytes > 2 << 40 {
        maxheaderbytes = default_maxheaderbytes
        fmt.Println("Invalid max header bytes, using default.")
    }
    
    server := http.Server{
        Addr: host + ":" + strconv.Itoa(port),
        Handler: &myHandler{},
        ReadTimeout: time.Duration(readtimeout) * time.Second,
        WriteTimeout: time.Duration(writetimeout) * time.Second,
        MaxHeaderBytes: maxheaderbytes,
    }
    
    mux = make(map[string]func(http.ResponseWriter, *http.Request))
    
    log.Fatal(server.ListenAndServe())
}

func get_property(params url.Values, headers http.Header, key string) (bool, string) {
    if values, ok := params[key]; ok {
        for _, value := range values {
            return true, value
        }
    }

    for header, values := range headers {
        if http.CanonicalHeaderKey(header) == http.CanonicalHeaderKey(key) {
            for _, value := range values {
                return true, value
            }
        }
    }
    return false, ""
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generate_string(n int, s int64) string {
    rand.Seed(s)
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

func help(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-type", "text/plain")
    help := `
Dummy API
=========

The following request headers and query parameters will make an impact on the response.

Delay
-----
header-delay = {int}         Delay to first header byte in ms.
body-delay = {int}           Delay to first body byte in ms.

Cache-control
-------------
max-age = {int}              Set the cache-control max-age value.
s-maxage = {int}             Set the cache-control s-maxage value.
must-revalidate              Set cache-control must-revalidate.
public                       Set cache-control public.
private                      Set cache-control private.
no-store                     Set cache-control no-store.
no-cache                     Set cache-control no-cache.
no-transform                 Set cache-control no-transform.

Misc
----
content-length               Set the content-length header, otherwise chunked
                             transfer encoding is used.
random-content = {int}       Add random string to the response of given length.
predictable-content = {int}  Add predictable string to the response of given
                             length.
connection=close             Add connection=close to the response headers.
response-status = {int}      Set the response status.`

    io.WriteString(w, help)
}

func process(w http.ResponseWriter, r *http.Request) {
    // Defaults
    var header_delay = 0
    var body_delay = 0
    var response_status = 200
    var content_length = false
    //var random_content_length = 0
    //var random_content = ""
    //var predictable_content_length = 0
    //var predictable_content = ""
    var connection = ""
    cache_control := []string{}

    u, err := url.Parse(r.RequestURI)
    if err != nil {
        panic(err)
    }

    resp := map[string]string{}
    resp["host"] = r.Host
    resp["path"] = r.RequestURI
    resp["method"] = r.Method
    resp["protocol"] = r.Proto
    resp["client"] = r.RemoteAddr
    
    params, _ := url.ParseQuery(u.RawQuery)
    var headers = r.Header

    arg := "header-delay"
    set, value := get_property(params, headers, arg)
    if set  {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i > 0 && i < 300 {
                header_delay = i
                resp[arg] = value
            }
        }
    }

    arg = "body-delay"
    set, value = get_property(params, headers, arg)
    if set  {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i > 0 && i < 300 {
                body_delay = i
                resp[arg] = value
            }
        }
    }

    arg = "max-age"
    set, value = get_property(params, headers, arg)
    if set  {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i >= 0 {
                cache_control = append(cache_control, arg + "=" + value)
                resp[arg] = value
            }
        }
    }

    arg = "s-maxage"
    set, value = get_property(params, headers, arg)
    if set  {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i >= 0 {
                cache_control = append(cache_control, arg + "=" + value)
                resp[arg] = value
            }
        }
    }

    arg = "must-revalidate"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "public"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "private"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "no-store"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "no-cache"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "no-transform"
    set, value = get_property(params, headers, arg)
    if set  {
        cache_control = append(cache_control, arg)
        resp[arg] = "true"
    }

    arg = "content-length"
    set, value = get_property(params, headers, arg)
    if set {
        content_length = true
        resp[arg] = "true"
    }

    arg = "connection"
    set, value = get_property(params, headers, arg)
    if set {
        if value == "close" {
            connection = value
            resp[arg] = value
        }
    }

    arg = "response-status"
    set, value = get_property(params, headers, arg)
    if set {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i >= 100 && i < 600 {
                response_status = i
                resp[arg] = value
            }
        }
    }

    arg = "random-content"
    set, value = get_property(params, headers, arg)
    if set {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i > 0 {
                var seed = time.Now().UTC().UnixNano()
                resp["random-content"] = generate_string(i, seed)
            }
        }
    }

    arg = "predictable-content"
    set, value = get_property(params, headers, arg)
    if set {
        i, err := strconv.Atoi(value)
        if err == nil {
            if i > 0 {
                var seed = int64(len(r.Method + r.Host + r.RequestURI))
                resp["predictable-content"] = generate_string(i, seed)
            }
        }
    }
    
    content, err := json.Marshal(resp)
    if err != nil {
        fmt.Printf("Error: %s", err)
    }

    if header_delay > 0 {
        // Flush is not necessary
        time.Sleep(time.Duration(header_delay) * time.Millisecond)
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Server", "Dummy API")

    if content_length {
        w.Header().Set("Content-Length", strconv.Itoa(len(string(content))))
    }

    if connection != "" {
        w.Header().Set("Connection", connection)
    }

    if len(cache_control) > 0 {
        w.Header().Set("Cache-control", strings.Join(cache_control, ", "))
    }

    w.WriteHeader(response_status)

    if body_delay > 0 {
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        } else {
            log.Println("Unable to flush")
        }
        time.Sleep(time.Duration(body_delay) * time.Millisecond)
    }

    io.WriteString(w, string(content))
}

type myHandler struct{}
func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

    u, err := url.Parse(r.RequestURI)
    if err != nil {
        panic(err)
    }
    
    params, _ := url.ParseQuery(u.RawQuery)
    headers := r.Header

    arg := "help"
    set, _ := get_property(params, headers, arg)
    if set {
        help(w, r)
    } else {
        process(w, r)
    }
}
