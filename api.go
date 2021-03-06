package api

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var (
	mainAPIURL string
)

// Init sets API url and authorization parameters
func Init(url string, username string, password string) {
	mainAPIURL = url
	SetBasicAuth(username, password)
}

// GetConfigInfo returns current configuration
func GetConfigInfo() (ConfigInfo, error) {
	var result ConfigInfo
	err := Get(mainAPIURL+"/v1/config", &result)
	return result, err
}

// GetSlaveList returns list of defined slave probes
func GetSlaveList() ([]string, error) {
	var result map[string]string
	err := Get(mainAPIURL+"/v1/slaves", &result)
	list := make([]string, 0, len(result))
	for slave := range result {
		list = append(list, slave)
	}
	return list, err
}

// GetSlavesIPs returns list of defined slave probes with their ips
func GetSlavesIPs() (map[string]string, error) {
	var result map[string]string
	err := Get(mainAPIURL+"/v1/slaves", &result)
	slave2ip := make(map[string]string, len(result))
	for slave, addr := range result {
		slave2ip[slave] = strings.SplitN(addr, ":", 2)[0]
	}
	return slave2ip, err
}

// GetSlavesSources returns list of defined slave probes with IPv4 ips from which ping/traces are initiated
func GetSlavesSources() (map[string]string, error) {
	slaves, err := GetSlavesStatus()
	if nil != err {
		return nil, err
	}

	result := make(map[string]string, len(slaves))
	for slave, status := range slaves {
		result[slave] = status.Source
	}
	return result, err
}

// GetSlavesSources6 returns list of defined slave probes with IPv6 ips from which ping/traces are initiated
func GetSlavesSources6() (map[string]string, error) {
	slaves, err := GetSlavesStatus()
	if nil != err {
		return nil, err
	}

	result := make(map[string]string, len(slaves))
	for slave, status := range slaves {
		result[slave] = status.Source6
	}
	return result, err
}

// GetSlavesStatus returns actual slaves status
func GetSlavesStatus() (map[string]SlaveStatus, error) {
	var result map[string]SlaveStatus
	err := Get(mainAPIURL+"/v1/status/slaves", &result)
	return result, err
}

// GetSlavesAddrs returns list of defined slave probes with their ip:port
func GetSlavesAddrs() (map[string]string, error) {
	var result map[string]string
	err := Get(mainAPIURL+"/v1/slaves", &result)
	return result, err
}

// AddSlave adds slave to master on ip:port with name
// and possibly copy list of ips from just existing slave copyFrom
func AddSlave(ip net.IP, port uint16, name string, copyFrom string) error {
	return _okResultSend("POST", mainAPIURL+"/v1/slaves", map[string]interface{}{
		"ip":   ip.String(),
		"port": port,
		"name": name,
		"copy": copyFrom,
	})
}

// DeleteSlave removes slave from master
func DeleteSlave(slave string) error {
	return _okResultSend("DELETE", mainAPIURL+"/v1/slaves?slave="+url.QueryEscape(slave), nil)
}

// AddIP is simple interface for single IP adding
func AddIP(ip string, slaves []string, description string, groups []string, favorite bool) error {
	return _okResultSend("PUT", mainAPIURL+"/v1/config/ping/"+ip, TestDesc{
		Description: ip + " " + description,
		Favorite:    favorite,
		Groups:      groups,
		Slaves:      slaves,
	})
}

// AddIPs function adds multiply ips using only one API call
func AddIPs(ips []string, slaves []string, description string, groups []string, favorite bool) error {
	payload := make(map[string]TestDesc, len(ips))

	for _, ip := range ips {
		payload[ip] = TestDesc{
			Description: ip + " " + description,
			Favorite:    favorite,
			Groups:      groups,
			Slaves:      slaves,
		}
	}

	return _okResultSend("PUT", mainAPIURL+"/v1/mconfig/add", map[string]interface{}{
		"ips": payload,
	})
}

// AddIPsRaw is extended function adds multiply ips using only one API call
func AddIPsRaw(ips map[string]TestDesc) error {
	return _okResultSend("PUT", mainAPIURL+"/v1/mconfig/add", map[string]interface{}{
		"ips": ips,
	})
}

// DeleteIP removes one IP from cocopacket instance
func DeleteIP(ip string) error {
	return _okResultSend("DELETE", mainAPIURL+"/v1/config/ping/"+ip, nil)
}

// DeleteIPs function deletes multiply ips using only one API call
func DeleteIPs(ips []string) error {
	return _okResultSend("PUT", mainAPIURL+"/v1/mconfig/delete", map[string]interface{}{
		"ips": ips,
	})
}

// ListUsers return map with logins and associated boolean indicating if user is admin
func ListUsers() (map[string]bool, error) {
	var users map[string]bool
	err := Get(mainAPIURL+"/v1/users", &users)
	return users, err
}

// AddUser adds new user (or replaces existing)
func AddUser(login string, password string, admin bool) (map[string]bool, error) {

	var users map[string]bool
	t := "user"
	if admin {
		t = "admin"
	}

	err := SendForm("PUT", mainAPIURL+"/v1/users", url.Values{
		"login":  []string{login},
		"passwd": []string{password},
		"type":   []string{t},
	}, &users)
	if nil != err {
		return nil, err
	}

	return users, err
}

// DeleteUser removes user from master
func DeleteUser(login string) (map[string]bool, error) {

	var users map[string]bool

	err := Send("DELETE", mainAPIURL+"/v1/users?login="+url.QueryEscape(login), nil, &users)
	if nil != err {
		return nil, err
	}

	return users, err
}

// GroupStats returns stats for all IPs/URLs in group for about last 24 hours with 1-hour aggregation (report -> limit only to ip+slaves selected for report using frontend)
func GroupStats(group string, report bool) (GroupStatsData, error) {
	var data GroupStatsData
	reportAdd := ""
	if report {
		reportAdd = "?report=true"
	}
	err := Get(mainAPIURL+"/v1/catstats/"+url.QueryEscape(group+"->")+reportAdd, &data)
	return data, err
}

// GroupLastStats returns stats for all IPs/URLs in group on one slave for last minute period (used for exports to other systems)
func GroupLastStats(group string, slave string) (ips map[string]*AvgChunk, urls map[string]*AvgChunk, err error) {
	var data struct {
		Ping   map[string]*AvgChunk `json:"Ping"`
		HTTP   map[string]*AvgChunk `json:"HTTP"`
		Result string               `json:"result"`
		Error  string               `json:"error"`
	}
	err = Get(mainAPIURL+"/v1/minute/"+url.QueryEscape(group+"->")+"?slave="+url.QueryEscape(slave), &data)
	if nil == err && "error" == data.Result {
		err = errors.New(data.Error)
	}
	return data.Ping, data.HTTP, err
}

// IPsSetSlaves add/remove slaves for list of ips, in case of "true" slave is added, in case of "false" slave removed, unlisted slaves are untouched
func IPsSetSlaves(ips []string, slaves map[string]bool) error {
	return _okResultSend("PUT", mainAPIURL+"/v1/mconfig/slaves", map[string]interface{}{
		"ips":    ips,
		"slaves": slaves,
	})
}

// GroupSetSlaves add/remove slaves for all ips in group, in case of "true" slave is added, in case of "false" slave removed, unlisted slaves are untouched; pass recursive=true to include subgroups
func GroupSetSlaves(group string, slaves map[string]bool, recursive bool) error {
	return _okResultSend("PUT", mainAPIURL+"/v1/groupslaves/"+url.QueryEscape(group+"->"), map[string]interface{}{
		"recursive": recursive,
		"slaves":    slaves,
	})
}
