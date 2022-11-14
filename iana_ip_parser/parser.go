package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"math/big"

	_ "github.com/go-sql-driver/mysql"
	"github.com/smira/go-ftp-protocol/protocol"
)

type Header struct {
	Version   float32 `json:"version"`
	Registry  string  `json:"registry"`
	Serial    string  `json:"serial"`
	Records   int     `json:"records"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	UTCOffset string  `json:"utc_offset"`
}

type Record struct {
	Registry    string    `json:"registry"`
	CountryCode string    `json:"country_code"`
	Type        string    `json:"type"`
	Start       string    `json:"start"`
	Value       string    `json:"value"`
	Date        time.Time `json:"date"`
	Status      string    `json:"status"`
}

type Summary struct {
	Registry string `json:"registry"`
	Type     string `json:"type"`
	Count    string `json:"count"`
	Summary  string `json:"summary"`
}

type Downloads struct {
	Name string `json: "name"`
	Url  string `json: "url"`
}

type Database struct {
	Host         string `json:"host"`
	DatabaseName string `json:"database_name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type Config struct {
	Files          []Downloads `json:"files"`
	DownloadDir    string      `json:"download_dir"`
	DatabaseConfig Database    `json:"database"`
}

func getFile(file string, url string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	transport := &http.Transport{}
	transport.RegisterProtocol("ftp", &protocol.FTPRoundTripper{})
	client := &http.Client{Transport: transport}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func ipRange(str string) (net.IP, net.IP) {
	_, mask, err := net.ParseCIDR(str)
	if err != nil {
		fmt.Println("Error parsing CIDR - ", str, " - ", err)
		return nil, nil
	}
	first := mask.IP.Mask(mask.Mask).To16()
	second := make(net.IP, len(first))
	copy(second, first)
	ones, _ := mask.Mask.Size()
	if first.To4() != nil {
		ones += 96
	}
	lastBytes := (8*16 - ones) / 8
	lastBits := 8 - ones%8
	or := 0
	for x := 0; x < lastBits; x++ {
		or = or*2 + 1
	}
	for x := 16 - lastBytes; x < 16; x++ {
		second[x] = 0xff
	}
	if lastBits < 8 {
		second[16-lastBytes-1] |= byte(or)
	}
	return first, second
}

func nextIP4(ip net.IP, inc uint) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	v += inc
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}

// Inet_Aton converts an IPv4 net.IP object to a 64 bit integer.
func Inet_Aton(ip net.IP) int64 {
	ipv4Int := big.NewInt(0)
	ipv4Int.SetBytes(ip.To4())
	return ipv4Int.Int64()
}

// Inet6_Aton converts an IP Address (IPv4 or IPv6) net.IP object to a hexadecimal
// representaiton. This function is the equivalent of
// inet6_aton({{ ip address }}) in MySQL.
func Inet6_Aton(ip net.IP) *big.Int {
	ipv4 := false
	if ip.To4() != nil {
		ipv4 = true
	}

	ipInt := big.NewInt(0)
	if ipv4 {
		ipInt.SetBytes(ip.To4())
		return ipInt
	}

	ipInt.SetBytes(ip.To16())
	return ipInt
}

func main() {
	separatorPtr := flag.String("separator", "|", "Default IANA csv separator")
	configPtr := flag.String("config", "iana_ip_parser.conf", "Config file")
	logPtr := flag.String("log", "iana_ip_parser.log", "Log file")

	flag.Parse()

	fmt.Println("IANA IP Parser v1.0")
	fmt.Println("Developed by Otávio Ribeiro <otavio.ribeiro@gmail.com>\n")

	os.Remove(*logPtr)
	logFile, error := os.OpenFile(*logPtr, os.O_CREATE|os.O_WRONLY, 0660)
	if error != nil {
		fmt.Println("Error creating log file. Do you have write permission?")
		return
	}
	defer logFile.Close()

	configData, err := ioutil.ReadFile(*configPtr)
	if err != nil {
		fmt.Println("File", *configPtr, "not found\n")
		return
	}

	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		fmt.Println("Error reading", *configPtr, ":", "Invalid Config File\n")
		fmt.Println(err)
		return
	}

	db, err := sql.Open("mysql", config.DatabaseConfig.Username+":"+config.DatabaseConfig.Password+"@"+config.DatabaseConfig.Host+"/"+config.DatabaseConfig.DatabaseName)
	if err != nil {
		fmt.Println("Error connecting mysql database")
		fmt.Println(err)
		return
	}
	defer db.Close()

	for _, element := range config.Files {
		fmt.Println("Downloading File:", element.Url)

		fileName := config.DownloadDir + "/" + element.Name + ".csv"

		if _, err := os.Stat(fileName); err != nil {
			err = getFile(fileName, element.Url)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("File", fileName, "already exists.")
		}

		csvFile, error := os.Open(fileName)
		if error != nil {
			fmt.Println("File", fileName, "not found\n")
			return
		}
		defer csvFile.Close()

		reader := bufio.NewReader(csvFile)

		var stmt *sql.Stmt
		stmt, err = db.Prepare("INSERT INTO iana_records (registry, country_code, reg_type, reg_start, reg_value, reg_date, reg_status, ip_start_range, ip_end_range) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")

		if err != nil {
			fmt.Println("Error Preparing Statment")
			fmt.Println(err)
			return
		}
		defer stmt.Close()

		count := 0
		for {
			line, error := reader.ReadString('\n')
			lineArr := strings.Split(line, *separatorPtr)

			if error == io.EOF {
				break
			} else if error != nil {
				fmt.Println(error)
				return
			}

			fmt.Print("Parsing line:", strings.Join(lineArr, "|"))

			if lineArr[0][0] == '#' {
				continue
			}

			count++

			if count == 1 {
				fmt.Println("Ignoring Header")
				//var lineObj Header;
				//lineObj = Header{
				//}
			} else if strings.Index(strings.Join(lineArr, ""), "summary") > -1 {
				fmt.Println("Ignoring Summary")
				//var lineObj Summary;
				//lineObj = Summary{
				//}
			} else {
				date, err := time.Parse("20060102", lineArr[5])

				var lineObj Record
				lineObj = Record{
					Registry:    lineArr[0],
					CountryCode: lineArr[1],
					Type:        lineArr[2],
					Start:       lineArr[3],
					Value:       lineArr[4],
					Date:        date,
					Status:      lineArr[6],
				}

				var ip_range_start net.IP
				var ip_range_end net.IP

				if lineObj.Type == "ipv6" {
					ip_range_start, ip_range_end = ipRange(lineObj.Start + "/" + lineObj.Value)
					fmt.Println("Start Range:", Inet6_Aton(ip_range_start))
					fmt.Println("End Range:", Inet6_Aton(ip_range_end))
				} else if lineObj.Type == "ipv4" {
					ip_range_start = net.ParseIP(lineObj.Start)
					len, _ := strconv.Atoi(lineObj.Value)
					ip_range_end = nextIP4(ip_range_start, uint(len))
					fmt.Println("Start Range:", Inet_Aton(ip_range_start))
					fmt.Println("End Range:", Inet_Aton(ip_range_end))
				}

				if lineObj.Type == "ipv6" || lineObj.Type == "ipv4" {
					_, err = stmt.Exec(
						lineObj.Registry,
						lineObj.CountryCode,
						lineObj.Type,
						lineObj.Start,
						lineObj.Value,
						lineObj.Date,
						lineObj.Status,
						Inet6_Aton(ip_range_start).Bytes(),
						Inet6_Aton(ip_range_end).Bytes(),
					)
				} else {
					_, err = stmt.Exec(
						lineObj.Registry,
						lineObj.CountryCode,
						lineObj.Type,
						lineObj.Start,
						lineObj.Value,
						lineObj.Date,
						lineObj.Status,
						nil,
						nil,
					)
				}

				if err != nil {
					fmt.Println("Error Executing Insert Query")
					fmt.Println(err)
					logFile.WriteString("Error Executing Insert Query\n")
					logFile.WriteString(err.Error() + "\n")
				}

				recordJson, _ := json.Marshal(lineObj)
				logString := string(recordJson)

				fmt.Println(logString)
			}
		}
	}
}
