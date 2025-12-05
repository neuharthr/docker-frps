package main

import (
    "os"
    "sync"
    "encoding/json"
    "fmt"
    "net/http"
    "bufio"
    "regexp"
    "strconv"
)

type Request struct {
	Version string      `json:"version"`
	Op      string      `json:"op"`
	Content map[string]interface{} `json:"content"`
}

type Response struct {
	Reject       bool        `json:"reject"`
	RejectReason string      `json:"reject_reason"`
	Unchange     bool        `json:"unchange"`
	Content      map[string]interface{} `json:"content"`
}

var mutex sync.RWMutex
var ports = make(map[string]int)

func getEnv(name string, def string) string {
    val := os.Getenv(name)

    if val == "" {
        return def
    }

    return val

}

func getEnvInt(name string, def int) int {
    val := os.Getenv(name)

    if val == "" {
        return def
    }

    ival, _ := strconv.ParseInt(val, 10, 32)

    return int(ival)
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

// getMetas extracts metas from the request content
// In FRP v0.65+, user-level metas are in content.user.metas
// and proxy-level metas are in content.metas
func getMetas(content map[string]interface{}) map[string]interface{} {
    // First try proxy-level metas (new format)
    if metas, ok := content["metas"]; ok && metas != nil {
        if metasMap, ok := metas.(map[string]interface{}); ok {
            return metasMap
        }
    }
    // Try user-level metas (new format)
    if user, ok := content["user"]; ok && user != nil {
        if userMap, ok := user.(map[string]interface{}); ok {
            if metas, ok := userMap["metas"]; ok && metas != nil {
                if metasMap, ok := metas.(map[string]interface{}); ok {
                    return metasMap
                }
            }
        }
    }
    return nil
}

// getProxyName extracts proxy_name from content (unchanged in new format)
func getProxyName(content map[string]interface{}) string {
    if name, ok := content["proxy_name"]; ok && name != nil {
        return name.(string)
    }
    return ""
}

// getProxyType extracts proxy_type from content (unchanged in new format)
func getProxyType(content map[string]interface{}) string {
    if ptype, ok := content["proxy_type"]; ok && ptype != nil {
        return ptype.(string)
    }
    return ""
}

// getRemotePort extracts remote_port from content (unchanged in new format)
func getRemotePort(content map[string]interface{}) int {
    if port, ok := content["remote_port"]; ok && port != nil {
        return int(port.(float64))
    }
    return 0
}

var portMin int = getEnvInt("PLUGIN_PORT_MIN", 30000)
var portMax int = getEnvInt("PLUGIN_PORT_MAX", 30900)

func savePortMapping() {

    f, err := os.Create("ports.map")
    check(err)

    for k, v := range ports {
        _, err := f.WriteString(fmt.Sprintf("%v %v\n", k, v))
        check(err)
    }

    defer f.Close()
}

func handler(w http.ResponseWriter, r *http.Request) {

        switch r.Method {
        case "POST":
                d := json.NewDecoder(r.Body)
                r := &Request{}
                o := &Response{}
                err := d.Decode(r)
                if err != nil {
                        http.Error(w, err.Error(), http.StatusInternalServerError)
                }

                if r.Op != "NewProxy" {
                    w.WriteHeader(http.StatusMethodNotAllowed)
                    fmt.Fprintf(w, "Not allowed.")
                    return
                }
                
                proxyType := getProxyType(r.Content)
                if proxyType == "tcp" || proxyType == "udp" {
                    
                    proxyName := getProxyName(r.Content)
                    var key = fmt.Sprintf("%v:%v", proxyName, proxyType)
                    var port int = getRemotePort(r.Content)

                    // Allocate or retrieve port
                    if port == 0 {

                       mutex.Lock()

                        port, ok := ports[key]

                        if !ok {

                            var allocated = make(map[int]bool)

                            for _, v := range ports {
	                            allocated[v] = true
                            }

                            for i := portMin; i <= portMax; i++ {

                                if !allocated[i] {
                                    port = i
                                    break
                                }

                            }

                            if port == 0 {
                                fmt.Printf("[portmanager - %s] WARNING: Unable to allocate port, all available ports already taken.\n", key)
                                
                                o.Reject = true
                                o.RejectReason = "All available ports already taken"
                            } else {
                                fmt.Printf("[portmanager - %s] New client found, allocating new port: '%d'.\n", key, port)
                                
                                ports[key] = port
                                savePortMapping()

                                o.Reject = false
                                o.Unchange = false
                                o.Content = r.Content
                                o.Content["remote_port"] = port

                            }

                        } else {
                            fmt.Printf("[portmanager - %s] Known client ... using port %d.\n", key, port)
                            
                            o.Reject = false
                            o.Unchange = false
                            o.Content = r.Content
                            o.Content["remote_port"] = port

                        }

                        mutex.Unlock()

                    } else {
                        // Verify that port is not taken

                        mutex.Lock()

                        var found bool = false

                        fmt.Printf("[portmanager - %s] New client ... allocating requested port '%d'.\n", key, port)
                        
                        for k, v := range ports {
                            if v == port {
                                if k == key {
                                    o.Reject = false
                                    o.Unchange = true
                                } else {
                                    o.Reject = true
                                    o.RejectReason = "Port already taken by another proxy"
                                    fmt.Printf("[portmanager - %s] WARNING: Cannot allocate, port already taken by another client!.\n", key, port)
                                }
                                found = true
                            }
                        }

                        if !found {
                            if port >= portMin && port <= portMax {
                                ports[key] = port
                                o.Reject = false
                                o.Unchange = true

                                savePortMapping()

                            } else {
                                o.Reject = true
                                o.RejectReason = "Illegal port number"
                                fmt.Printf("[portmanager - %s] WARNING: Illegal port number requested!.\n", key, port)
                            }
                        }

                        mutex.Unlock()

                    }

                } else {
                    o.Reject = false
                    o.Unchange = true
                }

                js, err := json.Marshal(o)
                if err != nil {
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                    return
                }
                w.Header().Set("Content-Type", "application/json")
                w.Write(js)

        default:
                w.WriteHeader(http.StatusMethodNotAllowed)
                fmt.Fprintf(w, "I can't do that.")
        }
}

func main() {

    mutex.Lock()
    
    f, err := os.Open("ports.map")

    if !os.IsNotExist(err) {

        fmt.Printf("[portmanager]: Reading cached port mapping in 'ports.map'\n")
        var lineParser = regexp.MustCompile(`^(\S+) ([0-9]+)$`)

        s := bufio.NewScanner(f)
        for s.Scan() {
            line := s.Text()

            matches := lineParser.FindSubmatch([]byte(line))

            if len(matches) == 3 {
                port64, _ := strconv.ParseInt(string(matches[2]), 10, 32)
                port := int(port64)

                if port >= portMin && port <= portMax {
                    fmt.Printf("[portmanager]: Found port %d for %s\n", port, string(matches[1]))
                    ports[string(matches[1])] = int(port)
                } else {
                    fmt.Printf("[portmanager]: Found port %d for %s BUT DOES NOT MATCH LIMITS %d <= port <= %d.\n", port, string(matches[1]), portMin, portMax)
                }

            } else {
                fmt.Printf("[portmanager]: Line does ot contain three parts: '%s'.\n", line)
            }

        }

        fmt.Printf("[portmanager]: Done - total cached ports found: %d.\n", len(ports))
        f.Close()
    } else {
        fmt.Printf("[portmanager]: No cache for port mapping 'ports.map' file found\n")
    }
    
    mutex.Unlock()

    http.HandleFunc("/", handler)
    http.ListenAndServe(fmt.Sprintf(":%d", getEnvInt("PLUGIN_PORT", 9001)), nil)
}

