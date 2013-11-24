package main

import (
    "flag"
    "github.com/miekg/dns"
    "io"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
)
var zone string
var proxyIp net.IP


func proxyDnsMsg(w dns.ResponseWriter, m *dns.Msg) *dns.Msg {
    if len(m.Question) == 0 {
        return nil
    }
    q := m.Question[0]
    if q.Name != zone {
        return nil
    }
    if q.Qtype != dns.TypeA {
        return nil
    }

    log.Printf("Proxying request for %s IN A from %s", q.Name, w.RemoteAddr())
    resp := new(dns.Msg)
    resp.SetReply(m)

    rr := new(dns.A)
    rr.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA,
        Class: dns.ClassINET, Ttl: 0}
    rr.A = proxyIp.To4()
    m.Answer = append(m.Answer, rr)
    return m
}

func dnsHandler(w dns.ResponseWriter, m *dns.Msg) {
    if msg := proxyDnsMsg(w, m); msg != nil {
        w.WriteMsg(msg)
        return
    }

    c := new(dns.Client)
    c.Net = "udp"
    r, _, err := c.Exchange(m, "8.8.8.8:53")
    if err != nil {
        log.Print(err)
        return
    }
    log.Printf("Request passed from %s passed through", w.RemoteAddr())
    w.WriteMsg(r)
}

func tcpProxy(local net.Conn, remoteAddr string) {
    remote, err := net.Dial("tcp", remoteAddr)
    if err != nil {
        log.Print("Failed to connect to %s: %s", remoteAddr, err)
        return
    }
    go io.Copy(local, remote)
    go io.Copy(remote, local)
}

func listenAndServe() {
    go func() {
        err := dns.ListenAndServe(":53", "udp", dns.HandlerFunc(dnsHandler))
        if err != nil {
            log.Fatal(err)
        }
    }()
    go func() {
        err := dns.ListenAndServe(":53", "tcp", dns.HandlerFunc(dnsHandler))
        if err != nil {
            log.Fatal(err)
        }
    }()
    go func() {
        listener, err := net.Listen("tcp", ":80")
        if err != nil {
            log.Fatal(err)
        }
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Print(err)
            }
            go tcpProxy(conn, "movies.netflix.com:80")
        }
    }()
    go func() {
        listener, err := net.Listen("tcp", ":443")
        if err != nil {
            log.Fatal(err)
        }
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Print(err)
            }
            go tcpProxy(conn, "cbp-us.nccp.netflix.com:443")
        }
    }()
}

func main() {
    flZone := flag.String("zone", "", "the zone to proxy")
    flIp := flag.String("ip", "", "ip address to answer with")
    flag.Parse()

    if len(*flZone) == 0 {
        log.Fatal("Argument zone must be given")
    }
    if len(*flIp) == 0 {
        log.Fatal("Argument ip must be given")
    }

    zone = *flZone
    if proxyIp = net.ParseIP(*flIp); proxyIp == nil {
        log.Fatalf("Invalid IP address: %s", *flIp)
    }
    log.Printf("Proxying requests for zone: %s -> %s", zone, proxyIp)

    listenAndServe()

    sig := make(chan os.Signal)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    for {
        select {
        case s := <-sig:
            log.Fatalf("Signal (%d) received, stopping\n", s)
        }
    }
}
