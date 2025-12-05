package main

import (
    "sync"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
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

type DomainInfo struct {
	passthrough bool
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

// getProxyType extracts proxy_type from content (unchanged in new format)
func getProxyType(content map[string]interface{}) string {
    if ptype, ok := content["proxy_type"]; ok && ptype != nil {
        return ptype.(string)
    }
    return ""
}

// getSubdomain extracts subdomain from content
func getSubdomain(content map[string]interface{}) string {
    if subdomain, ok := content["subdomain"]; ok && subdomain != nil {
        return subdomain.(string)
    }
    return ""
}

// getCustomDomains extracts custom_domains from content
func getCustomDomains(content map[string]interface{}) []string {
    if domains, ok := content["custom_domains"]; ok && domains != nil {
        // Handle both []interface{} and []string
        switch d := domains.(type) {
        case []interface{}:
            result := make([]string, len(d))
            for i, v := range d {
                result[i] = v.(string)
            }
            return result
        case []string:
            return d
        }
    }
    return nil
}

type APIServer struct {
	logger *log.Logger
    proxy  *ProxyServer
    domain string
    mutex sync.RWMutex
}

func (s APIServer) handler(w http.ResponseWriter, r *http.Request) {

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

                o.Reject = false
                o.Unchange = true

                proxyType := getProxyType(r.Content)
                if proxyType == "http" || proxyType == "https" {
                    subdomain := getSubdomain(r.Content)
                    if subdomain != "" {
                        var full_domain = subdomain + "." + s.domain
                        s.proxy.addFrontend(full_domain, proxyType == "https")
                    }

                    customDomains := getCustomDomains(r.Content)
                    if customDomains != nil {
                        for _, domain := range customDomains {
                            s.proxy.addFrontend(domain, proxyType == "https")
                        }
                    }

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
                fmt.Fprintf(w, "Not allowed.")
        }
}

func createAPIServer(logger *log.Logger, proxy *ProxyServer, port int, domain string) *APIServer {

    api := &APIServer{
        logger: logger,
        proxy: proxy,
        domain: domain,
    }

	http.HandleFunc("/", api.handler)

     go func () {
	log.Println(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
     }()

    return api

}

