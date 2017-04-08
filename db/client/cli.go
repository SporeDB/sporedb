package client

import "fmt"

type cliMap map[string]func(arg string)

func (c *Client) getCLIMap() cliMap {
	return cliMap{
		"GET":     c.processGET,
		"VERSION": c.processVERSION,
		"SET":     c.processGeneric2("SET"),
		"CONCAT":  c.processGeneric2("CONCAT"),
		"ADD":     c.processGeneric2("ADD"),
		"MUL":     c.processGeneric2("MUL"),
		"SADD":    c.processGeneric2("SADD"),
		"SREM":    c.processGeneric2("SREM"),
		"POL":     c.SetPolicy,
	}
}

// SetPolicy sets the active client policy. Used for CLI mode mainly.
func (c *Client) SetPolicy(pol string) {
	c.policy = pol
	fmt.Println("Now using policy", pol)
}
