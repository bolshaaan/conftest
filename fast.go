package main

import (
	"io"
	"fmt"
	"strings"
	"encoding/json"
	"sync"
	"io/ioutil"
	"bytes"
	"regexp"
)

// глобальные переменные запрещены
// cgo запрещен

/*
Выходной формат:
Total: X
[idx] Name <Email>
где:
X - сколько нашлось результатов
idx - порядковый номер результата в файле, начинается с 1
Name - имя из записи
Email - адрес ящику, у которого была проведена замена "@" -> " [at] "
*/


// Decimal to integer.
// Returns number, characters consumed, success.
func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return big, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}
type IP []byte
// Bigger than we need, not too big to worry about overflow
const big = 0xFFFFFF
const IPv4len = 4
// Parse IPv4 address (d.d.d.d).
func parseIPv4(s string) IP {
	var p [IPv4len]byte
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return nil
		}
		if i > 0 {
			if s[0] != '.' {
				return nil
			}
			s = s[1:]
		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return nil
		}
		s = s[c:]
		p[i] = byte(n)
	}
	if len(s) != 0 {
		return nil
	}
	return IP{p[0], p[1], p[2], p[3]}
}

type IPMask []byte

func CIDRMask(ones, bits int) IPMask {
	if ones < 0 || ones > bits {
		return nil
	}
	l := bits / 8
	m := make(IPMask, l)
	n := uint(ones)
	for i := 0; i < l; i++ {
		if n >= 8 {
			m[i] = 0xff
			n -= 8
			continue
		}
		m[i] = ^byte(0xff >> n)
		n = 0
	}
	return m
}

func ParseCIDR(s string) (IP, IPMask,  error) {
	i := strings.IndexByte(s, '/')

	addr, mask := s[:i], s[i+1:]
	ip := parseIPv4(addr)

	n, i, _:= dtoi(mask)
	ipMask := CIDRMask(n, 8*IPv4len)

	return ip, ipMask, nil
}


type Network struct {
	ip, mask []byte
}


func Fast(in io.Reader, out io.Writer, networks []string) {
	// сюда писать код

	browserRegex, _ := regexp.Compile(`Chrome/(60.0.3112.90|52.0.2743.116|57.0.2987.133)`)

	nwrks := make([]Network, 0, len(networks) )
	var logs Logs

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for _, nw := range networks {
			ip, ipMask, _ := ParseCIDR(nw)
			nwrks = append(nwrks, Network{  ip: ip, mask: ipMask  })
		}
	}(wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		all, _ := ioutil.ReadAll(in)
		//logStr := strings.Split(string(all), "\n")

		for _, v := range bytes.Split(all, []byte("\n")) {
			if len(v) == 0  {
				continue
			}

			raw := &Log{}
			json.Unmarshal(v, raw)
			logs = append(logs, raw)
		}

	}(wg)

	wg.Wait()

	//comp, _ := regexp.Compile(`hits\"\:(\[(\"\d+\.\d+\.\d+\.\d+\",{0,1})+\])`)
	//comp, _ := regexp.Compile(`hits\"\:\[(\"\d+\.\d+\.\d+\.\d+\",{0,1})+.*\]`)
	//logs := strings.Split(string(all), "\n")

	var total = 0
	var res []string

	//comp, _ := regexp.Compile(`hits\"\:\[(.*)\]`)
	for logPos, log := range logs {
		//if len(log) == 0 {continue}
		//b := comp.FindAllStringSubmatch(log, -1)
		//addrs := strings.Split(b[0][1], ",")

		match := false

		ipCount := 0
		browserCount := 0

		for _, br := range log.Browsers {
			if browserRegex.MatchString(br) {
				browserCount++
			}
		}

		for _, addr := range log.Hits {
			match = false
			//ip := parseIPv4(addr[1:len(addr)-1])
			ip := parseIPv4(addr)

			// find match
			for _, nw := range nwrks {
				match = true
				for i:= 0; i<4; i++ {
					if ip[i] & nw.mask[i] != nw.ip[i] {
						match = false
						break
					}
				}

				//if logPos == 3 && match {
				//	fmt.Printf(" ip = %+v mask = %+v nw ip = %+v \n", ip, nw.mask, nw.ip )
				//}

				if match { break }
			}

			if match {
				ipCount++
			}
			if ipCount >= 3 {
				break
			}
		}

		if ipCount >= 3 && browserCount >= 3 {
			total++

			part := strings.Split(log.Email, "@")
			res = append(res, fmt.Sprintf("[%d] %s <%s [at] %s>", logPos + 1, log.Name, part[0], part[1])  )

		}
	}

	var str string
	if len(res) == 0 {
		str = fmt.Sprintf("Total: %d\n", total)
	} else {
		str = fmt.Sprintf("Total: %d\n%s\n", total, strings.Join(res, "\n") )
	}

	out.Write([]byte(str))

}

type Log struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Hits     []string `json:"hits"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

type Logs []*Log
