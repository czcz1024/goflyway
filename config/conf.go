package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var splitLines = regexp.MustCompile(`\r?\n[\s\t]*`)

type conf_t map[string]map[string]interface{}

func (c *conf_t) getSection(section string) map[string]interface{} {
	if sec, ok := (*c)[section]; ok {
		return sec
	} else {
		return (*c)["default"] // return a dummy so Get* functions won't panic
	}
}

func (c *conf_t) Iterate(section string, callback func(key string)) {
	for k, _ := range c.getSection(section) {
		callback(k)
	}
}

func (c *conf_t) GetString(section, key string, defaultvalue string) string {
	if s, ok := c.getSection(section)[key].(string); ok {
		return s
	} else {
		return defaultvalue
	}
}

func (c *conf_t) GetInt(section, key string, defaultvalue int64) int64 {
	if s, ok := c.getSection(section)[key].(float64); ok {
		return int64(s)
	} else {
		return defaultvalue
	}
}

func (c *conf_t) GetFloat(section, key string, defaultvalue float64) float64 {
	if s, ok := c.getSection(section)[key].(float64); ok {
		return s
	} else {
		return defaultvalue
	}
}

func (c *conf_t) GetBool(section, key string, defaultvalue bool) bool {
	if s, ok := c.getSection(section)[key].(bool); ok {
		return s
	} else {
		return defaultvalue
	}
}

func (c *conf_t) GetArray(section, key string) []interface{} {
	if s, ok := c.getSection(section)[key].([]interface{}); ok {
		return s
	} else {
		return nil
	}
}

func ParseConf(str string) (*conf_t, error) {
	key, value, value2 := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	config := make(conf_t)
	config["default"] = make(map[string]interface{})

	var curSection map[string]interface{}

	for ln, line := range splitLines.Split(str, -1) {
		key.Reset()
		value.Reset()

		idx, p, quote := 0, key, byte(0)

	L:
		for idx < len(line) {
			c := line[idx]
			switch c {
			case '\r', '\n':

			case '[':
				if e := strings.Index(line, "]"); e > 0 {
					curSection = make(map[string]interface{})
					config[line[1:e]] = curSection
					break L
				} else {
					return nil, fmt.Errorf("invalid section name: %s at line %d", line, ln)
				}
			case ' ', '\t':
				if quote != 0 {
					p.WriteByte(c)
				}
			case '\'', '"':
				if idx > 0 && line[idx-1] == '\\' {
					p.WriteByte(c)
				} else if quote == 0 {
					quote = c
				} else if quote == c {
					quote = 0
				} else {
					return nil, fmt.Errorf("wrong paired quote: %s at line %d", line, ln)
				}
			case '#':
				if quote == 0 {
					break L
				}
			case '=':
				p = value
			default:
				p.WriteByte(c)
			}

			idx++
		}

		if quote != 0 {
			return nil, fmt.Errorf("missing closing quote: %s at line %d", string(quote), ln)
		}

		k := key.String()
		if curSection == nil || k == "" {
			continue
		}

		value2.Reset()
		v, idx := value.Bytes(), 0

		for idx < len(v) {
			if v[idx] == '\\' {
				if idx == len(v)-1 {
					return nil, fmt.Errorf("invalid escape: %s at line %d", value.String(), ln)
				}

				switch v[idx+1] {
				case 'n':
					value2.WriteByte('\n')
				case 'r':
					value2.WriteByte('\r')
				case 't':
					value2.WriteByte('\t')
				default:
					value2.WriteByte(v[idx+1])
				}
				idx += 2
			} else {
				value2.WriteByte(v[idx])
				idx++
			}
		}

		v2 := value2.String()

		_append := func(v interface{}) {
			if ov, existed := curSection[k]; existed {
				if arr, ok := ov.([]interface{}); ok {
					arr = append(arr, v)
				} else {
					curSection[k] = []interface{}{ov, v}
				}
			} else {
				curSection[k] = v
			}
		}

		switch v2 {
		case "on", "yes", "true":
			_append(true)
		case "off", "no", "false":
			_append(false)
		default:
			if num, err := strconv.ParseFloat(v2, 64); err == nil {
				_append(num)
			} else {
				_append(v2)
			}
		}

	}

	return &config, nil
}