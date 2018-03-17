package main

import (
	"io"
	"fmt"
	"strings"
	"encoding/json"
	"sync"
	"regexp"
	"sort"
	"bufio"
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
	//var logs Logs

	//wg := &sync.WaitGroup{}
	//wg.Add(1)

	//go func(wg *sync.WaitGroup) {
	//	defer wg.Done()
		for _, nw := range networks {
			ip, ipMask, _ := ParseCIDR(nw)
			nwrks = append(nwrks, Network{  ip: ip, mask: ipMask  })
		}
	//}(wg)
	//wg.Wait()

	//go func(wg *sync.WaitGroup) {
	//	defer wg.Done()
	//
	//	all, _ := ioutil.ReadAll(in)
	//
	//	all = bytes.Replace(all, []byte("\n"), []byte(","), -1)
	//	all = all[:len(all)-1]
	//
	//	//all := []byte( "{") , all...,  []byte("}")
	//	allr := append(  []byte( "["), all...)
	//	allr = append(  allr,  byte(']') )
	//	fmt.Println(string(allr))
	//
	//
	//	logs := make(Logs, 0)
	//	 json.Unmarshal(allr, logs)
	//	//panic(err)
	//	fmt.Println(  logs )
	//
	//
	//	////logStr := strings.Split(string(all), "\n")
	//	//for _, v := range bytes.Split(all, []byte("\n")) {
	//	//	if len(v) == 0  {
	//	//		continue
	//	//	}
	//	//
	//	//	raw := &Log{}
	//	//	json.Unmarshal(v, raw)
	//	//	logs = append(logs, raw)
	//	//}
	//
	//}(wg)


	//comp, _ := regexp.Compile(`hits\"\:(\[(\"\d+\.\d+\.\d+\.\d+\",{0,1})+\])`)
	//comp, _ := regexp.Compile(`hits\"\:\[(\"\d+\.\d+\.\d+\.\d+\",{0,1})+.*\]`)
	//logs := strings.Split(string(all), "\n")

	var total = 0
	messages := make(chan *respos)
	waitchn := make(chan []*respos)
	tasks := make(chan *Task, 4)

	//batch := len(logs) / 2
	//if batch == 0 {
	//	batch = 1
	//}
	//wg2 := &sync.WaitGroup{}

	go func(messages chan *respos, waitchan chan []*respos) {
		res  := make([]*respos, 0, 1000)
		for r := range messages {
			res = append(res, r)
		}
		sort.Sort(  ByLex(res) )
		waitchan <- res
	}(messages, waitchn)

	wg2 := &sync.WaitGroup{}

	for  i := 0; i < 8; i++ {
		wg2.Add(1)
		go solve(tasks, browserRegex, nwrks, messages, wg2)
	}


	re := bufio.NewReader(in)

	pos := 0
	for true {
		b, err := re.ReadBytes('\n')
		if err != nil {
			break
		}

		if len(b) == 0  {
			continue
		}

		//tasks <- &Task{ LogStr: []byte( string(b) ), Pos: pos }
		tasks <- &Task{ LogStr:  b, Pos: pos }

		pos++
	}


	//all, _ := ioutil.ReadAll(in)
	//
	//
	//for pos, v := range bytes.Split(all, []byte("\n")) {
	//	if len(v) == 0  {
	//		continue
	//	}
	//
	//	tasks <- &Task{ LogStr: v, Pos: pos }
	//
	//	//raw := &Log{}
	//	//json.Unmarshal(v, raw)
	//	//logs = append(logs, raw)
	//}
	//}
	close(tasks)
	wg2.Wait()
	close(messages)

	res := <- waitchn

	//
	//offset := 0
	//for logs != nil && len(logs) > 0  {
	//	wg.Add(1)
	//
	//	var bbatch Logs
	//	if len(logs) > batch {
	//		bbatch = logs[:batch]
	//		logs = logs[batch:]
	//	} else {
	//		bbatch = logs
	//		logs = nil
	//	}
	//
	//	wg2.Add(1)
	//	go solve(bbatch, browserRegex, nwrks, wg2, messages, offset)
	//	offset += len(bbatch)
	//}


	total = len(res)
	var str string
	if len(res) == 0 {
		str = fmt.Sprintf("Total: %d\n", total)
	} else {
		a := make([]string, len(res))
		for k, v := range res {
			a[k] = v.s
		}

		str = fmt.Sprintf("Total: %d\n%s\n", total, strings.Join(a, "\n") )
	}

	out.Write([]byte(str))

}

type respos struct {
	s string
	pos int
}

type ByLex []*respos
func (a ByLex) Len() int           { return len(a) }
func (a ByLex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLex) Less(i, j int) bool { return a[i].pos < a[j].pos }

func solve (tasks chan *Task, browserRegex *regexp.Regexp, nwrks []Network, messages chan *respos, wg *sync.WaitGroup) {
	defer wg.Done()
	//comp, _ := regexp.Compile(`hits\"\:\[(.*)\]`)

	//var res2 []respos

	//for logPos, log := range logs {
	for t := range tasks {

		log := &Log{}
		json.Unmarshal(t.LogStr, log)

		//if len(log) == 0 {continue}
		//b := comp.FindAllStringSubmatch(log, -1)
		//addrs := strings.Split(b[0][1], ",")

		ipCount := 0
		browserCount := 0

		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func(wg *sync.WaitGroup)  {
			defer wg.Done()
			for _, br := range log.Browsers {
				if browserRegex.MatchString(br) {
					browserCount++
				}

				if browserCount >= 3 {
					break
				}

			}
		}(wg)

		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			match := false
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
		}(wg)

		wg.Wait()

		if ipCount >= 3 && browserCount >= 3 {
			part := strings.Split(log.Email, "@")
			//res2 = append(res2, respos{  s: fmt.Sprintf("[%d] %s <%s [at] %s>", offset + logPos + 1, log.Name, part[0], part[1]), pos: offset + logPos  } )

			messages <- &respos{ s: fmt.Sprintf("[%d] %s <%s [at] %s>", t.Pos + 1, log.Name, part[0], part[1]), pos: t.Pos  }
		}
	}

	//messages <- res2
}

type Task struct {
	LogStr []byte
	Pos int
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

	Pos int
}

type Logs []*Log
