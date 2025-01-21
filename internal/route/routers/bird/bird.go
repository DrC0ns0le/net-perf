package bird

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/routers"
	"github.com/cespare/xxhash"
)

const (
	birdSocket  = "/run/bird/bird.ctl"
	bird6Socket = "/run/bird/bird6.ctl" // for IPv6 Socket
	timeout     = 10 * time.Second
)

type Bird struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewRouter() *Bird {
	return &Bird{}
}

func (b *Bird) GetConfig(mode string) (routers.Config, error) {
	// Open the file
	file, err := os.Open(
		fmt.Sprintf("/etc/bird/bird%s.conf", func() string {
			if mode == "v6" {
				return "6"
			}
			return ""
		}()),
	)
	if err != nil {
		return routers.Config{}, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Regular expression to match "local as" followed by numbers
	re := regexp.MustCompile(`local\s+as\s+(\d+)`)

	config := routers.Config{}

	// Scan through each line
	for scanner.Scan() {
		line := scanner.Text()

		// Check if line contains the pattern we're looking for
		if strings.Contains(line, "local as") {
			// Extract the number using regex
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				// Parse the captured number
				var asNumber int
				_, err := fmt.Sscanf(matches[1], "%d", &asNumber)
				if err != nil {
					return config, fmt.Errorf("error parsing AS number: %v", err)
				}
				config.ASNumber = asNumber
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return config, fmt.Errorf("error reading file: %v", err)
	}

	return config, nil
}

func (b *Bird) GetRoutes(mode string) ([]routers.Route, hash.Hash64, error) {
	var err error
	err = b.connect(mode)
	if err != nil {
		return nil, nil, err
	}
	defer b.conn.Close()

	// Initialize hash
	h := xxhash.New()

	// Send command to show routes
	_, err = b.writer.WriteString("show route all\n")
	if err != nil {
		return nil, nil, err
	}
	b.writer.Flush()

	var line string
	var rt string
	var route routers.Route

	var routes []routers.Route
	var bgpPath routers.BGPPath

	var isBGP bool

	for {
		line, err = b.reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

	READLINE:
		if strings.HasPrefix(line, "0000") {
			break // End of data
		}

		// route entry
		if strings.HasPrefix(line, "1007-") {

			// complete previous BGP path and append to route
			if bgpPath.Next != nil || bgpPath.Interface != "" {
				route.Paths = append(route.Paths, bgpPath)

				// hash the BGP path
				if bgpPath.Next != nil {
					h.Write([]byte(bgpPath.Next.String()))
				}
				h.Write([]byte(bgpPath.Interface))
				binary.Write(h, binary.LittleEndian, uint32(bgpPath.AS))
				binary.Write(h, binary.LittleEndian, uint32(bgpPath.LocalPreference))
				binary.Write(h, binary.LittleEndian, uint32(bgpPath.MED))
				h.Write([]byte(bgpPath.OriginType))
				for _, as := range bgpPath.ASPath {
					binary.Write(h, binary.LittleEndian, uint32(as))
				}
				bgpPath = routers.BGPPath{}
			}

			// split to fields separated by space
			splits := strings.Fields(line)
			rt = strings.TrimPrefix(splits[0], "1007-")

			// check if this path is start of a new route
			if rt != "" {
				// append previous completed route
				if route.Network != nil {
					routes = append(routes, route)
					// Hash the completed route
					h.Write([]byte(route.Network.String()))
					binary.Write(h, binary.LittleEndian, uint32(route.OriginAS))
					route = routers.Route{}
				}

				// check if this route is a BGP route
				nextLine, err := b.reader.ReadString('\n')
				if err != nil {
					return nil, nil, err
				}
				var typeFields []string
				if strings.HasPrefix(nextLine, "1008-	Type:") {
					typeFields = strings.Fields(strings.SplitN(nextLine, ":", 2)[1])
				} else {
					goto READLINE
				}
				if len(typeFields) > 0 && strings.TrimSpace(typeFields[0]) == "BGP" {
					isBGP = true
					_, ipnet, _ := net.ParseCIDR(rt)
					route.Network = ipnet

					if route.OriginAS == 0 {
						originAS := splits[len(splits)-1]
						if originAS != "" {
							// Use regex to extract only digits
							re := regexp.MustCompile(`\d+`)
							match := re.FindString(originAS)

							// Parse the matched string to an integer
							if match != "" {
								originASint, err := strconv.Atoi(match)
								if err != nil {
									return nil, nil, fmt.Errorf("failed to parse origin AS: %w", err)
								}
								route.OriginAS = originASint
							}
						}
					}
					bgpPath.Next = net.ParseIP(splits[2])
					bgpPath.Interface = splits[4]
				} else {
					isBGP = false
				}

			} else if isBGP {
				// This line is part of the previous BGP route
				bgpPath.Next = net.ParseIP(splits[2])
				bgpPath.Interface = splits[4]
			}
		}

		if (strings.HasPrefix(line, "1012-") || strings.HasPrefix(strings.TrimSpace(line), "BGP.")) && isBGP {
			halfSplit := strings.SplitN(line, "BGP.", 2)
			split := strings.SplitN(halfSplit[1], ":", 2)

			switch split[0] {
			case "origin":
				bgpPath.OriginType = strings.TrimSpace(split[1])
			case "next_hop":
				bgpPath.Next = net.ParseIP(strings.Fields(strings.TrimSpace(split[1]))[0])
			case "as_path":
				if split[1] != "" {
					sp := strings.Split(strings.TrimSpace(split[1]), " ")

					for i, as := range sp {
						asInt, err := strconv.Atoi(strings.TrimSpace(as))
						if err != nil {
							return nil, nil, fmt.Errorf("failed to parse AS path: %w", err)
						}
						bgpPath.ASPath = append(bgpPath.ASPath, asInt)
						if i == 0 {
							bgpPath.AS = asInt
						}
					}
				} else {
					bgpPath.ASPath = make([]int, 0)
				}
			case "local_pref":
				bgpPath.LocalPreference, err = strconv.Atoi(strings.TrimSpace(split[1]))
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse local preference: %w", err)
				}
			case "med":
				bgpPath.MED, err = strconv.Atoi(strings.TrimSpace(split[1]))
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse MED: %w", err)
				}
			}
		}
	}

	// append the last completed route
	if route.Network != nil {
		if bgpPath.Next != nil || bgpPath.Interface != "" {
			route.Paths = append(route.Paths, bgpPath)

			// hash the final BGP path
			if bgpPath.Next != nil {
				h.Write([]byte(bgpPath.Next.String()))
			}
			h.Write([]byte(bgpPath.Interface))
			binary.Write(h, binary.LittleEndian, uint32(bgpPath.AS))
			binary.Write(h, binary.LittleEndian, uint32(bgpPath.LocalPreference))
			binary.Write(h, binary.LittleEndian, uint32(bgpPath.MED))
			h.Write([]byte(bgpPath.OriginType))
			for _, as := range bgpPath.ASPath {
				binary.Write(h, binary.LittleEndian, uint32(as))
			}
		}
		routes = append(routes, route)
		// hash the final route
		h.Write([]byte(route.Network.String()))
		binary.Write(h, binary.LittleEndian, uint32(route.OriginAS))
	}

	return routes, h, nil
}

func (b *Bird) connect(mode string) error {
	var err error
	if b.conn != nil {
		b.conn.Close()
	}
	b.conn, err = net.DialTimeout("unix", func() string {
		if mode == "v6" {
			return bird6Socket
		}
		return birdSocket
	}(), timeout)
	if err != nil {
		return fmt.Errorf("error connecting to BIRD socket: %v", err)
	}

	b.conn.SetDeadline(time.Now().Add(timeout))

	b.reader = bufio.NewReader(b.conn)
	b.writer = bufio.NewWriter(b.conn)

	// Read the welcome message
	welcome, err := b.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading welcome message: %v", err)
	}
	if !strings.HasPrefix(welcome, "0001 ") {
		return fmt.Errorf("invalid welcome message: %s", welcome)
	}

	return nil
}
