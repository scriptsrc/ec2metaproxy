package kingpin

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flyinprogrammer/ec2metaproxy/Godeps/_workspace/src/github.com/alecthomas/units"
)

// NOTE: Most of the base type values were lifted from:
// http://golang.org/src/pkg/flag/flag.go?s=20146:20222

// Value is the interface to the dynamic value stored in a flag.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolFlag() bool method returning true, the command-line
// parser makes --name equivalent to -name=true rather than using the next
// command-line argument, and adds a --no-name counterpart for negating the
// flag.
type Value interface {
	String() string
	Set(string) error
}

// Getter is an interface that allows the contents of a Value to be retrieved.
// It wraps the Value interface, rather than being part of it, because it
// appeared after Go 1 and its compatibility rules. All Value types provided
// by this package satisfy the Getter interface.
type Getter interface {
	Value
	Get() interface{}
}

// Optional interface to indicate boolean flags that don't accept a value, and
// implicitly have a --no-<x> negation counterpart.
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// Optional interface for arguments that cumulatively consume all remaining
// input.
type remainderArg interface {
	Value
	IsCumulative() bool
}

// -- bool Value
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	if s == "" {
		s = "true"
	}
	v, err := strconv.ParseBool(s)
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) }

func (b *boolValue) String() string { return fmt.Sprintf("%v", *b) }

func (b *boolValue) IsBoolFlag() bool { return true }

// -- int Value
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = intValue(v)
	return err
}

func (i *intValue) Get() interface{} { return int(*i) }

func (i *intValue) String() string { return fmt.Sprintf("%v", *i) }

// -- int64 Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() interface{} { return int64(*i) }

func (i *int64Value) String() string { return fmt.Sprintf("%v", *i) }

// -- uint Value
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uintValue(v)
	return err
}

func (i *uintValue) Get() interface{} { return uint(*i) }

func (i *uintValue) String() string { return fmt.Sprintf("%v", *i) }

// -- uint64 Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) }

func (i *uint64Value) String() string { return fmt.Sprintf("%v", *i) }

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return fmt.Sprintf("%s", *s) }

// -- float64 Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = float64Value(v)
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) }

func (f *float64Value) String() string { return fmt.Sprintf("%v", *f) }

// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	*d = durationValue(v)
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) }

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

// -- []string Value
type stringsValue []string

func newStringsValue(p *[]string) *stringsValue {
	return (*stringsValue)(p)
}

func (s *stringsValue) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *stringsValue) String() string {
	return strings.Join(*s, ",")
}

func (s *stringsValue) IsCumulative() bool {
	return true
}

// -- map[string]string Value
type stringMapValue map[string]string

func newStringMapValue(p *map[string]string) *stringMapValue {
	return (*stringMapValue)(p)
}

var stringMapRegex = regexp.MustCompile("[:=]")

func (s *stringMapValue) Set(value string) error {
	parts := stringMapRegex.Split(value, 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}
	(*s)[parts[0]] = parts[1]
	return nil
}
func (s *stringMapValue) String() string {
	return fmt.Sprintf("%s", map[string]string(*s))
}

func (s *stringMapValue) IsCumulative() bool {
	return true
}

// -- net.IP Value
type ipValue net.IP

func newIPValue(p *net.IP) *ipValue {
	return (*ipValue)(p)
}

func (i *ipValue) Set(value string) error {
	if ip := net.ParseIP(value); ip == nil {
		return fmt.Errorf("'%s' is not an IP address", value)
	} else {
		*i = *(*ipValue)(&ip)
		return nil
	}
}

func (i *ipValue) String() string {
	return (*net.IP)(i).String()
}

// -- *net.TCPAddr Value
type tcpAddrValue struct {
	addr **net.TCPAddr
}

func newTCPAddrValue(p **net.TCPAddr) *tcpAddrValue {
	return &tcpAddrValue{p}
}

func (i *tcpAddrValue) Set(value string) error {
	if addr, err := net.ResolveTCPAddr("tcp", value); err != nil {
		return fmt.Errorf("'%s' is not a valid TCP address: %s", value, err)
	} else {
		*i.addr = addr
		return nil
	}
}

func (i *tcpAddrValue) String() string {
	return (*i.addr).String()
}

// -- []*net.TCPAddr Value
type tcpAddrsValue []*net.TCPAddr

func newTCPAddrsValue(p *[]*net.TCPAddr) *tcpAddrsValue {
	return (*tcpAddrsValue)(p)
}

func (i *tcpAddrsValue) Set(value string) error {
	if addr, err := net.ResolveTCPAddr("tcp", value); err != nil {
		return fmt.Errorf("'%s' is not a valid TCP address: %s", value, err)
	} else {
		*i = append(*i, addr)
		return nil
	}
}

func (i *tcpAddrsValue) String() string {
	s := make([]string, 0, len(*i))
	for _, a := range *i {
		s = append(s, a.String())
	}
	return strings.Join(s, ",")
}

// -- existingFile Value

type fileStatValue struct {
	path      *string
	predicate func(os.FileInfo) error
}

func newFileStatValue(p *string, predicate func(os.FileInfo) error) *fileStatValue {
	return &fileStatValue{
		path:      p,
		predicate: predicate,
	}
}

func (e *fileStatValue) Set(value string) error {
	if s, err := os.Stat(value); os.IsNotExist(err) {
		return fmt.Errorf("path '%s' does not exist", value)
	} else if err != nil {
		return err
	} else if err := e.predicate(s); err != nil {
		return err
	}
	*e.path = value
	return nil
}

func (e *fileStatValue) String() string {
	return *e.path
}

// -- os.File value

type fileValue struct {
	f    **os.File
	flag int
	perm os.FileMode
}

func newFileValue(p **os.File, flag int, perm os.FileMode) *fileValue {
	return &fileValue{p, flag, perm}
}

func (f *fileValue) Set(value string) error {
	if fd, err := os.OpenFile(value, f.flag, f.perm); err != nil {
		return err
	} else {
		*f.f = fd
		return nil
	}
}

func (f *fileValue) String() string {
	if *f.f == nil {
		return "<nil>"
	}
	return (*f.f).Name()
}

// -- url.URL Value
type urlValue struct {
	u **url.URL
}

func newURLValue(p **url.URL) *urlValue {
	return &urlValue{p}
}

func (u *urlValue) Set(value string) error {
	if url, err := url.Parse(value); err != nil {
		return fmt.Errorf("invalid URL: %s", err)
	} else {
		*u.u = url
		return nil
	}
}

func (u *urlValue) String() string {
	if *u.u == nil {
		return "<nil>"
	}
	return (*u.u).String()
}

// -- []*url.URL Value
type urlListValue []*url.URL

func newURLListValue(p *[]*url.URL) *urlListValue {
	return (*urlListValue)(p)
}

func (u *urlListValue) Set(value string) error {
	if url, err := url.Parse(value); err != nil {
		return fmt.Errorf("invalid URL: %s", err)
	} else {
		*u = append(*u, url)
		return nil
	}
}

func (u *urlListValue) String() string {
	out := []string{}
	for _, url := range *u {
		out = append(out, url.String())
	}
	return strings.Join(out, ",")
}

// A flag whose value must be in a set of options.
type enumValue struct {
	value   *string
	options []string
}

func newEnumFlag(target **string, options ...string) *enumValue {
	return &enumValue{
		value:   *target,
		options: options,
	}
}

func (a *enumValue) String() string {
	return *a.value
}

func (a *enumValue) Set(value string) error {
	for _, v := range a.options {
		if v == value {
			*a.value = value
			return nil
		}
	}
	return fmt.Errorf("must be one of %s, got '%s'", strings.Join(a.options, ","), value)
}

// -- units.Base2Bytes Value
type bytesValue units.Base2Bytes

func newBytesValue(val units.Base2Bytes, p *units.Base2Bytes) *bytesValue {
	*p = val
	return (*bytesValue)(p)
}

func (d *bytesValue) Set(s string) error {
	v, err := units.ParseBase2Bytes(s)
	*d = bytesValue(v)
	return err
}

func (d *bytesValue) Get() interface{} { return units.Base2Bytes(*d) }

func (d *bytesValue) String() string { return (*units.Base2Bytes)(d).String() }
