package hostess

import (
	"io/ioutil"
	"os"
	"strings"
)

const defaultOSX = `
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##

127.0.0.1       localhost
255.255.255.255 broadcasthost
::1             localhost
fe80::1%lo0     localhost
`

const defaultLinux = `
127.0.0.1   localhost
127.0.1.1   HOSTNAME

# The following lines are desirable for IPv6 capable hosts
::1     localhost ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
ff02::3 ip6-allhosts
`

// Hostfile represents /etc/hosts (or a similar file, depending on OS), and
// includes a list of Hostnames. Hostfile includes
type Hostfile struct {
	Path  string
	Hosts Hostlist
	data  []byte
}

// NewHostfile creates a new Hostfile object from the specified file.
func NewHostfile(path string) *Hostfile {
	return &Hostfile{path, Hostlist{}, []byte{}}
}

// Read the contents of the hostfile from disk
func (h *Hostfile) Read() error {
	data, err := ioutil.ReadFile(h.Path)
	if err == nil {
		h.data = data
	}
	return err
}

// Parse reads
func (h *Hostfile) Parse() []error {
	var errs []error
	for _, v := range strings.Split(string(h.data), "\n") {
		for _, hostname := range ParseLine(v) {
			err := h.Hosts.Add(hostname)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// LoadHostfile creates a new Hostfile struct and tries to populate it from
// disk. Read and/or parse errors are returned as a slice.
func LoadHostfile() (hostfile *Hostfile, errs []error) {
	hostfile = NewHostfile(GetHostsPath())
	readErr := hostfile.Read()
	if readErr != nil {
		errs = []error{readErr}
		return
	}
	errs = hostfile.Parse()
	return
}

// TrimWS (Trim Whitespace) removes space, newline, and tabs from a string
// using strings.Trim()
func TrimWS(s string) string {
	return strings.Trim(s, " \n\t")
}

// ParseLine parses an individual line in a hostfile, which may contain one
// (un)commented ip and one or more hostnames. For example
//
//	127.0.0.1 localhost mysite1 mysite2
func ParseLine(line string) Hostlist {
	var hostnames Hostlist

	if len(line) == 0 {
		return hostnames
	}

	// Parse leading # for disabled lines
	enabled := true
	if line[0:1] == "#" {
		enabled = false
		line = TrimWS(line[1:])
	}

	// Parse other #s for actual comments
	line = strings.Split(line, "#")[0]

	// Replace tabs and multispaces with single spaces throughout
	line = strings.Replace(line, "\t", " ", -1)
	for strings.Contains(line, "  ") {
		line = strings.Replace(line, "  ", " ", -1)
	}

	// Break line into words
	words := strings.Split(line, " ")

	// Separate the first bit (the ip) from the other bits (the domains)
	ip := words[0]
	domains := words[1:]

	if LooksLikeIPv4(ip) || LooksLikeIPv6(ip) {
		for _, v := range domains {
			hostname := NewHostname(v, ip, enabled)
			hostnames = append(hostnames, hostname)
		}
	}

	return hostnames
}

// MoveToFront looks for string in a slice of strings and if it finds it, moves
// it to the front of the slice.
// Note: this could probably be made faster using pointers to switch the values
// instead of copying a bunch of crap, but it works and speed is not a problem.
func MoveToFront(list []string, search string) []string {
	for k, v := range list {
		if v == search {
			list = append(list[:k], list[k+1:]...)
		}
	}
	return append([]string{search}, list...)
}

// Format takes the current list of Hostnames in this Hostfile and turns it
// into a string suitable for use as an /etc/hosts file.
// Sorting uses the following logic:
// 1. List is sorted by IP address
// 2. Commented items are left in place
// 3. 127.* appears at the top of the list (so boot resolvers don't break)
// 4. When present, localhost will always appear first in the domain list
func (h *Hostfile) Format() string {
	// localhost := "127.0.0.1 localhost"

	localhosts := make(map[string][]string)
	ips := make(map[string][]string)

	// Map domains and IPs into slices of domains keyd by IP
	// 127.0.0.1 = [localhost, blah, blah2]
	// 2.2.2.3 = [domain1, domain2]
	for _, hostname := range h.Hosts {
		if hostname.IP.String()[0:4] == "127." {
			localhosts[hostname.IP.String()] = append(localhosts[hostname.IP.String()], hostname.Domain)
		} else {
			ips[hostname.IP.String()] = append(ips[hostname.IP.String()], hostname.Domain)
		}
	}

	// localhosts_keys := getSortedMapKeys(localhosts)
	// ips_keys := getSortedMapKeys(ips)
	var out []string

	// for _, ip := range localhosts_keys {
	// 	enabled := ip
	// 	enabled_b := false
	// 	disabled := "# " + ip
	// 	disabled_b := false
	// 	IP := net.ParseIP(ip)
	// 	for _, domain := range h.ListDomainsByIP(IP) {
	// 		hostname := *h.Hosts[domain]
	// 		if hostname.IP.Equal(IP) {
	// 			if hostname.Enabled {
	// 				enabled += " " + hostname.Domain
	// 				enabled_b = true
	// 			} else {
	// 				disabled += " " + hostname.Domain
	// 				disabled_b = true
	// 			}
	// 		}
	// 	}
	// 	if enabled_b {
	// 		out = append(out, enabled)
	// 	}
	// 	if disabled_b {
	// 		out = append(out, disabled)
	// 	}
	// }

	// for _, ip := range ips_keys {
	// 	enabled := ip
	// 	enabled_b := false
	// 	disabled := "# " + ip
	// 	disabled_b := false
	// 	IP := net.ParseIP(ip)
	// 	for _, domain := range h.ListDomainsByIP(IP) {
	// 		hostname := *h.Hosts[domain]
	// 		if hostname.IP.Equal(IP) {
	// 			if hostname.Enabled {
	// 				enabled += " " + hostname.Domain
	// 				enabled_b = true
	// 			} else {
	// 				disabled += " " + hostname.Domain
	// 				disabled_b = true
	// 			}
	// 		}
	// 	}
	// 	if enabled_b {
	// 		out = append(out, enabled)
	// 	}
	// 	if disabled_b {
	// 		out = append(out, disabled)
	// 	}
	// }

	return strings.Join(out, "\n")
}

// Save writes the Hostfile to disk to /etc/hosts or to the location specified
// by the HOSTESS_PATH environment variable (if set).
func (h *Hostfile) Save() error {
	// h.Format(h.Path)
	return nil
}

// GetHostsPath returns the location of the hostfile; either env HOSTESS_PATH
// or /etc/hosts if HOSTESS_PATH is not set.
func GetHostsPath() string {
	path := os.Getenv("HOSTESS_PATH")
	if path == "" {
		path = "/etc/hosts"
	}
	return path
}
