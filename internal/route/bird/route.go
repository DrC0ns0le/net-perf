package bird

import (
	"encoding/binary"
	"fmt"
	"hash"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
)

type Route struct {
	Network  *net.IPNet
	Paths    []BGPPath
	OriginAS int
}
type BGPPath struct {
	AS              int
	ASPath          []int
	Next            net.IP
	Interface       string
	MED             int
	LocalPreference int
	OriginType      string
}

func GetRoutes(mode string) ([]Route, hash.Hash64, error) {
	conn, reader, writer, err := connect(mode)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	// Initialize hash
	h := xxhash.New()

	// Send command to show routes
	_, err = writer.WriteString("show route all\n")
	if err != nil {
		return nil, nil, err
	}
	writer.Flush()

	var line string
	var rt string
	var route Route

	var routes []Route
	var bgpPath BGPPath

	var isBGP bool

	for {
		line, err = reader.ReadString('\n')
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
				bgpPath = BGPPath{}
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
					route = Route{}
				}

				// check if this route is a BGP route
				nextLine, err := reader.ReadString('\n')
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
