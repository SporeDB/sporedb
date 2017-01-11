package client

type cliMap map[string]func(arg string)

func (c *Client) getCLIMap() cliMap {
	return cliMap{
		"GET":     c.processGET,
		"VERSION": c.processVERSION,
		"SET":     c.processGeneric2("SET"),
		"CONCAT":  c.processGeneric2("CONCAT"),
		"ADD":     c.processGeneric2("ADD"),
		"MUL":     c.processGeneric2("MUL"),
	}
}
